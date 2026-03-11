package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
)

type contextKey string

const (
	RequestIDKey contextKey = "request_id"
	UserIDKey    contextKey = "user_id"
)

var defaultLogger *slog.Logger

// Init initializes the default logger.
func Init(env string) {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		AddSource: env != "production",
	}

	if env == "production" {
		opts.Level = slog.LevelInfo
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		opts.Level = slog.LevelDebug
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
}

// InitWithWriter initializes the logger with a custom writer (useful for testing).
func InitWithWriter(w io.Writer, env string) {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		AddSource: false,
	}

	if env == "production" {
		opts.Level = slog.LevelInfo
		handler = slog.NewJSONHandler(w, opts)
	} else {
		opts.Level = slog.LevelDebug
		handler = slog.NewTextHandler(w, opts)
	}

	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
}

// Default returns the default logger.
func Default() *slog.Logger {
	if defaultLogger == nil {
		Init("development")
	}
	return defaultLogger
}

// WithContext returns a logger with context values (request_id, user_id).
func WithContext(ctx context.Context) *slog.Logger {
	logger := Default()

	if requestID, ok := ctx.Value(RequestIDKey).(string); ok && requestID != "" {
		logger = logger.With(slog.String("request_id", requestID))
	}

	if userID, ok := ctx.Value(UserIDKey).(string); ok && userID != "" {
		logger = logger.With(slog.String("user_id", userID))
	}

	return logger
}

// Info logs at info level.
func Info(msg string, args ...any) {
	Default().Info(msg, args...)
}

// Error logs at error level.
func Error(msg string, args ...any) {
	Default().Error(msg, args...)
}

// Debug logs at debug level.
func Debug(msg string, args ...any) {
	Default().Debug(msg, args...)
}

// Warn logs at warn level.
func Warn(msg string, args ...any) {
	Default().Warn(msg, args...)
}

// InfoContext logs at info level with context.
func InfoContext(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Info(msg, args...)
}

// ErrorContext logs at error level with context.
func ErrorContext(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Error(msg, args...)
}

// DebugContext logs at debug level with context.
func DebugContext(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Debug(msg, args...)
}

// WarnContext logs at warn level with context.
func WarnContext(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Warn(msg, args...)
}
