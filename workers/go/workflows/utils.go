package workflows

import (
	"fmt"

	"go.temporal.io/sdk/workflow"
)

// RunConcurrently runs the given functions concurrently, waiting for all to complete.
// Returns early if any of the functions return an error.
func RunConcurrently(ctx workflow.Context, funcs ...func(workflow.Context) error) error {
	completed := 0
	var innerErr error
	for _, f := range funcs {
		f := f
		workflow.Go(ctx, func(ctx workflow.Context) {
			if err := f(ctx); err != nil {
				innerErr = err
			}
			completed++
		})
	}

	awaitErr := workflow.Await(ctx, func() bool { return completed == len(funcs) || innerErr != nil })
	if awaitErr != nil {
		return fmt.Errorf("failed waiting on concurrent funcs: %w", awaitErr)
	}
	if innerErr != nil {
		return fmt.Errorf("error within concurrent funcs: %w", innerErr)
	}
	return nil
}
