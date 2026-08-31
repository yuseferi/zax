package zax

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// Logger wraps an observer-backed zap logger for asserting on recorded log entries.
type Logger struct {
	logger   *zap.Logger
	recorded *observer.ObservedLogs
	t        *testing.T
}

// NewLogger creates a test logger that records all entries at DebugLevel and above.
func NewLogger(t *testing.T) *Logger {
	core, recorded := observer.New(zapcore.DebugLevel)
	logger := &Logger{
		logger:   zap.New(core),
		recorded: recorded,
		t:        t,
	}
	return logger
}

// GetZapLogger returns the underlying zap logger.
func (l *Logger) GetZapLogger() *zap.Logger {
	return l.logger
}

// GetRecordedLogs returns all log entries recorded so far.
func (l *Logger) GetRecordedLogs() []observer.LoggedEntry {
	return l.recorded.All()
}

// AssertLogEntryExist fails the test unless a recorded field with the given key and string value exists.
func (l *Logger) AssertLogEntryExist(t assert.TestingT, key, value string) bool {
	for _, log := range l.recorded.All() {
		for _, r := range log.Context {
			if r.Key == key && r.String == value {
				return true
			}
		}
	}
	if key == "" && value == "" {
		return true
	}
	return assert.Fail(t, fmt.Sprintf("log entry does not exist with, %s = %s", key, value))
}

// AssertLogEntryKeyExist fails the test unless a recorded field with the given key exists.
func (l *Logger) AssertLogEntryKeyExist(t assert.TestingT, key string) bool {
	for _, log := range l.recorded.All() {
		for _, r := range log.Context {
			if r.Key == key {
				return true
			}
		}
	}
	return assert.Fail(t, fmt.Sprintf("log entry does not exist with key = %s ", key))
}

const (
	traceIDKey  = "trace_id"
	spanIDKey   = "span_id"
	testTraceID = "test-trace-id-3333"
)

func TestSet(t *testing.T) {
	testLog := NewLogger(t)

	testTraceID2 := "test-trace-id-new"
	ctx := context.Background()
	tests := map[string]struct {
		context             context.Context
		expectedLoggerKey   string
		expectedLoggerValue string
	}{
		"context for zax filed is empty": {
			context:             Set(ctx, nil),
			expectedLoggerKey:   "",
			expectedLoggerValue: "",
		},
		"context with trace-id": {
			context:             Set(ctx, []zap.Field{zap.String(traceIDKey, testTraceID)}),
			expectedLoggerKey:   traceIDKey,
			expectedLoggerValue: testTraceID,
		},
		"context with trace-id with new value(to check it will be updated)": {
			context:             Set(ctx, []zap.Field{zap.String(traceIDKey, testTraceID2)}),
			expectedLoggerKey:   traceIDKey,
			expectedLoggerValue: testTraceID2,
		},
	}

	for name, tc := range tests {
		t.Run(
			name, func(t *testing.T) {
				ctx := tc.context
				logger := testLog.logger.With(Get(ctx)...)
				logger.Info("just a test record")
				assert.NotNil(t, logger)
				testLog.AssertLogEntryExist(t, tc.expectedLoggerKey, tc.expectedLoggerValue)
			},
		)
	}
}

func TestAppend(t *testing.T) {
	testLog := NewLogger(t)
	ctx := context.Background()
	ctx = Set(ctx, []zap.Field{zap.String(traceIDKey, testTraceID)})
	tests := map[string]struct {
		context             context.Context
		expectedFieldNumber int
	}{
		"context for zax filed is empty": {
			context:             Append(ctx, nil),
			expectedFieldNumber: 1,
		},
		"context with appending span-id": {
			context:             Append(ctx, []zap.Field{zap.String(spanIDKey, testTraceID)}),
			expectedFieldNumber: 2,
		},
	}

	for name, tc := range tests {
		t.Run(
			name, func(t *testing.T) {
				ctx := tc.context
				logger := testLog.logger.With(Get(ctx)...)
				logger.Info("just a test record")
				assert.NotNil(t, logger)
				assert.Equal(t, tc.expectedFieldNumber, len(Get(ctx)))

			},
		)
	}
}

func TestAppendPreservesExistingOrder(t *testing.T) {
	ctx := context.Background()
	ctx = Set(ctx, []zap.Field{zap.String(traceIDKey, testTraceID)})
	ctx = Append(ctx, []zap.Field{zap.String(spanIDKey, "new-span-id")})

	fields := Get(ctx)
	assert.Len(t, fields, 2)
	assert.Equal(t, traceIDKey, fields[0].Key)
	assert.Equal(t, testTraceID, fields[0].String)
	assert.Equal(t, spanIDKey, fields[1].Key)
	assert.Equal(t, "new-span-id", fields[1].String)
}

func TestAppendOnEmptyContext(t *testing.T) {
	ctx := Append(context.Background(), []zap.Field{zap.String(spanIDKey, "first-span-id")})

	fields := Get(ctx)
	assert.Len(t, fields, 1)
	assert.Equal(t, spanIDKey, fields[0].Key)
	assert.Equal(t, "first-span-id", fields[0].String)
}

