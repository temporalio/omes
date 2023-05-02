package cmdoptions

import (
	"log"
	"os"

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
}

// BackupLogger is used in case we can't instantiate zap (it's nicer DX than panicking or using built-in `log`).
var BackupLogger = log.New(os.Stderr, "", 0)

// MustCreateLogger sets up a zap logger or panics on error.
func (l *LoggingOptions) MustCreateLogger() *zap.SugaredLogger {
	level, err := zap.ParseAtomicLevel(l.LogLevel)
	if err != nil {
		BackupLogger.Fatalf("Invalid log level: %v", err)
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

// AddCLIFlags adds the relevant flags to populate the options struct.
func (l *LoggingOptions) AddCLIFlags(fs *pflag.FlagSet) {
	fs.StringVar(&l.LogLevel, "log-level", "info", "(debug info warn error panic fatal)")
	fs.StringVar(&l.LogEncoding, "log-encoding", "console", "(console json)")
}

// ToFlags converts these options to string flags.
func (l *LoggingOptions) ToFlags() (flags []string) {
	if l.LogLevel != "" {
		flags = append(flags, "--log-level", l.LogLevel)
	}
	if l.LogEncoding != "" {
		flags = append(flags, "--log-encoding", l.LogEncoding)
	}
	return
}
