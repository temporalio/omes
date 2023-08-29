package activities

import (
	"context"
)

func Noop(_ context.Context) error {
	return nil
}
