package logging

import (
	"log"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// BackupLogger is used in case we can't instantiate zap (it's nicer DX than panicking or using built-in `log`)
var BackupLogger = log.New(os.Stderr, "", 0)

// MustSetupLogging sets up a zap logger - panics if provided invalid level or encoding
func MustSetupLogging(logLevel, logEncoding string) *zap.SugaredLogger {
	level, err := zap.ParseAtomicLevel(logLevel)
	if err != nil {
		BackupLogger.Fatalf("Invalid log level: %v", err)
	}
	logger, err := zap.Config{
		Level:            level,
		Encoding:         logEncoding,
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}.Build(zap.AddStacktrace(zapcore.FatalLevel))
	if err != nil {
		BackupLogger.Fatalf("Failed to initialize logger: %v", err)
	}

	return logger.Sugar()
}
