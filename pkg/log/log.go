package log

import (
	"os"
	"path/filepath"

	"github.com/jcdorr003/windash-agent/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// New creates a new logger with console and file output
func New(debug bool) *zap.SugaredLogger {
	// Get log directory
	logDir := config.GetLogDir()
	logFile := filepath.Join(logDir, "agent.log")

	// Lumberjack for log rotation
	fileWriter := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    10, // MB
		MaxBackups: 7,  // Keep last 7 files
		MaxAge:     7,  // days
		Compress:   true,
	}

	// Console encoder (pretty, colorful)
	consoleEncoder := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
		TimeKey:        "T",
		LevelKey:       "L",
		NameKey:        "N",
		MessageKey:     "M",
		StacktraceKey:  "S",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	})

	// File encoder (JSON for structured logs)
	fileEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())

	// Set log level
	level := zapcore.InfoLevel
	if debug {
		level = zapcore.DebugLevel
	}

	// Create multi-output core (console + file)
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level),
		zapcore.NewCore(fileEncoder, zapcore.AddSync(fileWriter), level),
	)

	// Create logger with caller info and stack traces on errors
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return logger.Sugar()
}
