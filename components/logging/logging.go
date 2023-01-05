package logging

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/pflag"
	"github.com/temporalio/omes/components"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Options for setting up the logger component
type Options struct {
	// Log level
	LogLevel string `flag:"log-level"`
	// Log encoding (console json)
	LogEncoding string `flag:"log-encoding"`
}

// BackupLogger is used in case we can't instantiate zap (it's nicer DX than panicking or using built-in `log`).
var BackupLogger = log.New(os.Stderr, "", 0)

// MustSetup sets up a zap logger - panics if provided invalid level or encoding.
func MustSetup(options *Options) *zap.SugaredLogger {
	level, err := zap.ParseAtomicLevel(options.LogLevel)
	if err != nil {
		BackupLogger.Fatalf("Invalid log level: %v", err)
	}
	logger, err := zap.Config{
		Level:            level,
		Encoding:         options.LogEncoding,
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
func AddCLIFlags(fs *pflag.FlagSet, options *Options, prefix string) {
	fs.StringVar(&options.LogLevel, fmt.Sprintf("%s%s", prefix, components.OptionToFlagName(options, "LogLevel")), "info", "(debug info warn error dpanic panic fatal)")
	fs.StringVar(&options.LogEncoding, fmt.Sprintf("%s%s", prefix, components.OptionToFlagName(options, "LogEncoding")), "console", "(console json)")
}
