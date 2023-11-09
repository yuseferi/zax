package zax

import (
	"context"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var someFields = []zap.Field{
	zap.String("field1", "value1"),
	zap.String("field2", "value2"),
	zap.Int("field3", 2),
}

func LogWithZap(logger *zap.Logger) {
	logger.With(someFields...).Info("logging something")
}

func LogWithZax(logger *zap.Logger) {
	ctx := context.Background()
	Set(ctx, someFields)
	logger.With(Get(ctx)...).Info("logging something")
}

func BenchmarkLoggingWithOnlyZap(b *testing.B) {
	// Create a no-op logger that discards log output
	logger := zap.NewExample()
	logger = logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewNopCore()
	}))

	for i := 1; i <= b.N; i++ {
		LogWithZap(logger)
	}
}

func BenchmarkLoggingWithZax(b *testing.B) {
	// Create a no-op logger that discards log output
	logger := zap.NewExample()
	logger = logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewNopCore()
	}))

	for i := 1; i <= b.N; i++ {
		LogWithZax(logger)
	}
}
