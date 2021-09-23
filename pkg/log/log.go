package log

import (
	"os"

	"go.uber.org/zap"
)

// NewLogger returns a configured Zap Logger
func NewLogger() *zap.Logger {
	var logConfig zap.Config
	if os.Getenv("DEBUG") == "true" {
		logConfig = zap.NewDevelopmentConfig()
	} else {
		logConfig = zap.NewProductionConfig()
	}

	level := os.Getenv("LOG_LEVEL")
	if level != "" {
		logConfig.Level = unmarshalLevel(level)
	}

	logger, err := logConfig.Build()
	if err != nil {
		panic(err)
	}
	return logger
}

func unmarshalLevel(l string) zap.AtomicLevel {
	switch l {
	case "warn":
		return zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		return zap.NewAtomicLevelAt(zap.ErrorLevel)
	case "debug":
		return zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		fallthrough
	default:
		return zap.NewAtomicLevelAt(zap.InfoLevel)
	}
}
