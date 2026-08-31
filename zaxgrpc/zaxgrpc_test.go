package zaxgrpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuseferi/zax/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	unaryMethod  = "/svc/Echo"
	streamMethod = "/svc/Stream"
)

var unaryInfo = &grpc.UnaryServerInfo{FullMethod: unaryMethod}

// newTestLogger returns an observer-backed logger and its recorded logs.
func newTestLogger() (*zap.Logger, *observer.ObservedLogs) {
	core, recorded := observer.New(zapcore.DebugLevel)
	return zap.New(core), recorded
}

// incomingContext builds a context carrying incoming gRPC metadata pairs.
func incomingContext(pairs ...string) context.Context {
	ctx := context.Background()
	if len(pairs) > 0 {
		ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(pairs...))
	}
	return ctx
}

// callUnary invokes the interceptor with the shared unary method info.
func callUnary(interceptor grpc.UnaryServerInterceptor, ctx context.Context, handler grpc.UnaryHandler) (any, error) {
	return interceptor(ctx, nil, unaryInfo, handler)
}

// TestUnaryPropagatesRequestID verifies that the metadata request ID reaches
// the handler context and the completion log entry.
func TestUnaryPropagatesRequestID(t *testing.T) {
	logger, recorded := newTestLogger()
	var seen string

	_, err := callUnary(UnaryServerInterceptor(logger), incomingContext("x-request-id", "req-1"),
		func(ctx context.Context, _ any) (any, error) {
			seen = zax.GetField(ctx, "request_id").String
			return nil, nil
		})

	assert.NoError(t, err)
	assert.Equal(t, "req-1", seen)

	completed := recorded.All()[1]
	assert.Equal(t, "rpc completed", completed.Message)
	assert.Equal(t, zapcore.InfoLevel, completed.Level)
	assert.Contains(t, completed.Context, zap.String("request_id", "req-1"))
	assert.Contains(t, completed.Context, zap.String("grpc_method", unaryMethod))
	assert.Contains(t, completed.Context, zap.String("grpc_code", "OK"))
}

// TestUnaryFallsBackToSecondKey verifies that the second default metadata key
// is used when the first is absent.
func TestUnaryFallsBackToSecondKey(t *testing.T) {
	logger, _ := newTestLogger()
	var seen string

	_, err := callUnary(UnaryServerInterceptor(logger), incomingContext("request-id", "req-2"),
		func(ctx context.Context, _ any) (any, error) {
			seen = zax.GetField(ctx, "request_id").String
			return nil, nil
		})

	assert.NoError(t, err)
	assert.Equal(t, "req-2", seen)
}

// TestUnaryOmitsRequestIDWhenMissing verifies that no request_id field is set
// when the metadata carries none, while grpc_method still is.
func TestUnaryOmitsRequestIDWhenMissing(t *testing.T) {
	logger, recorded := newTestLogger()
	var found bool

	_, err := callUnary(UnaryServerInterceptor(logger), context.Background(),
		func(ctx context.Context, _ any) (any, error) {
			_, found = zax.LookupField(ctx, "request_id")
			return nil, nil
		})

	assert.NoError(t, err)
	assert.False(t, found)
	assert.Contains(t, recorded.All()[1].Context, zap.String("grpc_method", unaryMethod))
}

// TestUnaryIgnoresEmptyRequestIDValue verifies that an empty metadata value
// is treated as absent.
func TestUnaryIgnoresEmptyRequestIDValue(t *testing.T) {
	logger, _ := newTestLogger()
	var found bool

	_, err := callUnary(UnaryServerInterceptor(logger), incomingContext("x-request-id", ""),
		func(ctx context.Context, _ any) (any, error) {
			_, found = zax.LookupField(ctx, "request_id")
			return nil, nil
		})

	assert.NoError(t, err)
	assert.False(t, found)
}

