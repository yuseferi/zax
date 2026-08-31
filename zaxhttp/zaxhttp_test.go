package zaxhttp

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuseferi/zax/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// newTestLogger returns an observer-backed logger and its recorded logs.
func newTestLogger() (*zap.Logger, *observer.ObservedLogs) {
	core, recorded := observer.New(zapcore.DebugLevel)
	return zap.New(core), recorded
}

// doRequest sends a GET /hello request through the given middleware and handler.
func doRequest(t *testing.T, middleware func(http.Handler) http.Handler, handler http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(middleware(handler))
	t.Cleanup(server.Close)

	resp, err := http.Get(server.URL + "/hello")
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
}

// seenRequestID runs one request through the middleware and returns the
// request_id field visible to the handler.
func seenRequestID(t *testing.T, middleware func(http.Handler) http.Handler, req *http.Request) string {
	t.Helper()
	var seen string
	middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = zax.GetField(r.Context(), "request_id").String
	})).ServeHTTP(httptest.NewRecorder(), req)
	return seen
}

// TestMiddlewareUsesIncomingRequestID verifies that an incoming X-Request-ID
// header is propagated to the handler context.
func TestMiddlewareUsesIncomingRequestID(t *testing.T) {
	logger, _ := newTestLogger()
	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	req.Header.Set("X-Request-ID", "abc-123")

	assert.Equal(t, "abc-123", seenRequestID(t, Middleware(logger), req))
}

// TestMiddlewareRespectsCustomHeader verifies that WithRequestIDHeader
// changes which header carries the request ID.
func TestMiddlewareRespectsCustomHeader(t *testing.T) {
	logger, _ := newTestLogger()
	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	req.Header.Set("X-Trace-ID", "trace-9")

	middleware := Middleware(logger, WithRequestIDHeader("X-Trace-ID"))

	assert.Equal(t, "trace-9", seenRequestID(t, middleware, req))
}

// TestMiddlewareGeneratesRequestID verifies that the configured generator is
// used when the request carries no ID header.
func TestMiddlewareGeneratesRequestID(t *testing.T) {
	logger, _ := newTestLogger()
	middleware := Middleware(logger, WithRequestIDGenerator(func() string { return "gen-42" }))

	assert.Equal(t, "gen-42", seenRequestID(t, middleware, httptest.NewRequest(http.MethodGet, "/hello", nil)))
}

// TestDefaultRequestIDGenerator verifies that default IDs are 32 hex chars
// and unique across calls.
func TestDefaultRequestIDGenerator(t *testing.T) {
	assert.Len(t, defaultRequestID(), 32)
	assert.NotEqual(t, defaultRequestID(), defaultRequestID())
}

// TestMiddlewareLogsStartAndCompletion verifies the start and completion log
// entries, their levels, and the fields they carry.
func TestMiddlewareLogsStartAndCompletion(t *testing.T) {
	logger, recorded := newTestLogger()
	doRequest(t, Middleware(logger), func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	entries := recorded.All()
	assert.Len(t, entries, 2)
	assert.Equal(t, "request started", entries[0].Message)
	assert.Equal(t, zapcore.DebugLevel, entries[0].Level)
	assert.Equal(t, "request completed", entries[1].Message)
	assert.Equal(t, zapcore.InfoLevel, entries[1].Level)

	fields := entries[1].Context
	assert.Contains(t, fields, zap.String("method", http.MethodGet))
	assert.Contains(t, fields, zap.String("path", "/hello"))
	assert.Contains(t, fields, zap.Int("status", http.StatusOK))
	assert.True(t, hasKeys(fields, "duration", "request_id"))
}

// TestMiddlewareLogsErrorFor5xx verifies that 5xx responses log at Error level.
func TestMiddlewareLogsErrorFor5xx(t *testing.T) {
	logger, recorded := newTestLogger()
	doRequest(t, Middleware(logger), func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	completed := recorded.All()[1]
	assert.Equal(t, zapcore.ErrorLevel, completed.Level)
	assert.Contains(t, completed.Context, zap.Int("status", http.StatusInternalServerError))
}

// TestMiddlewareLogsWarnFor4xx verifies that 4xx responses log at Warn level.
func TestMiddlewareLogsWarnFor4xx(t *testing.T) {
	logger, recorded := newTestLogger()
	doRequest(t, Middleware(logger), func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	assert.Equal(t, zapcore.WarnLevel, recorded.All()[1].Level)
}

// TestMiddlewarePreservesExistingFields verifies that zax fields set upstream
// survive middleware enrichment.
func TestMiddlewarePreservesExistingFields(t *testing.T) {
	logger, _ := newTestLogger()
	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	req = req.WithContext(zax.Set(req.Context(), []zap.Field{zap.String("tenant_id", "t-1")}))
	var seen string

	Middleware(logger)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = zax.GetField(r.Context(), "tenant_id").String
	})).ServeHTTP(httptest.NewRecorder(), req)

	assert.Equal(t, "t-1", seen)
}

