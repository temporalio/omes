package loadgen

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetryVisibilityCountIgnoresTransientErrorsUntilMinimumObserved(t *testing.T) {
	attempts := 0
	err := retryVisibilityCount(
		context.Background(),
		3,
		100*time.Millisecond,
		time.Millisecond,
		func(context.Context) (int64, error) {
			attempts++
			switch attempts {
			case 1:
				return 0, errors.New("transient deadline")
			case 2:
				return 1, nil
			default:
				return 3, nil
			}
		},
	)

	if err != nil {
		t.Fatal(err)
	}
	if attempts != 3 {
		t.Fatalf("expected three attempts, got %d", attempts)
	}
}
