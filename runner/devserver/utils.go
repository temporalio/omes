package devserver

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

// waitServerReady repeatedly attempts to dial the server with given options until it is ready or it is time to give up.
func waitServerReady(ctx context.Context, logger *zap.SugaredLogger, options client.Options) error {
	logger.Info("Waiting for server to be ready")

	lastErr := retryFor(600, 100*time.Millisecond, func() (bool, error) {
		myClient, err := client.Dial(options)
		if err != nil {
			return false, err
		}
		defer myClient.Close()
		return true, nil
	})
	if lastErr != nil {
		return fmt.Errorf("failed connecting after timeout, last error: %w", lastErr)
	}

	return nil
}

// retryFor retries some function until it passes or we run out of attempts. Wait interval between attempts.
func retryFor(maxAttempts int, interval time.Duration, cond func() (bool, error)) error {
	var lastErr error
	for i := 0; i < maxAttempts; i++ {
		if ok, curE := cond(); ok {
			return nil
		} else {
			lastErr = curE
		}
		time.Sleep(interval)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("failed after %d attempts", maxAttempts)
	}
	return lastErr
}
