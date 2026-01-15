package logutil

import (
	"context"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestWithLogger(t *testing.T) {
	core, _ := observer.New(zap.DebugLevel)
	logger := zap.New(core)

	ctx := context.Background()
	ctx = WithLogger(ctx, logger)

	retrieved := FromContext(ctx)
	if retrieved != logger {
		t.Error("Expected to retrieve the same logger instance")
	}
}

func TestFromContextNil(t *testing.T) {
	logger := FromContext(context.TODO())
	if logger == nil {
		t.Error("Expected non-nil logger")
	}
}

func TestFromContextEmpty(t *testing.T) {
	ctx := context.Background()
	logger := FromContext(ctx)
	if logger == nil {
		t.Error("Expected non-nil logger (should return no-op)")
	}
}

func TestFromContextOrDefault(t *testing.T) {
	core, _ := observer.New(zap.DebugLevel)
	defaultLogger := zap.New(core)

	ctx := context.Background()
	retrieved := FromContextOrDefault(ctx, defaultLogger)

	if retrieved != defaultLogger {
		t.Error("Expected to retrieve the default logger")
	}
}

func TestWithField(t *testing.T) {
	core, observed := observer.New(zap.DebugLevel)
	logger := zap.New(core)

	ctx := context.Background()
	ctx = WithLogger(ctx, logger)
	ctx = WithField(ctx, "request_id", "12345")

	Debug(ctx, "test message")

	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(entries))
	}

	found := false
	for _, field := range entries[0].Context {
		if field.Key == "request_id" && field.String == "12345" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find request_id field in log context")
	}
}

func TestDebugHelper(t *testing.T) {
	core, observed := observer.New(zap.DebugLevel)
	logger := zap.New(core)

	ctx := WithLogger(context.Background(), logger)
	Debug(ctx, "debug message", zap.String("key", "value"))

	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(entries))
	}

	if entries[0].Message != "debug message" {
		t.Errorf("Expected message 'debug message', got '%s'", entries[0].Message)
	}
}

func TestInfoHelper(t *testing.T) {
	core, observed := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	ctx := WithLogger(context.Background(), logger)
	Info(ctx, "info message")

	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(entries))
	}

	if entries[0].Level != zap.InfoLevel {
		t.Errorf("Expected INFO level, got %v", entries[0].Level)
	}
}