func TestGet(t *testing.T) {
	testLog := NewLogger(t)
	traceIDKey := traceIDKey
	ctx := context.Background()
	tests := map[string]struct {
		context           context.Context
		expectedLoggerKey *string
	}{
		"context empty": {
			context:           context.TODO(),
			expectedLoggerKey: nil,
		},
		"context with trace-id field": {
			context:           Set(ctx, []zap.Field{zap.String(traceIDKey, testTraceID)}),
			expectedLoggerKey: &traceIDKey,
		},
	}

	for name, tc := range tests {
		t.Run(
			name, func(t *testing.T) {
				ctx := tc.context
				testLog.logger.With(Get(ctx)...).Info("just a test record")
				if tc.expectedLoggerKey != nil {
					testLog.AssertLogEntryKeyExist(t, *tc.expectedLoggerKey)
				}
			},
		)
	}
}

func TestGetReturnsClonedFields(t *testing.T) {
	ctx := Set(context.Background(), []zap.Field{zap.String(traceIDKey, testTraceID)})

	fields := Get(ctx)
	fields[0] = zap.String(traceIDKey, "mutated")

	original := Get(ctx)
	assert.Equal(t, testTraceID, original[0].String)
}

func TestGetSugared(t *testing.T) {
	testLog := NewLogger(t)
	sugar := testLog.logger.Sugar()

	traceIDKey := traceIDKey
	ctx := context.Background()
	tests := map[string]struct {
		context           context.Context
		expectedLoggerKey *string
	}{
		"context empty": {
			context:           context.TODO(),
			expectedLoggerKey: nil,
		},
		"context with trace-id field": {
			context:           Set(ctx, []zap.Field{zap.String(traceIDKey, testTraceID)}),
			expectedLoggerKey: &traceIDKey,
		},
	}

	for name, tc := range tests {
		t.Run(
			name, func(t *testing.T) {
				ctx := tc.context
				sugar.With(GetSugared(ctx)...).Errorf("just a test record")
				if tc.expectedLoggerKey != nil {
					testLog.AssertLogEntryKeyExist(t, *tc.expectedLoggerKey)
					testLog.AssertLogEntryExist(t, *tc.expectedLoggerKey, testTraceID)
				}
			},
		)
	}
}

func TestGetSugaredSupportsCommonTypes(t *testing.T) {
	testErr := errors.New("boom")
	ctx := Set(context.Background(), []zap.Field{
		zap.String("trace_id", testTraceID),
		zap.Bool("sampled", true),
		zap.Int("attempt", 2),
		zap.Error(testErr),
	})

	values := GetSugared(ctx)
	assert.Equal(t, []any{
		"trace_id", testTraceID,
		"sampled", true,
		"attempt", int64(2),
		"error", testErr,
	}, values)
}

func TestSetClonesInputFields(t *testing.T) {
	fields := []zap.Field{zap.String(traceIDKey, testTraceID)}
	ctx := Set(context.Background(), fields)

	fields[0] = zap.String(traceIDKey, "mutated")

	stored := Get(ctx)
	assert.Equal(t, testTraceID, stored[0].String)
}

func TestGetField(t *testing.T) {
	traceIDKey := traceIDKey
	ctx := context.Background()
	tests := map[string]struct {
		context       context.Context
		expectedValue string
	}{
		"context empty": {
			context:       context.TODO(),
			expectedValue: "",
		},
		"context with trace-id field": {
			context:       Set(ctx, []zap.Field{zap.String(traceIDKey, testTraceID)}),
			expectedValue: testTraceID,
		},
	}

	for name, tc := range tests {
		t.Run(
			name, func(t *testing.T) {
				ctx := tc.context
				field := GetField(ctx, traceIDKey)
				assert.Equal(t, tc.expectedValue, field.String)
			},
		)
	}
}

func TestLookupField(t *testing.T) {
	ctx := Set(context.Background(), []zap.Field{
		zap.String(traceIDKey, testTraceID),
		zap.Int("attempt", 0),
	})

	field, ok := LookupField(ctx, traceIDKey)
	assert.True(t, ok)
	assert.Equal(t, testTraceID, field.String)

	// A zero-valued field that exists must still report found=true,
	// which GetField alone cannot express.
	field, ok = LookupField(ctx, "attempt")
	assert.True(t, ok)
	assert.Equal(t, int64(0), field.Integer)

	_, ok = LookupField(ctx, "missing")
	assert.False(t, ok)

	_, ok = LookupField(context.Background(), traceIDKey)
	assert.False(t, ok)
}

func TestRemove(t *testing.T) {
	ctx := Set(context.Background(), []zap.Field{
		zap.String(traceIDKey, testTraceID),
		zap.String(spanIDKey, "span-1"),
		zap.String("user_email", "user@example.com"),
	})

	// Remove a single key, e.g. PII before passing the context on.
	ctx = Remove(ctx, "user_email")
	fields := Get(ctx)
	assert.Len(t, fields, 2)
	_, found := LookupField(ctx, "user_email")
	assert.False(t, found)

	// Remove multiple keys at once.
	ctx = Remove(ctx, spanIDKey, traceIDKey)
	assert.Empty(t, Get(ctx))
}

func TestRemoveReturnsSameContextWhenKeysAbsent(t *testing.T) {
	ctx := Set(context.Background(), []zap.Field{zap.String(traceIDKey, testTraceID)})

	assert.True(t, ctx == Remove(ctx, "missing"))
	assert.True(t, ctx == Remove(ctx))

	emptyCtx := context.Background()
	assert.True(t, emptyCtx == Remove(emptyCtx, traceIDKey))
}