// TestStatusRecorderDefaultsTo200 verifies the implicit 200 when the handler
// writes a body without an explicit status.
func TestStatusRecorderDefaultsTo200(t *testing.T) {
	rec := httptest.NewRecorder()
	logger, _ := newTestLogger()

	Middleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestStatusRecorderRecordsExplicitStatus verifies that an explicit
// WriteHeader call is recorded.
func TestStatusRecorderRecordsExplicitStatus(t *testing.T) {
	writer, recorder := newStatusRecorder(httptest.NewRecorder())

	writer.WriteHeader(http.StatusTeapot)

	assert.Equal(t, http.StatusTeapot, recorder.status())
}

// TestStatusRecorderKeepsFirstStatus verifies that only the first written
// status counts, mirroring net/http behavior.
func TestStatusRecorderKeepsFirstStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	writer, recorder := newStatusRecorder(rec)

	writer.WriteHeader(http.StatusTeapot)
	writer.WriteHeader(http.StatusInternalServerError)

	assert.Equal(t, http.StatusTeapot, recorder.status())
	assert.Equal(t, http.StatusTeapot, rec.Code)
}

// TestStatusRecorderRecordsImplicit200OnWrite verifies that Write without a
// prior WriteHeader records an implicit 200.
func TestStatusRecorderRecordsImplicit200OnWrite(t *testing.T) {
	writer, recorder := newStatusRecorder(httptest.NewRecorder())

	n, err := writer.Write([]byte("ok"))

	assert.NoError(t, err)
	assert.Len(t, "ok", n)
	assert.Equal(t, http.StatusOK, recorder.status())
}

// TestStatusRecorderExposesSupportedInterfaces verifies that the wrapper
// exposes exactly the interfaces the wrapped writer supports.
func TestStatusRecorderExposesSupportedInterfaces(t *testing.T) {
	// httptest.ResponseRecorder implements http.Flusher only.
	writer, _ := newStatusRecorder(httptest.NewRecorder())

	assert.Implements(t, (*http.Flusher)(nil), writer)
	assert.NotImplements(t, (*http.Hijacker)(nil), writer)
	assert.NotImplements(t, (*http.Pusher)(nil), writer)
	assert.NotImplements(t, (*io.ReaderFrom)(nil), writer)
}

// TestStatusRecorderHidesInterfacesWhenUnsupported verifies that no optional
// interface is advertised when the wrapped writer supports none.
func TestStatusRecorderHidesInterfacesWhenUnsupported(t *testing.T) {
	writer, _ := newStatusRecorder(&bareWriter{})

	assert.NotImplements(t, (*http.Flusher)(nil), writer)
	assert.NotImplements(t, (*http.Hijacker)(nil), writer)
	assert.NotImplements(t, (*http.Pusher)(nil), writer)
	assert.NotImplements(t, (*io.ReaderFrom)(nil), writer)
}

// TestStatusRecorderExposesHTTP1Capabilities verifies Flusher, Hijacker, and
// ReaderFrom are exposed for a standard HTTP/1 writer.
func TestStatusRecorderExposesHTTP1Capabilities(t *testing.T) {
	writer, _ := newStatusRecorder(&http1Writer{})

	assert.Implements(t, (*http.Flusher)(nil), writer)
	assert.Implements(t, (*http.Hijacker)(nil), writer)
	assert.Implements(t, (*io.ReaderFrom)(nil), writer)
	assert.NotImplements(t, (*http.Pusher)(nil), writer)
}

// TestStatusRecorderExposesHTTP2Capabilities verifies Flusher and Pusher are
// exposed for a standard HTTP/2 writer.
func TestStatusRecorderExposesHTTP2Capabilities(t *testing.T) {
	writer, _ := newStatusRecorder(&http2Writer{})

	assert.Implements(t, (*http.Flusher)(nil), writer)
	assert.Implements(t, (*http.Pusher)(nil), writer)
	assert.NotImplements(t, (*http.Hijacker)(nil), writer)
	assert.NotImplements(t, (*io.ReaderFrom)(nil), writer)
}

// TestStatusRecorderFlushPassthrough verifies Flush reaches the wrapped writer.
func TestStatusRecorderFlushPassthrough(t *testing.T) {
	writer, _ := newStatusRecorder(httptest.NewRecorder())

	flusher, ok := writer.(http.Flusher)
	assert.True(t, ok)
	flusher.Flush() // must not panic
}

// bareWriter supports none of the optional ResponseWriter interfaces.
type bareWriter struct {
	http.ResponseWriter
}

// http1Writer mimics the standard net/http HTTP/1 server writer.
type http1Writer struct {
	http.ResponseWriter
}

// Flush is a no-op so http1Writer implements http.Flusher.
func (f *http1Writer) Flush() {}

// Hijack is a no-op so http1Writer implements http.Hijacker.
func (f *http1Writer) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, nil
}

// ReadFrom is a no-op so http1Writer implements io.ReaderFrom.
func (f *http1Writer) ReadFrom(io.Reader) (int64, error) {
	return 0, nil
}

// http2Writer mimics the standard net/http HTTP/2 server writer.
type http2Writer struct {
	http.ResponseWriter
}

// Flush is a no-op so http2Writer implements http.Flusher.
func (f *http2Writer) Flush() {}

// Push is a no-op so http2Writer implements http.Pusher.
func (f *http2Writer) Push(string, *http.PushOptions) error {
	return nil
}

// hasKeys reports whether all given keys appear in the field list.
func hasKeys(fields []zap.Field, keys ...string) bool {
	for _, key := range keys {
		found := false
		for _, f := range fields {
			if f.Key == key {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
