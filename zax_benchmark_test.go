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

// LogWithZap logs directly with zap fields, as a baseline for the benchmark.
func LogWithZap(logger *zap.Logger) {
	logger.With(someFields...).Info("logging something")
}

// LogWithZax logs with fields carried in context via zax Set/Get.
func LogWithZax(logger *zap.Logger) {
	ctx := context.Background()
	ctx = Set(ctx, someFields)
	logger.With(Get(ctx)...).Info("logging something")
}

func BenchmarkLoggingWithOnlyZap(b *testing.B) {
	// Create a no-op logger that discards log output
	logger := zap.NewExample()
	logger = logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewNopCore()
	}))

	for b.Loop() {
		LogWithZap(logger)
	}
}

func BenchmarkLoggingWithZax(b *testing.B) {
	// Create a no-op logger that discards log output
	logger := zap.NewExample()
	logger = logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewNopCore()
	}))

	for b.Loop() {
		LogWithZax(logger)
	}
}
