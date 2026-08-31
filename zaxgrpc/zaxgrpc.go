// Package zaxgrpc provides gRPC server interceptors that populate zax fields
// for every RPC, carrying request context into all downstream logs.
package zaxgrpc

import (
	"context"
	"time"

	"github.com/yuseferi/zax/v2"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var defaultRequestIDKeys = []string{"x-request-id", "request-id"}

type config struct {
	requestIDKeys []string
}

// Option configures the server interceptors.
type Option func(*config)

// WithRequestIDMetadataKeys sets the metadata keys checked for a request ID,
// in priority order. Defaults to "x-request-id" and "request-id".
func WithRequestIDMetadataKeys(keys ...string) Option {
	return func(cfg *config) {
		if len(keys) > 0 {
			cfg.requestIDKeys = keys
		}
	}
}

// UnaryServerInterceptor returns an interceptor that enriches the RPC context
// with zax fields and logs start and completion of each unary call.
func UnaryServerInterceptor(logger *zap.Logger, opts ...Option) grpc.UnaryServerInterceptor {
	cfg := newConfig(opts)
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		ctx = zax.Append(ctx, buildFields(ctx, info.FullMethod, cfg))

		logger.Debug("rpc started", zax.Get(ctx)...)

		resp, err := handler(ctx, req)
		logCompletion(logger, ctx, err, time.Since(start))
		return resp, err
	}
}

// StreamServerInterceptor returns an interceptor that enriches the RPC context
// with zax fields and logs start and completion of each streaming call.
func StreamServerInterceptor(logger *zap.Logger, opts ...Option) grpc.StreamServerInterceptor {
	cfg := newConfig(opts)
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		ctx := zax.Append(stream.Context(), buildFields(stream.Context(), info.FullMethod, cfg))

		logger.Debug("rpc started", zax.Get(ctx)...)

		err := handler(srv, &wrappedStream{ServerStream: stream, ctx: ctx})
		logCompletion(logger, ctx, err, time.Since(start))
		return err
	}
}

// newConfig builds the interceptor configuration from defaults and options.
func newConfig(opts []Option) *config {
	cfg := &config{requestIDKeys: defaultRequestIDKeys}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// wrappedStream overrides Context so handlers see the zax-enriched context.
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the zax-enriched context instead of the stream's original.
func (w *wrappedStream) Context() context.Context {
	return w.ctx
}

// buildFields assembles the zax fields describing an incoming RPC.
func buildFields(ctx context.Context, fullMethod string, cfg *config) []zap.Field {
	fields := []zap.Field{zap.String("grpc_method", fullMethod)}

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if id := firstValue(md, cfg.requestIDKeys); id != "" {
			fields = append(fields, zap.String("request_id", id))
		}
	}
	return fields
}

// firstValue returns the first non-empty value found for the given keys.
func firstValue(md metadata.MD, keys []string) string {
	for _, key := range keys {
		if values := md.Get(key); len(values) > 0 && values[0] != "" {
			return values[0]
		}
	}
	return ""
}

// logCompletion logs the RPC outcome, at Error level when the code is not OK.
func logCompletion(logger *zap.Logger, ctx context.Context, err error, duration time.Duration) {
	code := status.Code(err)
	fields := append(zax.Get(ctx),
		zap.String("grpc_code", code.String()),
		zap.Duration("duration", duration),
	)

	if code != codes.OK {
		logger.Error("rpc completed", fields...)
		return
	}
	logger.Info("rpc completed", fields...)
}
