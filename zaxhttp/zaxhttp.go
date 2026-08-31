// Package zaxhttp provides net/http middleware that populates zax fields
// for every request, enabling one-line adoption of context-aware logging.
package zaxhttp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

			recorder := newStatusRecorder(w)
			next.ServeHTTP(recorder, r)

			logCompletion(logger, ctx, recorder.status(), time.Since(start))
		})
	}
}

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

func buildFields(r *http.Request, cfg *config) []zap.Field {
	return []zap.Field{
		zap.String("request_id", requestID(r, cfg)),
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.String("remote_addr", r.RemoteAddr),
	}
}

func requestID(r *http.Request, cfg *config) string {
	if id := r.Header.Get(cfg.requestIDHeader); id != "" {
		return id
	}
	return cfg.requestIDGenerator()
}

func defaultRequestID() string {
	b := make([]byte, 16)
	// crypto/rand.Read only errors if the entropy source is unavailable;
	// in that case the ID falls back to a deterministic zero string.
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// statusRecorder captures the response status code for completion logging.
type statusRecorder struct {
	http.ResponseWriter
	code int
}

func newStatusRecorder(w http.ResponseWriter) *statusRecorder {
	return &statusRecorder{ResponseWriter: w, code: http.StatusOK}
}

func (r *statusRecorder) WriteHeader(code int) {
	r.code = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (r *statusRecorder) status() int {
	return r.code
}

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
