package verification

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/temporalio/omes/loadgen"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

const (
	defaultVisibilityTimeout = 2 * time.Minute
	visibilityPollInterval   = 3 * time.Second
)

// Config holds all parameters for post-run verification.
type Config struct {
	Dial      func() (client.Client, error)
	Namespace string
	Logger    *zap.SugaredLogger

	ExecutionID          string
	MinWorkflowCount     int
	MinThroughputPerHour float64
	LoadDuration         time.Duration
}

// RunAll executes all applicable post-run verification checks.
func RunAll(ctx context.Context, cfg Config) error {
	cl, err := cfg.Dial()
	if err != nil {
		return fmt.Errorf("failed to dial Temporal for verification: %w", err)
	}
	defer cl.Close()

	cfg.Logger.Info("Starting post-run verification")

	var errs []error

	if cfg.MinWorkflowCount > 0 {
		if err := verifyVisibilityCount(ctx, cl, cfg); err != nil {
			errs = append(errs, err)
		}
	}

	// Throughput verification requires a known completed count. In --duration mode,
	// MinWorkflowCount is 0 (unknown), so we skip this check.
	if cfg.MinThroughputPerHour > 0 && cfg.MinWorkflowCount > 0 {
		if err := verifyThroughput(cfg.MinWorkflowCount, cfg.LoadDuration, cfg.MinThroughputPerHour); err != nil {
			errs = append(errs, err)
		}
	}

	if err := verifyNoFailedWorkflows(ctx, cl, cfg); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		cfg.Logger.Errorf("Post-run verification found %d issue(s)", len(errs))
		return errors.Join(errs...)
	}
	cfg.Logger.Info("Post-run verification passed")
	return nil
}

// verifyVisibilityCount polls visibility until at least minCount workflows
// are found with the given execution ID, or the timeout expires.
func verifyVisibilityCount(ctx context.Context, cl client.Client, cfg Config) error {
	query := fmt.Sprintf("%s='%s'", loadgen.OmesExecutionIDSearchAttribute, cfg.ExecutionID)
	req := &workflowservice.CountWorkflowExecutionsRequest{
		Namespace: cfg.Namespace,
		Query:     query,
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultVisibilityTimeout)
	defer cancel()

	ticker := time.NewTicker(visibilityPollInterval)
	defer ticker.Stop()

	for {
		resp, err := cl.CountWorkflow(timeoutCtx, req)
		if err != nil {
			return fmt.Errorf("failed to count workflows: %w", err)
		}
		if resp.Count >= int64(cfg.MinWorkflowCount) {
			return nil
		}
		cfg.Logger.Infof("Visibility count: %d (waiting for %d)", resp.Count, cfg.MinWorkflowCount)

		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf(
				"expected at least %d workflows in visibility, got %d after %v",
				cfg.MinWorkflowCount, resp.Count, defaultVisibilityTimeout,
			)
		case <-ticker.C:
		}
	}
}

// verifyNoFailedWorkflows checks that no workflows with the given execution ID
// have FAILED or TERMINATED status.
func verifyNoFailedWorkflows(ctx context.Context, cl client.Client, cfg Config) error {
	var errs []string
	for _, status := range []enums.WorkflowExecutionStatus{
		enums.WORKFLOW_EXECUTION_STATUS_TERMINATED,
		enums.WORKFLOW_EXECUTION_STATUS_FAILED,
	} {
		query := fmt.Sprintf("%s='%s' and ExecutionStatus = '%s'",
			loadgen.OmesExecutionIDSearchAttribute, cfg.ExecutionID, status)
		resp, err := cl.CountWorkflow(ctx, &workflowservice.CountWorkflowExecutionsRequest{
			Namespace: cfg.Namespace,
			Query:     query,
		})
		if err != nil {
			errs = append(errs, fmt.Sprintf("query %q failed: %v", query, err))
			continue
		}
		if resp.Count > 0 {
			errs = append(errs, fmt.Sprintf("%d workflows with status %s", resp.Count, status))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("workflow verification failed: %s", strings.Join(errs, "; "))
	}
	cfg.Logger.Info("No failed or terminated workflows found")
	return nil
}

// verifyThroughput checks that the completed workflow count meets the minimum
// throughput threshold (workflows per hour).
func verifyThroughput(completedCount int, loadDuration time.Duration, minPerHour float64) error {
	if loadDuration <= 0 {
		return fmt.Errorf("load duration must be positive for throughput verification")
	}
	actual := float64(completedCount) / loadDuration.Hours()
	if actual < minPerHour {
		return fmt.Errorf(
			"throughput %.1f workflows/hour is below minimum %.1f (completed %d in %v)",
			actual, minPerHour, completedCount, loadDuration,
		)
	}
	return nil
}
