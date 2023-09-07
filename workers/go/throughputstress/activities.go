package throughputstress

import (
	"context"
	"errors"
	"fmt"
	"math/rand"

	"go.temporal.io/api/workflowservice/v1"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/serviceerror"

	"github.com/temporalio/omes/loadgen/throughputstress"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
)

type Activities struct {
	Client client.Client
}

type PayloadActivityInput struct {
	IgnoredInputData []byte
	// DesiredOutputSize determines the size of the output data in bytes, filled randomly.
	DesiredOutputSize int
}

func MakePayloadInput(inSize, outSize int) *PayloadActivityInput {
	inDat := make([]byte, inSize)
	rand.Read(inDat)
	return &PayloadActivityInput{
		IgnoredInputData:  inDat,
		DesiredOutputSize: outSize,
	}
}

// Payload serves no purpose other than to accept inputs and return outputs of a
// specific size.
func (a *Activities) Payload(_ context.Context, in *PayloadActivityInput) ([]byte, error) {
	output := make([]byte, in.DesiredOutputSize)
	//goland:noinspection GoDeprecation -- This is fine. We don't need crypto security.
	rand.Read(output)
	return output, nil
}

func (a *Activities) SelfQuery(ctx context.Context, queryType string) error {
	info := activity.GetInfo(ctx)
	wid := info.WorkflowExecution.ID

	resp, err := a.Client.QueryWorkflowWithOptions(
		ctx,
		&client.QueryWorkflowWithOptionsRequest{
			WorkflowID: wid,
			QueryType:  queryType,
		},
	)

	if err != nil {
		return err
	}

	if resp.QueryRejected != nil {
		return fmt.Errorf("query rejected: %s", resp.QueryRejected)
	}

	return nil
}

func (a *Activities) SelfDescribe(ctx context.Context) error {
	info := activity.GetInfo(ctx)
	wid := info.WorkflowExecution.ID

	_, err := a.Client.DescribeWorkflowExecution(ctx, wid, "")
	if err != nil {
		return err
	}
	return nil
}

func (a *Activities) SelfUpdate(ctx context.Context, updateName string) error {
	we := activity.GetInfo(ctx).WorkflowExecution
	handle, err := a.Client.UpdateWorkflow(ctx, we.ID, we.RunID, updateName, nil)
	if err != nil {
		return err
	}
	return handle.Get(ctx, nil)
}

func (a *Activities) SelfSignal(ctx context.Context, signalName string) error {
	we := activity.GetInfo(ctx).WorkflowExecution
	return a.Client.SignalWorkflow(ctx, we.ID, we.RunID, signalName, nil)
}

func (a *Activities) EnsureSearchAttributeRegistered(ctx context.Context, namespace string) error {
	attribMap := map[string]enums.IndexedValueType{
		throughputstress.ThroughputStressScenarioIdSearchAttribute: enums.INDEXED_VALUE_TYPE_KEYWORD,
	}
	_, err := a.Client.OperatorService().AddSearchAttributes(ctx,
		&operatorservice.AddSearchAttributesRequest{
			Namespace:        namespace,
			SearchAttributes: attribMap,
		})
	var svcErr *serviceerror.AlreadyExists
	if !errors.As(err, &svcErr) {
		return err
	}

	return nil
}

func (a *Activities) VisibilityCount(ctx context.Context, query string) (int64, error) {
	visibilityCount, err := a.Client.CountWorkflow(ctx, &workflowservice.CountWorkflowExecutionsRequest{
		Query: query,
	})
	if err != nil {
		return -1, err
	}
	return visibilityCount.Count, nil
}

type VisibilityCountMatchesInput struct {
	Query    string
	Expected int
}

func (a *Activities) VisibilityCountMatches(ctx context.Context, input *VisibilityCountMatchesInput) error {
	count, err := a.VisibilityCount(ctx, input.Query)
	if err != nil {
		return err
	}
	if count != int64(input.Expected) {
		return fmt.Errorf("expected %d workflows in visibility, got %d", input.Expected, count)
	}
	return nil
}
