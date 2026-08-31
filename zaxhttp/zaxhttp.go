// Package zaxhttp provides net/http middleware that populates zax fields
// for every request, enabling one-line adoption of context-aware logging.
package zaxhttp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"time"

	"github.com/yuseferi/zax/v2"
	"go.uber.org/zap"
)

const defaultRequestIDHeader = "X-Request-ID"

type config struct {
	requestIDHeader    string
	requestIDGenerator func() string
}

// Option configures the request middleware.
type Option func(*config)

// WithRequestIDHeader sets the header used to propagate a request ID.
// Defaults to "X-Request-ID".
func WithRequestIDHeader(header string) Option {
	return func(cfg *config) {
		if header != "" {
			cfg.requestIDHeader = header
		}
	}
}

// WithRequestIDGenerator sets the function used to create a request ID
// when the incoming request does not carry one.
// Defaults to a random 16-byte hex string.
func WithRequestIDGenerator(f func() string) Option {
	return func(cfg *config) {
		if f != nil {
			cfg.requestIDGenerator = f
		}
	}
}

// Middleware returns an http middleware that enriches the request context
// with zax fields (request_id, method, path, remote_addr) and logs
// request start and completion with those fields.
func Middleware(logger *zap.Logger, opts ...Option) func(http.Handler) http.Handler {
	cfg := newConfig(opts)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ctx := zax.Append(r.Context(), buildFields(r, cfg))
			r = r.WithContext(ctx)

			logger.Debug("request started", zax.Get(ctx)...)

			writer, recorder := newStatusRecorder(w)
			next.ServeHTTP(writer, r)

			logCompletion(logger, ctx, recorder.status(), time.Since(start))
		})
	}
}

// newConfig builds the middleware configuration from defaults and options.
func newConfig(opts []Option) *config {
	cfg := &config{
		requestIDHeader:    defaultRequestIDHeader,
		requestIDGenerator: defaultRequestID,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// buildFields assembles the zax fields describing an incoming request.
func buildFields(r *http.Request, cfg *config) []zap.Field {
	return []zap.Field{
		zap.String("request_id", requestID(r, cfg)),
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.String("remote_addr", r.RemoteAddr),
	}
}

// requestID returns the ID from the configured header, or generates one.
func requestID(r *http.Request, cfg *config) string {
	if id := r.Header.Get(cfg.requestIDHeader); id != "" {
		return id
	}
	return cfg.requestIDGenerator()
}

// defaultRequestID returns a random 16-byte hex string.
func defaultRequestID() string {
	b := make([]byte, 16)
	// crypto/rand.Read only errors if the entropy source is unavailable;
	// in that case the ID falls back to a deterministic zero string.
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// statusRecorder captures the effective response status for completion logging.
// Only the first written status counts, mirroring net/http behavior.
type statusRecorder struct {
	http.ResponseWriter
	code        int
	wroteHeader bool
}

// newStatusRecorder wraps w so the effective status can be read after the
// handler returns. The returned writer exposes the optional ResponseWriter
// interfaces (http.Flusher, http.Hijacker, http.Pusher, io.ReaderFrom) only
// when w itself supports them, so hijacking and streaming handlers keep working.
func newStatusRecorder(w http.ResponseWriter) (http.ResponseWriter, *statusRecorder) {
	recorder := &statusRecorder{ResponseWriter: w, code: http.StatusOK}

	_, isFlusher := w.(http.Flusher)
	_, isHijacker := w.(http.Hijacker)
	_, isPusher := w.(http.Pusher)
	_, isReaderFrom := w.(io.ReaderFrom)

	switch {
	case isFlusher && isHijacker && isReaderFrom: // standard HTTP/1 server writer
		return &flusherHijackerReaderFromWriter{recorder, w.(http.Flusher), w.(http.Hijacker), w.(io.ReaderFrom)}, recorder
	case isFlusher && isPusher: // standard HTTP/2 server writer
		return &flusherPusherWriter{recorder, w.(http.Flusher), w.(http.Pusher)}, recorder
	case isFlusher:
		return &flusherWriter{recorder, w.(http.Flusher)}, recorder
	default:
		return recorder, recorder
	}
}

// WriteHeader records the first status and delegates to the wrapped writer.
// Later calls are ignored, matching net/http behavior.
func (r *statusRecorder) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}
	r.code = code
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(code)
}

// Write records an implicit 200 if no status was written yet, then delegates.
func (r *statusRecorder) Write(p []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(p)
}

// status returns the effective response status recorded so far.
func (r *statusRecorder) status() int {
	return r.code
}

// flusherWriter adds http.Flusher support on top of the status recorder.
type flusherWriter struct {
	*statusRecorder
	http.Flusher
}

// flusherHijackerReaderFromWriter mirrors the standard net/http HTTP/1 writer.
type flusherHijackerReaderFromWriter struct {
	*statusRecorder
	http.Flusher
	http.Hijacker
	io.ReaderFrom
}

// flusherPusherWriter mirrors the standard net/http HTTP/2 writer.
type flusherPusherWriter struct {
	*statusRecorder
	http.Flusher
	http.Pusher
}

// logCompletion logs the request outcome, choosing the level from the status.
func logCompletion(logger *zap.Logger, ctx context.Context, code int, duration time.Duration) {
	fields := append(zax.Get(ctx), zap.Int("status", code), zap.Duration("duration", duration))

	switch {
	case code >= http.StatusInternalServerError:
		logger.Error("request completed", fields...)
	case code >= http.StatusBadRequest:
		logger.Warn("request completed", fields...)
	default:
		logger.Info("request completed", fields...)
	}
}
