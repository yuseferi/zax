// Package zax provides contextual field logging around the uber-zap logger.

package zax

import (
	"context"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Key string

// Key name which used for save fields in context
const loggerKey = Key("zax")

// Set Add passed fields in context
func Set(ctx context.Context, fields []zap.Field) context.Context {
	return context.WithValue(ctx, loggerKey, fields)
}

// Append  appending passed fields to the existing fields in context.
// it's recommended to use Append when you want to append some fields and do not lose the already added fields to context.
func Append(ctx context.Context, fields []zap.Field) context.Context {
	if loggerFields, ok := ctx.Value(loggerKey).([]zap.Field); ok {
		fields = append(fields, loggerFields...)
	}
	return context.WithValue(ctx, loggerKey, fields)
}

// Get zap stored fields from context
func Get(ctx context.Context) []zap.Field {
	if loggerFields, ok := ctx.Value(loggerKey).([]zap.Field); ok {
		return loggerFields
	}
	return nil
}

// GetField Get a specific zap stored field from context by key
func GetField(ctx context.Context, key string) (field zap.Field) {
	if loggerFields, ok := ctx.Value(loggerKey).([]zap.Field); ok {
		for _, field := range loggerFields {
			if field.Key == key {
				return field
			}
		}
	}
	return
}

// GetSugared converts zap.Fields stored in context to key-value pairs
// compatible with zap.SugaredLogger.With(...).
// beat in Mind in Sugar version it's limited to String, Bool, Int, Error and Interface.
func GetSugared(ctx context.Context) []interface{} {
	fields := Get(ctx)
	var kv []interface{}

	for _, f := range fields {
		switch f.Type {
		case zapcore.StringType:
			kv = append(kv, f.Key, f.String)
		case zapcore.BoolType:
			kv = append(kv, f.Key, f.Integer == 1)
		case zapcore.Int64Type, zapcore.Uint64Type:
			kv = append(kv, f.Key, f.Integer)
		case zapcore.ErrorType:
			if err, ok := f.Interface.(error); ok {
				kv = append(kv, f.Key, err)
			}
		default:
			// fallback to Interface, if it exists
			if f.Interface != nil {
				kv = append(kv, f.Key, f.Interface)
			}
		}
	}
	return kv
}
