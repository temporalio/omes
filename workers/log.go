package workers

import (
	"bytes"

	"go.uber.org/zap"
)

// logWriter implements io.Writer and streams output line by line to a logger.
type logWriter struct {
	logger *zap.SugaredLogger
	buffer bytes.Buffer
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	w.buffer.Write(p)

	for {
		line, readErr := w.buffer.ReadBytes('\n')
		if readErr != nil {
			// No complete line found, put back the partial line
			w.buffer.Write(line)
			break
		}
		lineStr := string(bytes.TrimSuffix(line, []byte("\n")))
		w.logger.Debugf("%s", lineStr)
	}

	// EOF from ReadBytes just means no complete line yet; Write consumed all of p.
	return len(p), nil //nolint:nilerr
}
