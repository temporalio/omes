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
	info.Logger.Infof("Initialising Search Attribute %q", attributeName)

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
		info.Logger.Infof("Search Attribute %q already exists", attributeName)
	} else if err != nil {
		info.Logger.Warnf("Failed to add Search Attribute %q: %v", attributeName, err)
		if !errors.As(err, &deniedErr) {
			return err
		}
	} else {
		info.Logger.Infof("Search Attribute %q added", attributeName)
	}

	return nil
}

// MinVisibilityCountEventually checks that the given visibility query returns at least the expected
// number of workflows. It repeatedly queries until it either finds the expected count or times out.
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
	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("expected at least %d workflows in visibility, got %d after waiting %v",
				minCount, lastVisibilityCount, waitAtMost)

		case <-printTicker.C:
			info.Logger.Infof("current visibility count: %d (expected at least: %d)\n", lastVisibilityCount, minCount)

		case <-countTicker.C:
			visibilityCount, err := info.Client.CountWorkflow(ctx, request)
			if err != nil {
				return fmt.Errorf("failed to count workflows in visibility: %w", err)
			}
			lastVisibilityCount = visibilityCount.Count
			if lastVisibilityCount >= int64(minCount) {
				return nil
			}
		}
	}
}
