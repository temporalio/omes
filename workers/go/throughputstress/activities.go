package throughputstress

import (
	"context"
	"fmt"
	"math/rand"
	"time"

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

type SleepActivityInput struct {
	Priority      int
	SleepDuration time.Duration
}

func MakePayloadInput(inSize, outSize int) *PayloadActivityInput {
	inDat := make([]byte, inSize)
	rand.Read(inDat)
	return &PayloadActivityInput{
		IgnoredInputData:  inDat,
		DesiredOutputSize: outSize,
	}
}

func MakeSleepInput(distribution throughputstress.SleepActivity[int]) *SleepActivityInput {
	prio, ok := distribution.PatternsDist.Sample()
	if !ok {
		return nil
	}
	sleep, ok := distribution.PatternDurationsDist[prio].Sample()
	if !ok {
		return nil
	}
	fmt.Println("priority", prio, "sleep", sleep)
	return &SleepActivityInput{Priority: prio, SleepDuration: sleep}
}

// Payload serves no purpose other than to accept inputs and return outputs of a
// specific size.
func (a *Activities) Payload(_ context.Context, in *PayloadActivityInput) ([]byte, error) {
	output := make([]byte, in.DesiredOutputSize)
	//goland:noinspection GoDeprecation -- This is fine. We don't need crypto security.
	rand.Read(output)
	return output, nil
}

// Sleep is an activity that sleeps for a specified duration.
func (a *Activities) Sleep(_ context.Context, in *SleepActivityInput) error {
	time.Sleep(in.SleepDuration)
	return nil
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
	handle, err := a.Client.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
		WorkflowID:   we.ID,
		RunID:        we.RunID,
		UpdateName:   updateName,
		WaitForStage: client.WorkflowUpdateStageCompleted,
	})
	if err != nil {
		return err
	}
	return handle.Get(ctx, nil)
}

func (a *Activities) SelfSignal(ctx context.Context, signalName string) error {
	we := activity.GetInfo(ctx).WorkflowExecution
	return a.Client.SignalWorkflow(ctx, we.ID, we.RunID, signalName, nil)
}
