// Package zaxotel bridges OpenTelemetry trace context with zax,
// exposing trace and span IDs as zap fields for log correlation.
package zaxotel

import (
	"context"

	"github.com/yuseferi/zax/v2"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Fields returns zap fields describing the trace and span IDs found in ctx.
// It returns nil when the context carries no valid OpenTelemetry span context.
func Fields(ctx context.Context) []zap.Field {
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return nil
	}

	return []zap.Field{
		zap.String("trace_id", spanContext.TraceID().String()),
		zap.String("span_id", spanContext.SpanID().String()),
		zap.Bool("sampled", spanContext.IsSampled()),
	}
}

// WithTrace appends the trace and span ID fields from ctx to the zax fields
// already stored in it, so subsequent logs carry trace correlation data.
// It returns the original context unchanged when no valid span context exists.
func WithTrace(ctx context.Context) context.Context {
	fields := Fields(ctx)
	if fields == nil {
		return ctx
	}
	return zax.Append(ctx, fields)
}
