package logging

import (
	"log"
	"os"

	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Options struct {
	LogLevel    string
	LogEncoding string
}

// BackupLogger is used in case we can't instantiate zap (it's nicer DX than panicking or using built-in `log`)
var BackupLogger = log.New(os.Stderr, "", 0)

// MustSetupLogging sets up a zap logger - panics if provided invalid level or encoding
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

func AddCLIFlags(fs *pflag.FlagSet, options *Options) {
	fs.StringVar(&options.LogLevel, "log-level", "info", "(debug info warn error dpanic panic fatal)")
	fs.StringVar(&options.LogEncoding, "log-encoding", "console", "(console json)")
}
