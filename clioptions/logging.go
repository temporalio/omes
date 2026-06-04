package clioptions

import (
	"log"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LoggingOptions for setting up the logger component
type LoggingOptions struct {
	// Log level
	LogLevel string
	// Log encoding (console json)
	LogEncoding string
	// Prepared logger to use instead of creating a new one from options. Progamatic use only.
	PreparedLogger *zap.SugaredLogger

	fs *pflag.FlagSet
}

// BackupLogger is used in case we can't instantiate zap (it's nicer DX than panicking or using built-in `log`).
var BackupLogger = log.New(os.Stderr, "", 0)

// MustCreateLogger sets up a zap logger or panics on error.
func (l *LoggingOptions) MustCreateLogger() *zap.SugaredLogger {
	// If PreparedLogger is specified, return it instead of creating a new one.
	if l.PreparedLogger != nil {
		return l.PreparedLogger
	}

	level, err := zap.ParseAtomicLevel(l.LogLevel)
	if err != nil {
		if strings.ToLower(l.LogLevel) == "trace" {
			// Zap doesn't know about trace level, but, some of the workers do.
			level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
		} else {
			BackupLogger.Fatalf("Invalid log level: %v", err)
		}
	}
	logger, err := zap.Config{
		Level:            level,
		Encoding:         l.LogEncoding,
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}.Build(zap.AddStacktrace(zapcore.FatalLevel))
	if err != nil {
		BackupLogger.Fatalf("Failed to initialize logger: %v", err)
	}

	return logger.Sugar()
}

// FlagSet adds the relevant flags to populate the options struct and returns a pflag.FlagSet.
func (l *LoggingOptions) FlagSet() *pflag.FlagSet {
	if l.fs != nil {
		return l.fs
	}
	l.fs = pflag.NewFlagSet("logging_options", pflag.ExitOnError)
	l.fs.StringVar(&l.LogLevel, "log-level", "info", "(debug info warn error panic fatal)")
	l.fs.StringVar(&l.LogEncoding, "log-encoding", "console", "(console json)")
	return l.fs
}
