package loadgen

import (
	"context"
	"time"
)

// RetryUntilCtx repeatedly invokes fn until it reports completion or the context is done.
// - fn should return (true, nil) when the operation has succeeded and no further retries are needed.
// - If fn returns (true, err), the retry loop stops and err is returned.
// - If fn returns (false, err), the function will be retried after a backoff delay.
// Backoff starts at 1s and doubles each time up to a maximum of 10s.
// If the context is canceled or its deadline expires, the last non-nil error from fn is returned if present;
// otherwise, the context error is returned.
func RetryUntilCtx(ctx context.Context, fn func(context.Context) (bool, error)) error {
	backoff := 1 * time.Second
	for {
		done, err := fn(ctx)
		if done {
			return err
		}

		select {
		case <-ctx.Done():
			if err != nil {
				return err
			}
			return ctx.Err()
		case <-time.After(backoff):
		}

		if backoff < 10*time.Second {
			backoff *= 2
			if backoff > 10*time.Second {
				backoff = 10 * time.Second
			}
		}
	}
}
