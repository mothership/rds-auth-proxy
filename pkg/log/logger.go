package log

import "go.uber.org/zap"

var rootLogger *zap.Logger = NewLogger()

// SetLogger sets a new logger as the root logger
func SetLogger(l *zap.Logger) {
	rootLogger = l
}

// Debug forwards debug logs to the root logger
func Debug(msg string, args ...zap.Field) {
	rootLogger.Debug(msg, args...)
}

// Info forwards info logs to the root logger
func Info(msg string, args ...zap.Field) {
	rootLogger.Info(msg, args...)
}

// Warn forwards warn logs to the root logger
func Warn(msg string, args ...zap.Field) {
	rootLogger.Warn(msg, args...)
}

// Error forwards error logs to the root logger
func Error(msg string, args ...zap.Field) {
	rootLogger.Error(msg, args...)
}

// Fatal forwards fatal logs to the root logger
func Fatal(msg string, args ...zap.Field) {
	rootLogger.Fatal(msg, args...)
}

// With returns a new logger with fields persisted
func With(args ...zap.Field) *zap.Logger {
	return rootLogger.With(args...)
}

// WithOptions returns a new logger with options persisted
func WithOptions(args ...zap.Field) *zap.Logger {
	return rootLogger.With(args...)
}