// TestUnaryRespectsCustomMetadataKeys verifies that WithRequestIDMetadataKeys
// overrides the default keys.
func TestUnaryRespectsCustomMetadataKeys(t *testing.T) {
	logger, _ := newTestLogger()
	interceptor := UnaryServerInterceptor(logger, WithRequestIDMetadataKeys("x-trace-id"))
	var seen string

	_, err := callUnary(interceptor, incomingContext("x-trace-id", "t-1", "x-request-id", "ignored"),
		func(ctx context.Context, _ any) (any, error) {
			seen = zax.GetField(ctx, "request_id").String
			return nil, nil
		})

	assert.NoError(t, err)
	assert.Equal(t, "t-1", seen)
}

// TestUnaryLogsErrorOnFailure verifies that a failed RPC logs at Error level
// with the gRPC code.
func TestUnaryLogsErrorOnFailure(t *testing.T) {
	logger, recorded := newTestLogger()

	_, err := callUnary(UnaryServerInterceptor(logger), incomingContext(),
		func(context.Context, any) (any, error) {
			return nil, status.Error(codes.Internal, "boom")
		})

	assert.Error(t, err)
	completed := recorded.All()[1]
	assert.Equal(t, zapcore.ErrorLevel, completed.Level)
	assert.Contains(t, completed.Context, zap.String("grpc_code", "Internal"))
}

// TestUnaryPreservesExistingFields verifies that zax fields set upstream
// survive interceptor enrichment.
func TestUnaryPreservesExistingFields(t *testing.T) {
	logger, _ := newTestLogger()
	ctx := zax.Set(incomingContext(), []zap.Field{zap.String("tenant_id", "t-1")})
	var seen string

	_, err := callUnary(UnaryServerInterceptor(logger), ctx,
		func(ctx context.Context, _ any) (any, error) {
			seen = zax.GetField(ctx, "tenant_id").String
			return nil, nil
		})

	assert.NoError(t, err)
	assert.Equal(t, "t-1", seen)
}

// fakeStream is a minimal grpc.ServerStream exposing only Context.
type fakeStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the fake stream's context.
func (f *fakeStream) Context() context.Context {
	return f.ctx
}

// TestStreamExposesEnrichedContext verifies that the stream handler sees the
// enriched context and that completion is logged at Info level.
func TestStreamExposesEnrichedContext(t *testing.T) {
	logger, recorded := newTestLogger()
	info := &grpc.StreamServerInfo{FullMethod: streamMethod}
	stream := &fakeStream{ctx: incomingContext("x-request-id", "s-1")}
	var seen string

	err := StreamServerInterceptor(logger)(nil, stream, info, func(_ any, ss grpc.ServerStream) error {
		seen = zax.GetField(ss.Context(), "request_id").String
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "s-1", seen)

	completed := recorded.All()[1]
	assert.Equal(t, "rpc completed", completed.Message)
	assert.Equal(t, zapcore.InfoLevel, completed.Level)
	assert.Contains(t, completed.Context, zap.String("grpc_code", "OK"))
}

// TestStreamLogsErrorOnFailure verifies that a failed stream logs at Error
// level with the gRPC code.
func TestStreamLogsErrorOnFailure(t *testing.T) {
	logger, recorded := newTestLogger()
	info := &grpc.StreamServerInfo{FullMethod: streamMethod}
	stream := &fakeStream{ctx: incomingContext()}

	err := StreamServerInterceptor(logger)(nil, stream, info, func(any, grpc.ServerStream) error {
		return status.Error(codes.Internal, "stream failed")
	})

	assert.Error(t, err)
	completed := recorded.All()[1]
	assert.Equal(t, zapcore.ErrorLevel, completed.Level)
	assert.Contains(t, completed.Context, zap.String("grpc_code", "Internal"))
}

// TestStreamLogsStartAtDebugLevel verifies that the rpc started entry logs at
// Debug level.
func TestStreamLogsStartAtDebugLevel(t *testing.T) {
	logger, recorded := newTestLogger()
	info := &grpc.StreamServerInfo{FullMethod: streamMethod}
	stream := &fakeStream{ctx: incomingContext()}

	err := StreamServerInterceptor(logger)(nil, stream, info, func(any, grpc.ServerStream) error {
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "rpc started", recorded.All()[0].Message)
	assert.Equal(t, zapcore.DebugLevel, recorded.All()[0].Level)
}
