package logutil

import (
	"context"

	"go.uber.org/zap"
)

type contextKey string

const loggerKey contextKey = "logger"

// WithLogger añade un logger al contexto
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext recupera el logger del contexto
// Si no hay logger, devuelve un logger no-op para evitar panics
func FromContext(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return zap.NewNop()
	}
	if logger, ok := ctx.Value(loggerKey).(*zap.Logger); ok {
		return logger
	}
	return zap.NewNop()
}

// FromContextOrDefault recupera el logger del contexto o devuelve el proporcionado como default
func FromContextOrDefault(ctx context.Context, defaultLogger *zap.Logger) *zap.Logger {
	if ctx == nil {
		return defaultLogger
	}
	if logger, ok := ctx.Value(loggerKey).(*zap.Logger); ok {
		return logger
	}
	return defaultLogger
}

// WithField añade un campo al logger en el contexto y devuelve un nuevo contexto
func WithField(ctx context.Context, key string, value any) context.Context {
	logger := FromContext(ctx)
	return WithLogger(ctx, logger.With(zap.Any(key, value)))
}

// WithFields añade múltiples campos al logger en el contexto y devuelve un nuevo contexto
func WithFields(ctx context.Context, fields ...zap.Field) context.Context {
	logger := FromContext(ctx)
	return WithLogger(ctx, logger.With(fields...))
}

// Debug log a debug message using the logger from context
func Debug(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).Debug(msg, fields...)
}

// Info log an info message using the logger from context
func Info(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).Info(msg, fields...)
}

// Warn log a warning message using the logger from context
func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).Warn(msg, fields...)
}

// Error log an error message using the logger from context
func Error(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).Error(msg, fields...)
}
