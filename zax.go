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
	return context.WithValue(ctx, loggerKey, cloneFields(fields))
}

// Append appends passed fields to the existing fields in context.
// It is recommended when you want to add fields without losing existing ones.
func Append(ctx context.Context, fields []zap.Field) context.Context {
	loggerFields := Get(ctx)
	if len(loggerFields) == 0 {
		return context.WithValue(ctx, loggerKey, cloneFields(fields))
	}

	appended := make([]zap.Field, 0, len(loggerFields)+len(fields))
	appended = append(appended, loggerFields...)
	appended = append(appended, fields...)

	return context.WithValue(ctx, loggerKey, appended)
}

// Get zap stored fields from context
func Get(ctx context.Context) []zap.Field {
	if loggerFields, ok := ctx.Value(loggerKey).([]zap.Field); ok {
		return cloneFields(loggerFields)
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
// It converts fields using Zap's encoder behavior to preserve values across field types.
func GetSugared(ctx context.Context) []interface{} {
	fields := Get(ctx)
	var kv []interface{}

	for _, f := range fields {
		if f.Type == zapcore.ErrorType {
			if err, ok := f.Interface.(error); ok {
				kv = append(kv, f.Key, err)
				continue
			}
		}

		encoder := zapcore.NewMapObjectEncoder()
		f.AddTo(encoder)
		if value, ok := encoder.Fields[f.Key]; ok {
			kv = append(kv, f.Key, value)
		}
	}
	return kv
}

func cloneFields(fields []zap.Field) []zap.Field {
	if len(fields) == 0 {
		return nil
	}

	cloned := make([]zap.Field, len(fields))
	copy(cloned, fields)

	return cloned
}
