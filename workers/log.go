package workers

import (
	"bytes"

	"go.uber.org/zap"
)

// LogWriter implements io.Writer and streams output line by line to a logger
type LogWriter struct {
	logger *zap.SugaredLogger
	buffer bytes.Buffer
}

// NewLogWriter returns a LogWriter that streams output line by line to logger.
func NewLogWriter(logger *zap.SugaredLogger) *LogWriter {
	return &LogWriter{logger: logger}
}

func (w *LogWriter) Write(p []byte) (n int, err error) {
	w.buffer.Write(p)

	for {
		line, err := w.buffer.ReadBytes('\n')
		if err != nil {
			// No complete line found, put back the partial line
			w.buffer.Write(line)
			break
		}
		lineStr := string(bytes.TrimSuffix(line, []byte("\n")))
		w.logger.Debugf("%s", lineStr)
	}

	return len(p), nil
}
