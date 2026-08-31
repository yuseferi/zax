package zaxhttp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuseferi/zax/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func newTestLogger() (*zap.Logger, *observer.ObservedLogs) {
	core, recorded := observer.New(zapcore.DebugLevel)
	return zap.New(core), recorded
}

func doRequest(t *testing.T, middleware func(http.Handler) http.Handler, handler http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(middleware(handler))
	t.Cleanup(server.Close)

	resp, err := http.Get(server.URL + "/hello")
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
}

func seenRequestID(t *testing.T, middleware func(http.Handler) http.Handler, req *http.Request) string {
	t.Helper()
	var seen string
	middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = zax.GetField(r.Context(), "request_id").String
	})).ServeHTTP(httptest.NewRecorder(), req)
	return seen
}

func TestMiddlewareUsesIncomingRequestID(t *testing.T) {
	logger, _ := newTestLogger()
	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	req.Header.Set("X-Request-ID", "abc-123")

	assert.Equal(t, "abc-123", seenRequestID(t, Middleware(logger), req))
}

func TestMiddlewareRespectsCustomHeader(t *testing.T) {
	logger, _ := newTestLogger()
	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	req.Header.Set("X-Trace-ID", "trace-9")

	middleware := Middleware(logger, WithRequestIDHeader("X-Trace-ID"))

	assert.Equal(t, "trace-9", seenRequestID(t, middleware, req))
}

func TestMiddlewareGeneratesRequestID(t *testing.T) {
	logger, _ := newTestLogger()
	middleware := Middleware(logger, WithRequestIDGenerator(func() string { return "gen-42" }))

	assert.Equal(t, "gen-42", seenRequestID(t, middleware, httptest.NewRequest(http.MethodGet, "/hello", nil)))
}

func TestDefaultRequestIDGenerator(t *testing.T) {
	assert.Len(t, defaultRequestID(), 32)
	assert.NotEqual(t, defaultRequestID(), defaultRequestID())
}

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

func TestMiddlewareLogsErrorFor5xx(t *testing.T) {
	logger, recorded := newTestLogger()
	doRequest(t, Middleware(logger), func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	completed := recorded.All()[1]
	assert.Equal(t, zapcore.ErrorLevel, completed.Level)
	assert.Contains(t, completed.Context, zap.Int("status", http.StatusInternalServerError))
}

func TestMiddlewareLogsWarnFor4xx(t *testing.T) {
	logger, recorded := newTestLogger()
	doRequest(t, Middleware(logger), func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	assert.Equal(t, zapcore.WarnLevel, recorded.All()[1].Level)
}

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

func TestStatusRecorderDefaultsTo200(t *testing.T) {
	rec := httptest.NewRecorder()
	logger, _ := newTestLogger()

	Middleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestStatusRecorderFlushPassthrough(t *testing.T) {
	recorder := newStatusRecorder(httptest.NewRecorder())

	flusher, ok := any(recorder).(http.Flusher)
	assert.True(t, ok)
	flusher.Flush() // must not panic even though httptest.ResponseRecorder flushes
}

func TestStatusRecorderRecordsExplicitStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	recorder := newStatusRecorder(rec)

	recorder.WriteHeader(http.StatusTeapot)

	assert.Equal(t, http.StatusTeapot, recorder.status())
	assert.Equal(t, http.StatusTeapot, rec.Code)
}

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
