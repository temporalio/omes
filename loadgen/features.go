package loadgen

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/temporalio/omes/clioptions"
)

// featureProbeBackoff is the wait after each failed capability probe, so a probe
// makes up to len(featureProbeBackoff)+1 attempts before giving up.
var featureProbeBackoff = []time.Duration{300 * time.Millisecond, 600 * time.Millisecond, 1200 * time.Millisecond}

// featureProbeTimeout bounds a single attempt. Without it a server that answers
// slowly rather than not at all would hang the start of the run indefinitely: the
// context here carries no deadline, and retries never begin because the attempt
// never ends.
const featureProbeTimeout = 10 * time.Second

// ResolveFeatureOptions finalizes capability-gated options declared with
// [OptionSet.Feature] against what the server reports. It is a no-op for a
// scenario that declares none, so such a scenario never pays for the calls and
// cannot be broken by one failing.
func ResolveFeatureOptions(
	ctx context.Context, cl client.Client, namespace string, opts *OptionSet, logger *zap.SugaredLogger,
) error {
	if opts == nil || !opts.hasFeatures() {
		return nil
	}

	caps, err := fetchCapabilities(ctx, cl, namespace)
	if err != nil {
		return err
	}
	if err := opts.ResolveFeaturesFromCapabilities(caps); err != nil {
		return err
	}

	for _, name := range opts.sortedFeatureNames() {
		enabled, _ := opts.GetBool(name)
		msg := fmt.Sprintf("feature %s enabled for namespace %q? %v", name, namespace, enabled)
		// Library callers reach this without a zap logger; a nil one would panic
		// here and take down a run that had otherwise resolved correctly.
		if logger != nil {
			logger.Info(msg)
		} else {
			clioptions.BackupLogger.Println(msg)
		}
	}
	return nil
}

// fetchCapabilities reads both what the namespace reports and what the server as
// a whole reports, since a feature may be gated on either.
func fetchCapabilities(ctx context.Context, cl client.Client, namespace string) (Capabilities, error) {
	var caps Capabilities

	ns, err := probe(ctx, "namespace capabilities",
		func(ctx context.Context) (*workflowservice.DescribeNamespaceResponse, error) {
			return cl.WorkflowService().DescribeNamespace(ctx,
				&workflowservice.DescribeNamespaceRequest{Namespace: namespace})
		})
	if err != nil {
		return Capabilities{}, err
	}
	caps.Namespace = ns.GetNamespaceInfo().GetCapabilities()

	sys, err := probe(ctx, "server capabilities",
		func(ctx context.Context) (*workflowservice.GetSystemInfoResponse, error) {
			return cl.WorkflowService().GetSystemInfo(ctx, &workflowservice.GetSystemInfoRequest{})
		})
	if err != nil {
		return Capabilities{}, err
	}
	caps.System = sys.GetCapabilities()

	return caps, nil
}

// probe calls get until it succeeds, giving each attempt featureProbeTimeout and
// waiting out featureProbeBackoff in between. Resolution is all-or-nothing:
// guessing at a capability would silently change the load, so a probe that never
// answers fails the run.
func probe[T any](ctx context.Context, what string, get func(context.Context) (T, error)) (T, error) {
	var resp T
	var err error
	for attempt := 0; ; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, featureProbeTimeout)
		resp, err = get(attemptCtx)
		cancel()
		if err == nil {
			return resp, nil
		}
		// Retrying a refusal only delays the report: the answer will not change.
		if !retryableProbeError(err) || attempt >= len(featureProbeBackoff) {
			return resp, fmt.Errorf("failed to probe %s: %w", what, err)
		}
		select {
		case <-ctx.Done():
			return resp, fmt.Errorf("failed to probe %s: %w", what, ctx.Err())
		case <-time.After(featureProbeBackoff[attempt]):
		}
	}
}

// retryableProbeError reports whether another attempt could plausibly answer.
// Rejections that describe the caller rather than the server's condition —
// missing permission, an unimplemented call — are settled on the first attempt.
func retryableProbeError(err error) bool {
	switch status.Code(err) {
	case codes.PermissionDenied,
		codes.Unauthenticated,
		codes.Unimplemented,
		codes.InvalidArgument,
		codes.NotFound,
		codes.FailedPrecondition:
		return false
	}
	return true
}
