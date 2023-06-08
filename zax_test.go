package zax

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

type Logger struct {
	logger   *zap.Logger
	recorded *observer.ObservedLogs
	t        *testing.T
}

func NewLogger(t *testing.T) *Logger {
	core, recorded := observer.New(zapcore.DebugLevel)
	logger := &Logger{
		logger:   zap.New(core),
		recorded: recorded,
		t:        t,
	}
	return logger
}

func (l *Logger) GetZapLogger() *zap.Logger {
	return l.logger
}

func (l *Logger) GetRecordedLogs() []observer.LoggedEntry {
	return l.recorded.All()
}

func (l *Logger) AssertLogEntryExist(t assert.TestingT, key, value string) bool {
	for _, log := range l.recorded.All() {
		for _, r := range log.Context {
			if r.Key == key && r.String == value {
				return true
			}
		}
	}
	return assert.Fail(t, fmt.Sprintf("log entry does not exist with, %s = %s", key, value))
}

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

const traceIDKey = "trace_id"

func TestSet(t *testing.T) {
	testLog := NewLogger(t)
	testTraceID := "test-trace-id-3333"
	testTraceID2 := "test-trace-id-new"
	ctx := context.Background()
	tests := map[string]struct {
		context             context.Context
		expectedLoggerKey   string
		expectedLoggerValue string
	}{
		"context with trace-id": {
			context:             Set(ctx, testLog.logger, []zap.Field{zap.String(traceIDKey, testTraceID)}),
			expectedLoggerKey:   traceIDKey,
			expectedLoggerValue: testTraceID,
		},
		"context with trace-id with new value(to check it will be updated)": {
			context:             Set(ctx, testLog.logger, []zap.Field{zap.String(traceIDKey, testTraceID2)}),
			expectedLoggerKey:   traceIDKey,
			expectedLoggerValue: testTraceID2,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := tc.context
			logger := ctx.Value(loggerKey).(*zap.Logger)
			logger.Info("just a test record")
			assert.NotNil(t, logger)
			testLog.AssertLogEntryExist(t, tc.expectedLoggerKey, tc.expectedLoggerValue)
		})
	}
}

func TestGet(t *testing.T) {
	testLog := NewLogger(t)
	testTraceID := "test-trace-id-3333"
	traceIDKey := traceIDKey
	ctx := context.Background()
	tests := map[string]struct {
		context           context.Context
		expectedLoggerKey *string
	}{
		"context with trace-id": {
			context:           context.TODO(),
			expectedLoggerKey: nil,
		},
		"context with trace-id with new value(to check it will be updated)": {
			context:           Set(ctx, testLog.logger, []zap.Field{zap.String(traceIDKey, testTraceID)}),
			expectedLoggerKey: &traceIDKey,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := tc.context
			Get(ctx).Info("just a test record")
			if tc.expectedLoggerKey != nil {
				testLog.AssertLogEntryKeyExist(t, *tc.expectedLoggerKey)
			}
		})
	}
}
