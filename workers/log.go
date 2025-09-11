package workers

import (
	"bytes"

	"go.uber.org/zap"
)

// logWriter implements io.Writer and streams output line by line to a logger
type logWriter struct {
	logger *zap.SugaredLogger
	buffer bytes.Buffer
}

func (w *logWriter) Write(p []byte) (n int, err error) {
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
