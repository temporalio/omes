package harness

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func configureLogger(logLevel, logEncoding string) (*zap.SugaredLogger, error) {
	if logLevel == "trace" {
		// Corresponding trace level for zap logger
		logLevel = "debug"
	}
	level, err := zap.ParseAtomicLevel(logLevel)
	if err != nil {
		return nil, err
	}
	logger, err := zap.Config{
		Level:            level,
		Encoding:         logEncoding,
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}.Build(zap.AddStacktrace(zapcore.FatalLevel))
	if err != nil {
		return nil, err
	}
	return logger.Sugar(), nil
}
