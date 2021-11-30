package logging

import (
	"runtime"
	"runtime/debug"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type testLogger struct {
	name    string
	discard bool
	tb      testing.TB
	encoder zapcore.Encoder
	fields  []zap.Field
}

func CreateTestLogger(tb testing.TB, discard bool) Logger {
	return &testLogger{
		tb:      tb,
		discard: discard,
		encoder: zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig()),
	}
}

func (t testLogger) Named(s string) Logger {
	t.name = s
	return t
}

func (t testLogger) WouldLog(_ zapcore.Level) bool {
	return true
}

func (t testLogger) With(fields ...zap.Field) Logger {
	t.fields = append(t.fields, fields...)
	return t
}

func (t testLogger) Debug(msg string, fields ...zap.Field) {
	t.tb.Helper()
	buf, err := t.encoder.EncodeEntry(zapcore.Entry{
		Level:      zapcore.DebugLevel,
		Time:       time.Now().UTC(),
		LoggerName: t.name,
		Message:    msg,
		Caller:     zapcore.NewEntryCaller(runtime.Caller(2)),
	}, append(t.fields, fields...))
	if err == nil && !t.discard {
		t.tb.Log(buf.String())
	}
}

func (t testLogger) Info(msg string, fields ...zap.Field) {
	t.tb.Helper()
	buf, err := t.encoder.EncodeEntry(zapcore.Entry{
		Level:      zapcore.InfoLevel,
		Time:       time.Now().UTC(),
		LoggerName: t.name,
		Message:    msg,
		Caller:     zapcore.NewEntryCaller(runtime.Caller(2)),
	}, append(t.fields, fields...))
	if err == nil && !t.discard {
		t.tb.Log(buf.String())
	}
}

func (t testLogger) Warn(msg string, fields ...zap.Field) {
	t.tb.Helper()
	buf, err := t.encoder.EncodeEntry(zapcore.Entry{
		Level:      zapcore.WarnLevel,
		Time:       time.Now().UTC(),
		LoggerName: t.name,
		Message:    msg,
		Caller:     zapcore.NewEntryCaller(runtime.Caller(2)),
	}, append(t.fields, fields...))
	if err == nil && !t.discard {
		t.tb.Log(buf.String())
	}
}

func (t testLogger) Error(msg string, fields ...zap.Field) {
	t.tb.Helper()
	buf, err := t.encoder.EncodeEntry(zapcore.Entry{
		Level:      zapcore.ErrorLevel,
		Time:       time.Now().UTC(),
		LoggerName: t.name,
		Message:    msg,
		Caller:     zapcore.NewEntryCaller(runtime.Caller(2)),
		Stack:      string(debug.Stack()),
	}, append(t.fields, fields...))
	if err == nil && !t.discard {
		t.tb.Log(buf.String())
	}
}

func (t testLogger) Panic(msg string, fields ...zap.Field) {
	t.tb.Helper()
	buf, err := t.encoder.EncodeEntry(zapcore.Entry{
		Level:      zapcore.PanicLevel,
		Time:       time.Now().UTC(),
		LoggerName: t.name,
		Message:    msg,
		Caller:     zapcore.NewEntryCaller(runtime.Caller(2)),
		Stack:      string(debug.Stack()),
	}, append(t.fields, fields...))
	if err == nil && !t.discard {
		t.tb.Log(buf.String())
	}
}

func (t testLogger) Fatal(msg string, fields ...zap.Field) {
	t.tb.Helper()
	buf, err := t.encoder.EncodeEntry(zapcore.Entry{
		Level:      zapcore.FatalLevel,
		Time:       time.Now().UTC(),
		LoggerName: t.name,
		Message:    msg,
		Caller:     zapcore.NewEntryCaller(runtime.Caller(2)),
		Stack:      string(debug.Stack()),
	}, append(t.fields, fields...))
	if err == nil && !t.discard {
		t.tb.Log(buf.String())
	}
}

func (t testLogger) Sync() error {
	return nil
}
