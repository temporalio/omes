package loadgen

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
)

// InitSearchAttribute ensures that a search attribute is defined in the namespace.
// It creates the search attribute if it doesn't exist, or verifies it exists if it does.
// This implementation matches the throughput_stress initSearchAttribute behavior.
func InitSearchAttribute(
	ctx context.Context,
	info ScenarioInfo,
	attributeName string,
) error {
	_, err := info.Client.OperatorService().AddSearchAttributes(ctx,
		&operatorservice.AddSearchAttributesRequest{
			Namespace: info.Namespace,
			SearchAttributes: map[string]enums.IndexedValueType{
				attributeName: enums.INDEXED_VALUE_TYPE_KEYWORD,
			},
		})
	var deniedErr *serviceerror.PermissionDenied
	var alreadyErr *serviceerror.AlreadyExists
	if errors.As(err, &alreadyErr) {
		info.Logger.Infof("Search Attribute %q not added: already exists", attributeName)
	} else if err != nil {
		info.Logger.Warnf("Search Attribute %q not added: %v", attributeName, err)
		if !errors.As(err, &deniedErr) {
			return err
		}
	} else {
		info.Logger.Infof("Search Attribute %q added", attributeName)
	}

	// Since adding a search attribute is eventually consistent, we verify it exists before proceeding.
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		listResp, err := info.Client.OperatorService().ListSearchAttributes(ctx,
			&operatorservice.ListSearchAttributesRequest{
				Namespace: info.Namespace,
			})

		if err == nil {
			if customAttrs := listResp.GetCustomAttributes(); customAttrs != nil {
				if _, exists := customAttrs[attributeName]; exists {
					info.Logger.Infof("Search Attribute %q verified", attributeName)
					return nil
				}
			}
		} else {
			info.Logger.Warnf("Failed to list search attributes: %v", err)
		}

		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("could not find Search Attribute %q", attributeName)
}

func MinVisibilityCountEventually(
	ctx context.Context,
	info ScenarioInfo,
	request *workflowservice.CountWorkflowExecutionsRequest,
	minCount int,
	waitAtMost time.Duration,
) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, waitAtMost)
	defer cancel()

	countTicker := time.NewTicker(3 * time.Second)
	defer countTicker.Stop()

	printTicker := time.NewTicker(30 * time.Second)
	defer printTicker.Stop()

	var lastVisibilityCount int64
	done := false

	check := func() error {
		visibilityCount, err := info.Client.CountWorkflow(timeoutCtx, request)
		if err != nil {
			return fmt.Errorf("failed to count workflows in visibility: %w", err)
		}
		lastVisibilityCount = visibilityCount.Count
		if lastVisibilityCount >= int64(minCount) {
			done = true
		}
		return nil
	}

	// Initial check before entering the loop.
	if err := check(); err != nil {
		return err
	}

	// Loop until we reach the desired count or timeout.
	for !done {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf(
				"expected at least %d workflows in visibility, got %d after waiting %v",
				minCount, lastVisibilityCount, waitAtMost,
			)

		case <-printTicker.C:
			info.Logger.Infof("current visibility count: %d (expected at least: %d)\n",
				lastVisibilityCount, minCount)

		case <-countTicker.C:
			if err := check(); err != nil {
				return err
			}
		}
	}

	return nil
}
