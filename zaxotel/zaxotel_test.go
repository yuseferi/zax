package zaxotel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuseferi/zax/v2"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

const (
	validTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	validSpanID  = "00f067aa0ba902b7"
)

func validSpanContext() trace.SpanContext {
	traceID, _ := trace.TraceIDFromHex(validTraceID)
	spanID, _ := trace.SpanIDFromHex(validSpanID)
	return trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
}

func TestFields(t *testing.T) {
	t.Run("valid span context returns trace fields", func(t *testing.T) {
		ctx := trace.ContextWithSpanContext(context.Background(), validSpanContext())

		fields := Fields(ctx)

		assert.ElementsMatch(t, []zap.Field{
			zap.String("trace_id", validTraceID),
			zap.String("span_id", validSpanID),
			zap.Bool("sampled", true),
		}, fields)
	})

	t.Run("context without span returns nil", func(t *testing.T) {
		assert.Nil(t, Fields(context.Background()))
	})

	t.Run("invalid span context returns nil", func(t *testing.T) {
		ctx := trace.ContextWithSpanContext(context.Background(), trace.SpanContext{})

		assert.Nil(t, Fields(ctx))
	})
}

func TestWithTrace(t *testing.T) {
	t.Run("appends trace fields to zax context", func(t *testing.T) {
		ctx := trace.ContextWithSpanContext(context.Background(), validSpanContext())
		ctx = zax.Set(ctx, []zap.Field{zap.String("user_id", "u1")})

		ctx = WithTrace(ctx)

		fields := zax.Get(ctx)
		assert.Len(t, fields, 4)
		assert.Equal(t, "u1", zax.GetField(ctx, "user_id").String)
		assert.Equal(t, validTraceID, zax.GetField(ctx, "trace_id").String)
	})

	t.Run("returns original context when no span present", func(t *testing.T) {
		ctx := context.Background()

		assert.True(t, ctx == WithTrace(ctx), "context should be returned unchanged")
	})

	t.Run("trace fields reach the logger", func(t *testing.T) {
		core, recorded := observer.New(zapcore.DebugLevel)
		logger := zap.New(core)
		ctx := trace.ContextWithSpanContext(context.Background(), validSpanContext())

		logger.Info("handled", zax.Get(WithTrace(ctx))...)

		entry := recorded.All()[0]
		assert.Contains(t, entry.Context, zap.String("trace_id", validTraceID))
		assert.Contains(t, entry.Context, zap.String("span_id", validSpanID))
	})
}
