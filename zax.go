// Package zax provides contextual field logging around the uber-zap logger.
package zax

import (
	"context"
	"slices"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Key string

// loggerKey is the key used to save fields in context.
const loggerKey = Key("zax")

// Set stores the passed fields in context, replacing any existing fields.
func Set(ctx context.Context, fields []zap.Field) context.Context {
	return context.WithValue(ctx, loggerKey, slices.Clone(fields))
}

// Append appends passed fields to the existing fields in context.
// It is recommended when you want to add fields without losing existing ones.
func Append(ctx context.Context, fields []zap.Field) context.Context {
	loggerFields := Get(ctx)
	if len(loggerFields) == 0 {
		return context.WithValue(ctx, loggerKey, slices.Clone(fields))
	}

	appended := make([]zap.Field, 0, len(loggerFields)+len(fields))
	appended = append(appended, loggerFields...)
	appended = append(appended, fields...)

	return context.WithValue(ctx, loggerKey, appended)
}

// Get returns the zap fields stored in context.
// The returned slice is a clone, so mutating it does not affect the context.
func Get(ctx context.Context) []zap.Field {
	if loggerFields, ok := ctx.Value(loggerKey).([]zap.Field); ok {
		return slices.Clone(loggerFields)
	}
	return nil
}

// GetField returns a specific zap field stored in context by key.
// It returns the zero-value zap.Field when the key is not present;
// use LookupField to distinguish absence from a zero-valued field.
func GetField(ctx context.Context, key string) zap.Field {
	field, _ := LookupField(ctx, key)
	return field
}

// LookupField returns a specific zap field stored in context by key,
// and whether the key was found.
func LookupField(ctx context.Context, key string) (zap.Field, bool) {
	loggerFields, ok := ctx.Value(loggerKey).([]zap.Field)
	if !ok {
		return zap.Field{}, false
	}
	for _, field := range loggerFields {
		if field.Key == key {
			return field, true
		}
	}
	return zap.Field{}, false
}

// Remove returns a context with the given keys removed from the stored fields.
// If none of the keys are present, the original context is returned unchanged.
func Remove(ctx context.Context, keys ...string) context.Context {
	loggerFields, ok := ctx.Value(loggerKey).([]zap.Field)
	if !ok || len(loggerFields) == 0 || len(keys) == 0 {
		return ctx
	}

	filtered := make([]zap.Field, 0, len(loggerFields))
	for _, field := range loggerFields {
		if !slices.Contains(keys, field.Key) {
			filtered = append(filtered, field)
		}
	}

	if len(filtered) == len(loggerFields) {
		return ctx
	}
	return context.WithValue(ctx, loggerKey, filtered)
}

// GetSugared converts zap.Fields stored in context to key-value pairs
// compatible with zap.SugaredLogger.With(...).
// It converts fields using Zap's encoder behavior to preserve values across field types.
func GetSugared(ctx context.Context) []any {
	fields := Get(ctx)
	var kv []any

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
