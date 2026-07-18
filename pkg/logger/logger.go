package logger

import (
	"log/slog"
	"os"
)

// Logger wraps slog.Logger with Logrus-style methods
type Logger struct {
	slog *slog.Logger
}

type Fields map[string]any

// New creates a new Logger instance with optional JSON output
func New(jsonOutput bool) *Logger {
	var handler slog.Handler
	if jsonOutput {
		handler = slog.NewJSONHandler(os.Stdout, nil)
	} else {
		handler = slog.NewTextHandler(os.Stdout, nil)
	}

	return &Logger{slog: slog.New(handler)}
}

// WithFields mimics logrus.WithFields
func (l *Logger) WithFields(fields Fields) *Logger {
	attrs := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		attrs = append(attrs, k, v)
	}
	group := slog.Group("fields", attrs...)
	return &Logger{slog: l.slog.With(group)}
}

// Info logs at Info level
func (l *Logger) Info(msg string) {
	l.slog.Info(msg)
}

// Error logs at Error level
func (l *Logger) Error(msg string) {
	l.slog.Error(msg)
}

// Add more levels if needed...
func (l *Logger) ErrorWithError(msg string, err error) {
	l.slog.Error(msg, slog.String("error", err.Error()))
}
