// Package zax provides contextual field logging around the uber-zap logger.

package zax

import (
	"context"
	"go.uber.org/zap"
)

type Key string

// Key name which used for save logger in context
const loggerKey = Key("zax")

// Set Add passed fields to logger and store zap.Logger as variable in context
func Set(ctx context.Context, logger *zap.Logger, fields []zap.Field) context.Context {
	if len(fields) > 0 {
		logger = logger.With(fields...)
	}
	return context.WithValue(ctx, loggerKey, logger)
}

// Get zap.Logger from context
func Get(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value(loggerKey).(*zap.Logger); ok {
		return logger
	}
	return zap.L()
}
