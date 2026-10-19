package scenarios

import (
	"crypto/sha256"
	"fmt"
	"hash/fnv"
	"time"

	ks "github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/api/common/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

func finiteScheduleRootSelected(rootWorkflowID string, percentage int) bool {
	if percentage == 0 {
		return false
	}
	hash := fnv.New64a()
	hash.Write([]byte(rootWorkflowID))
	return int(hash.Sum64()%100) < percentage
}

func finiteScheduleID(rootWorkflowID string, internalIteration int) string {
	return fmt.Sprintf("omes-finite-%x-%d", sha256.Sum256([]byte(rootWorkflowID)), internalIteration)
}

func finiteScheduleAction(rootWorkflowID string, internalIteration int) *ks.Action {
	input := ks.FiniteScheduleInput{ScheduleID: finiteScheduleID(rootWorkflowID, internalIteration)}
	action := ks.RemoteActivityWithRetry(10*time.Second, 3, time.Second, 2)(&ks.ExecuteActivityAction{
		ActivityType: &ks.ExecuteActivityAction_Generic{
			Generic: &ks.ExecuteActivityAction_GenericActivity{
				Type: ks.CreateFiniteScheduleActivityName, Arguments: []*common.Payload{ks.ConvertToPayload(input)},
			},
		},
	})
	activity := action.GetExecActivity()
	activity.ScheduleToCloseTimeout = durationpb.New(30 * time.Second)
	activity.GetRemote().CancellationType = ks.ActivityCancellationType_WAIT_CANCELLATION_COMPLETED
	return action
}
