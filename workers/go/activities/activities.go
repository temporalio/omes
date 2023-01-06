package activities

import "context"

func NoopActivity(ctx context.Context) error {
	return nil
}
