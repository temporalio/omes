package loadgen

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

// featureProbeBackoff is the wait after each failed capability probe, so the
// probe makes up to len(featureProbeBackoff)+1 attempts before giving up.
var featureProbeBackoff = []time.Duration{300 * time.Millisecond, 600 * time.Millisecond, 1200 * time.Millisecond}

// ResolveFeatureOptions finalizes capability-gated options declared with
// [OptionSet.Feature] against the namespace under test. It is a no-op for a
// scenario that declares none, so such a scenario never pays for a
// DescribeNamespace call and can't be broken by one failing.
func ResolveFeatureOptions(
	ctx context.Context, cl client.Client, namespace string, opts *OptionSet, logger *zap.SugaredLogger,
) error {
	if opts == nil || !opts.hasFeatures() {
		return nil
	}

	var resp *workflowservice.DescribeNamespaceResponse
	var err error
	for attempt := 0; ; attempt++ {
		resp, err = cl.WorkflowService().DescribeNamespace(ctx, &workflowservice.DescribeNamespaceRequest{
			Namespace: namespace,
		})
		if err == nil {
			break
		}
		if attempt >= len(featureProbeBackoff) {
			return fmt.Errorf("failed to probe namespace capabilities: %w", err)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("failed to probe namespace capabilities: %w", ctx.Err())
		case <-time.After(featureProbeBackoff[attempt]):
		}
	}

	if err := opts.resolveFeatures(resp.GetNamespaceInfo().GetCapabilities()); err != nil {
		return err
	}

	for _, name := range opts.sortedFeatureNames() {
		enabled, _ := opts.GetBool(name)
		logger.Infof("feature %s enabled for namespace %q? %v", name, namespace, enabled)
	}
	return nil
}
