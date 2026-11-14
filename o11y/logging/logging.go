package logging

import (
	"context"
	"log/slog"
	"os"
)

// Log is the shared logger configured by InitLogger.
var Log *Logger

// Logger is the structured logger used throughout the application.
type Logger = slog.Logger

type loggerCtxKey struct{}

// InitLogger configures the shared logger to write debug-level JSON to standard output.
func InitLogger() *Logger {
	Log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	return Log
}

// WithLogger returns a context carrying log.
func WithLogger(ctx context.Context, log *Logger) context.Context {
	return context.WithValue(ctx, loggerCtxKey{}, log)
}

// LoggerFromContext returns the context logger, shared logger, or slog default logger.
func LoggerFromContext(ctx context.Context) *Logger {
	if ctx != nil {
		if log, ok := ctx.Value(loggerCtxKey{}).(*Logger); ok && log != nil {
			return log
		}
	}

	if Log == nil {
		return slog.Default()
	}

	return Log
}
