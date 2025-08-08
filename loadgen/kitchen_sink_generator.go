package loadgen

import (
	cryptorand "crypto/rand"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"time"

	"github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/api/common/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Generator configuration
type KitchenSinkGeneratorConfig struct {
	MaxClientActionSets    int           `json:"max_client_action_sets"`
	MaxClientActionsPerSet int           `json:"max_client_actions_per_set"`
	MaxActionsPerSet       int           `json:"max_actions_per_set"`
	MaxTimer               time.Duration `json:"max_timer"`
	MaxPayloadSize         int           `json:"max_payload_size"`
	MaxClientActionSetWait time.Duration `json:"max_client_action_set_wait"`
	ActionChances          ActionChances `json:"action_chances"`
	MaxInitialActions      int           `json:"max_initial_actions"`
}

func (g *KitchenSinkGeneratorConfig) SetDefaults() {
	g.MaxClientActionSets = 250
	g.MaxClientActionsPerSet = 5
	g.MaxActionsPerSet = 5
	g.MaxTimer = time.Second
	g.MaxPayloadSize = 256
	g.MaxClientActionSetWait = time.Second
	g.ActionChances.SetDefaults()
	g.MaxInitialActions = 10
}

// Action chances configuration
type ActionChances struct {
	Timer                  float32 `json:"timer"`
	Activity               float32 `json:"activity"`
	ChildWorkflow          float32 `json:"child_workflow"`
	PatchMarker            float32 `json:"patch_marker"`
	SetWorkflowState       float32 `json:"set_workflow_state"`
	AwaitWorkflowState     float32 `json:"await_workflow_state"`
	UpsertMemo             float32 `json:"upsert_memo"`
	UpsertSearchAttributes float32 `json:"upsert_search_attributes"`
	NestedActionSet        float32 `json:"nested_action_set"`
}

func (a *ActionChances) SetDefaults() {
	a.Timer = 25.0
	a.Activity = 25.0
	a.ChildWorkflow = 25.0
	a.NestedActionSet = 12.5
	a.PatchMarker = 2.5
	a.SetWorkflowState = 2.5
	a.AwaitWorkflowState = 2.5
	a.UpsertMemo = 2.5
	a.UpsertSearchAttributes = 2.5
}

func (a *ActionChances) Verify() bool {
	sum := a.Timer + a.Activity + a.ChildWorkflow + a.PatchMarker +
		a.SetWorkflowState + a.AwaitWorkflowState + a.UpsertMemo +
		a.UpsertSearchAttributes + a.NestedActionSet
	return math.Abs(float64(sum-100.0)) < 0.001
}

func (a *ActionChances) SelectAction(value float32) string {
	if value <= a.Timer {
		return "timer"
	}
	value -= a.Timer
	if value <= a.Activity {
		return "activity"
	}
	value -= a.Activity
	if value <= a.ChildWorkflow {
		return "child_workflow"
	}
	value -= a.ChildWorkflow
	if value <= a.NestedActionSet {
		return "nested_action_set"
	}
	value -= a.NestedActionSet
	if value <= a.PatchMarker {
		return "patch_marker"
	}
	value -= a.PatchMarker
	if value <= a.SetWorkflowState {
		return "set_workflow_state"
	}
	value -= a.SetWorkflowState
	if value <= a.AwaitWorkflowState {
		return "await_workflow_state"
	}
	value -= a.AwaitWorkflowState
	if value <= a.UpsertMemo {
		return "upsert_memo"
	}
	return "upsert_search_attributes"
}

// Global context for arbitrary generation
type ArbContext struct {
	Config               KitchenSinkGeneratorConfig
	CurrentWorkflowState *kitchensink.WorkflowState
	ActionSetNestLevel   int
	Rand                 *rand.Rand
}

var (
	wfStateFieldValue = "x"
	wfTypeName        = "kitchenSink"
	searchAttrKeys    = []string{"KS_Keyword", "KS_Int"}
)

// GenerateKitchenSinkTestInput generates a kitchen sink test input with the given seed
func GenerateKitchenSinkTestInput(seed uint64, configOverride *string) (*kitchensink.TestInput, error) {
	// Create random generator with seed
	rng := rand.New(rand.NewSource(int64(seed)))

	// Load configuration
	var config KitchenSinkGeneratorConfig
	if configOverride != nil {
		configData := []byte(*configOverride)
		if err := json.Unmarshal(configData, &config); err != nil {
			return nil, fmt.Errorf("failed to parse config JSON: %w", err)
		}
	} else {
		config.SetDefaults()
	}

	if !config.ActionChances.Verify() {
		return nil, fmt.Errorf("ActionChances must sum to exactly 100, got %+v", config.ActionChances)
	}

	// Initialize global context
	arbContext := &ArbContext{
		Config:               config,
		CurrentWorkflowState: &kitchensink.WorkflowState{Kvs: make(map[string]string)},
		ActionSetNestLevel:   0,
		Rand:                 rng,
	}

	// Generate test input
	testInput := generateTestInput(arbContext)

	return testInput, nil
}

func generateTestInput(arbContext *ArbContext) *kitchensink.TestInput {
	// Generate client sequence
	clientSequence := generateClientSequence(arbContext)

	// Sometimes add with-start action
	var withStartAction *kitchensink.WithStartClientAction
	if arbContext.Rand.Float32() > 0.8 {
		withStartAction = generateWithStartClientAction(arbContext)
	}

	testInput := &kitchensink.TestInput{
		ClientSequence:  clientSequence,
		WithStartAction: withStartAction,
	}

	// Add workflow input if needed
	if arbContext.Rand.Float32() < 0.5 {
		testInput.WorkflowInput = generateWorkflowInput(arbContext)
	}

	// Add final return action
	clientSequence.ActionSets = append(clientSequence.ActionSets, &kitchensink.ClientActionSet{
		Actions: []*kitchensink.ClientAction{
			makeClientSignalAction([]*kitchensink.Action{
				{
					Variant: &kitchensink.Action_ReturnResult{
						ReturnResult: &kitchensink.ReturnResultAction{
							ReturnThis: emptyPayload(),
						},
					},
				},
			}),
		},
	})

	return testInput
}

func generateWorkflowInput(arbContext *ArbContext) *kitchensink.WorkflowInput {
	numActions := arbContext.Rand.Intn(arbContext.Config.MaxInitialActions) + 1
	actions := make([]*kitchensink.Action, numActions)
	for i := range actions {
		actions[i] = generateAction(arbContext)
	}

	return &kitchensink.WorkflowInput{
		InitialActions: []*kitchensink.ActionSet{
			{
				Actions:    actions,
				Concurrent: arbContext.Rand.Float32() < 0.5,
			},
		},
	}
}

func generateClientSequence(arbContext *ArbContext) *kitchensink.ClientSequence {
	numActionSets := arbContext.Rand.Intn(arbContext.Config.MaxClientActionSets) + 1
	actionSets := make([]*kitchensink.ClientActionSet, numActionSets)
	for i := range actionSets {
		actionSets[i] = generateClientActionSet(arbContext)
	}

	return &kitchensink.ClientSequence{
		ActionSets: actionSets,
	}
}

func generateClientActionSet(arbContext *ArbContext) *kitchensink.ClientActionSet {
	arbContext.ActionSetNestLevel++
	defer func() { arbContext.ActionSetNestLevel-- }()

	// Small chance of continue as new
	if arbContext.ActionSetNestLevel == 1 && arbContext.Rand.Float32() < 0.01 {
		workflowInput := &kitchensink.WorkflowInput{
			InitialActions: []*kitchensink.ActionSet{
				{
					Actions: []*kitchensink.Action{
						{
							Variant: &kitchensink.Action_SetWorkflowState{
								SetWorkflowState: arbContext.CurrentWorkflowState,
							},
						},
					},
				},
			},
		}

		return &kitchensink.ClientActionSet{
			Actions: []*kitchensink.ClientAction{
				makeClientSignalAction([]*kitchensink.Action{
					{
						Variant: &kitchensink.Action_ContinueAsNew{
							ContinueAsNew: &kitchensink.ContinueAsNewAction{
								WorkflowType: wfTypeName,
								Arguments: []*common.Payload{
									toProtoPayload(workflowInput, "temporal.omes.kitchen_sink.WorkflowInput"),
								},
							},
						},
					},
				}),
			},
			WaitForCurrentRunToFinishAtEnd: true,
		}
	}

	numActions := arbContext.Rand.Intn(arbContext.Config.MaxClientActionsPerSet) + 1
	actions := make([]*kitchensink.ClientAction, numActions)
	for i := range actions {
		actions[i] = generateClientAction(arbContext)
	}

	var waitAtEnd *durationpb.Duration
	if arbContext.Rand.Float32() < 0.3 {
		waitMs := arbContext.Rand.Int63n(arbContext.Config.MaxClientActionSetWait.Nanoseconds() / 1000000)
		waitAtEnd = durationpb.New(time.Duration(waitMs) * time.Millisecond)
	}

	return &kitchensink.ClientActionSet{
		Actions:                        actions,
		Concurrent:                     arbContext.Rand.Float32() < 0.5,
		WaitAtEnd:                      waitAtEnd,
		WaitForCurrentRunToFinishAtEnd: false,
	}
}

func generateClientAction(arbContext *ArbContext) *kitchensink.ClientAction {
	// Limit nesting depth
	actionKind := arbContext.Rand.Intn(4)
	if arbContext.ActionSetNestLevel > 1 {
		actionKind = arbContext.Rand.Intn(3) // No nested actions at deeper levels
	}

	switch actionKind {
	case 0:
		return &kitchensink.ClientAction{
			Variant: &kitchensink.ClientAction_DoSignal{
				DoSignal: generateDoSignal(arbContext),
			},
		}
	case 1:
		return &kitchensink.ClientAction{
			Variant: &kitchensink.ClientAction_DoQuery{
				DoQuery: generateDoQuery(arbContext),
			},
		}
	case 2:
		return &kitchensink.ClientAction{
			Variant: &kitchensink.ClientAction_DoUpdate{
				DoUpdate: generateDoUpdate(arbContext),
			},
		}
	case 3:
		return &kitchensink.ClientAction{
			Variant: &kitchensink.ClientAction_NestedActions{
				NestedActions: generateClientActionSet(arbContext),
			},
		}
	default:
		panic("unexpected action kind")
	}
}

func generateWithStartClientAction(arbContext *ArbContext) *kitchensink.WithStartClientAction {
	actionKind := arbContext.Rand.Intn(2)
	switch actionKind {
	case 0:
		doSignal := generateDoSignal(arbContext)
		doSignal.WithStart = true
		return &kitchensink.WithStartClientAction{
			Variant: &kitchensink.WithStartClientAction_DoSignal{
				DoSignal: doSignal,
			},
		}
	case 1:
		doUpdate := generateDoUpdate(arbContext)
		doUpdate.WithStart = true
		return &kitchensink.WithStartClientAction{
			Variant: &kitchensink.WithStartClientAction_DoUpdate{
				DoUpdate: doUpdate,
			},
		}
	default:
		panic("unexpected action kind")
	}
}

func generateDoSignal(arbContext *ArbContext) *kitchensink.DoSignal {
	if arbContext.Rand.Float32() < 0.95 {
		// 95% of the time do actions
		if arbContext.Rand.Float32() < 0.5 {
			return &kitchensink.DoSignal{
				Variant: &kitchensink.DoSignal_DoSignalActions_{
					DoSignalActions: &kitchensink.DoSignal_DoSignalActions{
						Variant: &kitchensink.DoSignal_DoSignalActions_DoActions{
							DoActions: generateActionSet(arbContext),
						},
					},
				},
				WithStart: arbContext.Rand.Float32() < 0.3,
			}
		} else {
			return &kitchensink.DoSignal{
				Variant: &kitchensink.DoSignal_DoSignalActions_{
					DoSignalActions: &kitchensink.DoSignal_DoSignalActions{
						Variant: &kitchensink.DoSignal_DoSignalActions_DoActionsInMain{
							DoActionsInMain: generateActionSet(arbContext),
						},
					},
				},
				WithStart: arbContext.Rand.Float32() < 0.3,
			}
		}
	} else {
		// Sometimes do a not found signal
		return &kitchensink.DoSignal{
			Variant: &kitchensink.DoSignal_Custom{
				Custom: nonexistentHandlerInvocation(),
			},
			WithStart: arbContext.Rand.Float32() < 0.3,
		}
	}
}

func generateDoQuery(arbContext *ArbContext) *kitchensink.DoQuery {
	failureExpected := false
	if arbContext.Rand.Float32() < 0.95 {
		// 95% of the time report state
		return &kitchensink.DoQuery{
			Variant: &kitchensink.DoQuery_ReportState{
				ReportState: &common.Payloads{
					Payloads: []*common.Payload{emptyPayload()},
				},
			},
			FailureExpected: failureExpected,
		}
	} else {
		// Sometimes do a not found query
		failureExpected = true
		return &kitchensink.DoQuery{
			Variant: &kitchensink.DoQuery_Custom{
				Custom: nonexistentHandlerInvocation(),
			},
			FailureExpected: failureExpected,
		}
	}
}

func generateDoUpdate(arbContext *ArbContext) *kitchensink.DoUpdate {
	failureExpected := false
	rand := arbContext.Rand.Float32()

	if rand < 0.95 {
		// 95% of the time do actions
		return &kitchensink.DoUpdate{
			Variant: &kitchensink.DoUpdate_DoActions{
				DoActions: &kitchensink.DoActionsUpdate{
					Variant: &kitchensink.DoActionsUpdate_DoActions{
						DoActions: generateActionSet(arbContext),
					},
				},
			},
			FailureExpected: failureExpected,
			WithStart:       arbContext.Rand.Float32() < 0.3,
		}
	} else if rand < 0.975 {
		// 2.5% of the time do a rejection
		failureExpected = true
		return &kitchensink.DoUpdate{
			Variant: &kitchensink.DoUpdate_DoActions{
				DoActions: &kitchensink.DoActionsUpdate{
					Variant: &kitchensink.DoActionsUpdate_RejectMe{
						RejectMe: &emptypb.Empty{},
					},
				},
			},
			FailureExpected: failureExpected,
			WithStart:       arbContext.Rand.Float32() < 0.3,
		}
	} else {
		// Or not found
		failureExpected = true
		return &kitchensink.DoUpdate{
			Variant: &kitchensink.DoUpdate_Custom{
				Custom: nonexistentHandlerInvocation(),
			},
			FailureExpected: failureExpected,
			WithStart:       arbContext.Rand.Float32() < 0.3,
		}
	}
}

func generateActionSet(arbContext *ArbContext) *kitchensink.ActionSet {
	numActions := arbContext.Rand.Intn(arbContext.Config.MaxActionsPerSet) + 1
	actions := make([]*kitchensink.Action, numActions)
	for i := range actions {
		actions[i] = generateAction(arbContext)
	}

	return &kitchensink.ActionSet{
		Actions:    actions,
		Concurrent: arbContext.Rand.Float32() < 0.5,
	}
}

func generateAction(arbContext *ArbContext) *kitchensink.Action {
	actionValue := arbContext.Rand.Float32() * 100.0
	actionType := arbContext.Config.ActionChances.SelectAction(actionValue)

	switch actionType {
	case "timer":
		return &kitchensink.Action{
			Variant: &kitchensink.Action_Timer{
				Timer: generateTimerAction(arbContext),
			},
		}
	case "activity":
		return &kitchensink.Action{
			Variant: &kitchensink.Action_ExecActivity{
				ExecActivity: generateExecuteActivityAction(arbContext),
			},
		}
	case "child_workflow":
		return &kitchensink.Action{
			Variant: &kitchensink.Action_ExecChildWorkflow{
				ExecChildWorkflow: generateExecuteChildWorkflowAction(arbContext),
			},
		}
	case "patch_marker":
		return &kitchensink.Action{
			Variant: &kitchensink.Action_SetPatchMarker{
				SetPatchMarker: generateSetPatchMarkerAction(arbContext),
			},
		}
	case "set_workflow_state":
		chosenInt := arbContext.Rand.Intn(100) + 1
		arbContext.CurrentWorkflowState.Kvs[strconv.Itoa(chosenInt)] = wfStateFieldValue
		return &kitchensink.Action{
			Variant: &kitchensink.Action_SetWorkflowState{
				SetWorkflowState: &kitchensink.WorkflowState{
					Kvs: copyStringMap(arbContext.CurrentWorkflowState.Kvs),
				},
			},
		}
	case "await_workflow_state":
		if len(arbContext.CurrentWorkflowState.Kvs) == 0 {
			// Pick a different action if we've never set anything in state
			return generateAction(arbContext)
		}
		// Pick a random key from current state (sort for determinism)
		keys := make([]string, 0, len(arbContext.CurrentWorkflowState.Kvs))
		for k := range arbContext.CurrentWorkflowState.Kvs {
			keys = append(keys, k)
		}
		// Sort keys to ensure deterministic order
		for i := 0; i < len(keys); i++ {
			for j := i + 1; j < len(keys); j++ {
				if keys[i] > keys[j] {
					keys[i], keys[j] = keys[j], keys[i]
				}
			}
		}
		key := keys[arbContext.Rand.Intn(len(keys))]
		return &kitchensink.Action{
			Variant: &kitchensink.Action_AwaitWorkflowState{
				AwaitWorkflowState: &kitchensink.AwaitWorkflowState{
					Key:   key,
					Value: wfStateFieldValue,
				},
			},
		}
	case "upsert_memo":
		return &kitchensink.Action{
			Variant: &kitchensink.Action_UpsertMemo{
				UpsertMemo: generateUpsertMemoAction(arbContext),
			},
		}
	case "upsert_search_attributes":
		return &kitchensink.Action{
			Variant: &kitchensink.Action_UpsertSearchAttributes{
				UpsertSearchAttributes: generateUpsertSearchAttributesAction(arbContext),
			},
		}
	case "nested_action_set":
		return &kitchensink.Action{
			Variant: &kitchensink.Action_NestedActionSet{
				NestedActionSet: generateActionSet(arbContext),
			},
		}
	default:
		panic("unexpected action type: " + actionType)
	}
}

func generateTimerAction(arbContext *ArbContext) *kitchensink.TimerAction {
	maxMs := arbContext.Config.MaxTimer.Nanoseconds() / 1000000
	return &kitchensink.TimerAction{
		Milliseconds:    uint64(arbContext.Rand.Int63n(maxMs + 1)),
		AwaitableChoice: generateAwaitableChoice(arbContext),
	}
}

func generateExecuteActivityAction(arbContext *ArbContext) *kitchensink.ExecuteActivityAction {
	activityTypeChoice := arbContext.Rand.Intn(100) + 1

	action := &kitchensink.ExecuteActivityAction{
		StartToCloseTimeout: durationpb.New(5 * time.Second),
		AwaitableChoice:     generateAwaitableChoice(arbContext),
	}

	// Set locality
	if arbContext.Rand.Float32() < 0.5 {
		action.Locality = &kitchensink.ExecuteActivityAction_Remote{
			Remote: generateRemoteActivityOptions(arbContext),
		}
	} else {
		action.Locality = &kitchensink.ExecuteActivityAction_IsLocal{
			IsLocal: &emptypb.Empty{},
		}
	}

	// Set activity type
	switch {
	case activityTypeChoice <= 85:
		delay := time.Duration(arbContext.Rand.Intn(1001)) * time.Millisecond
		action.ActivityType = &kitchensink.ExecuteActivityAction_Delay{
			Delay: durationpb.New(delay),
		}
	case activityTypeChoice <= 90:
		action.ActivityType = &kitchensink.ExecuteActivityAction_Payload{
			Payload: generatePayloadActivity(arbContext),
		}
	default:
		action.ActivityType = &kitchensink.ExecuteActivityAction_Client{
			Client: generateClientActivity(arbContext),
		}
	}

	return action
}

func generateExecuteChildWorkflowAction(arbContext *ArbContext) *kitchensink.ExecuteChildWorkflowAction {
	input := &kitchensink.WorkflowInput{
		InitialActions: []*kitchensink.ActionSet{
			{
				Actions: []*kitchensink.Action{
					{
						Variant: &kitchensink.Action_Timer{
							Timer: &kitchensink.TimerAction{
								Milliseconds: uint64(arbContext.Rand.Intn(1001)),
							},
						},
					},
					{
						Variant: &kitchensink.Action_ReturnResult{
							ReturnResult: &kitchensink.ReturnResultAction{
								ReturnThis: emptyPayload(),
							},
						},
					},
				},
				Concurrent: false,
			},
		},
	}

	return &kitchensink.ExecuteChildWorkflowAction{
		WorkflowType:    wfTypeName,
		Input:           []*common.Payload{toProtoPayload(input, "temporal.omes.kitchen_sink.WorkflowInput")},
		AwaitableChoice: generateAwaitableChoice(arbContext),
	}
}

func generateSetPatchMarkerAction(arbContext *ArbContext) *kitchensink.SetPatchMarkerAction {
	patchId := arbContext.Rand.Intn(10) + 1
	return &kitchensink.SetPatchMarkerAction{
		PatchId:     strconv.Itoa(patchId),
		Deprecated:  patchId%2 == 0,
		InnerAction: generateAction(arbContext),
	}
}

func generateUpsertMemoAction(arbContext *ArbContext) *kitchensink.UpsertMemoAction {
	chosenInt := arbContext.Rand.Intn(100) + 1
	fields := make(map[string]*common.Payload)
	fields[strconv.Itoa(chosenInt)] = &common.Payload{
		Data: []byte{byte(chosenInt)},
	}

	return &kitchensink.UpsertMemoAction{
		UpsertedMemo: &common.Memo{
			Fields: fields,
		},
	}
}

func generateUpsertSearchAttributesAction(arbContext *ArbContext) *kitchensink.UpsertSearchAttributesAction {
	chosenSA := searchAttrKeys[arbContext.Rand.Intn(len(searchAttrKeys))]
	value := arbContext.Rand.Intn(255) + 1

	var data []byte
	var err error
	if chosenSA == "KS_Keyword" {
		data, err = json.Marshal(strconv.Itoa(value))
	} else {
		data, err = json.Marshal(value)
	}
	if err != nil {
		panic(err)
	}

	searchAttrs := make(map[string]*common.Payload)
	searchAttrs[chosenSA] = &common.Payload{
		Metadata: map[string][]byte{
			"encoding": []byte("json/plain"),
		},
		Data: data,
	}

	return &kitchensink.UpsertSearchAttributesAction{
		SearchAttributes: searchAttrs,
	}
}

func generateClientActivity(arbContext *ArbContext) *kitchensink.ExecuteActivityAction_ClientActivity {
	return &kitchensink.ExecuteActivityAction_ClientActivity{
		ClientSequence: &kitchensink.ClientSequence{
			ActionSets: []*kitchensink.ClientActionSet{
				{
					Actions:    []*kitchensink.ClientAction{generateClientAction(arbContext)},
					Concurrent: false,
				},
			},
		},
	}
}

func generatePayloadActivity(arbContext *ArbContext) *kitchensink.ExecuteActivityAction_PayloadActivity {
	maxSize := int32(arbContext.Config.MaxPayloadSize)
	return &kitchensink.ExecuteActivityAction_PayloadActivity{
		BytesToReceive: arbContext.Rand.Int31n(maxSize + 1),
		BytesToReturn:  arbContext.Rand.Int31n(maxSize + 1),
	}
}

func generateRemoteActivityOptions(arbContext *ArbContext) *kitchensink.RemoteActivityOptions {
	cancellationTypes := []kitchensink.ActivityCancellationType{
		kitchensink.ActivityCancellationType_ABANDON,
		kitchensink.ActivityCancellationType_TRY_CANCEL,
		kitchensink.ActivityCancellationType_WAIT_CANCELLATION_COMPLETED,
	}

	return &kitchensink.RemoteActivityOptions{
		CancellationType:    cancellationTypes[arbContext.Rand.Intn(len(cancellationTypes))],
		DoNotEagerlyExecute: false,
		VersioningIntent:    0,
	}
}

func generateAwaitableChoice(arbContext *ArbContext) *kitchensink.AwaitableChoice {
	choice := arbContext.Rand.Intn(5)
	result := &kitchensink.AwaitableChoice{}

	switch choice {
	case 0:
		result.Condition = &kitchensink.AwaitableChoice_WaitFinish{WaitFinish: &emptypb.Empty{}}
	case 1:
		result.Condition = &kitchensink.AwaitableChoice_Abandon{Abandon: &emptypb.Empty{}}
	case 2:
		result.Condition = &kitchensink.AwaitableChoice_CancelBeforeStarted{CancelBeforeStarted: &emptypb.Empty{}}
	case 3:
		result.Condition = &kitchensink.AwaitableChoice_CancelAfterStarted{CancelAfterStarted: &emptypb.Empty{}}
	case 4:
		result.Condition = &kitchensink.AwaitableChoice_CancelAfterCompleted{CancelAfterCompleted: &emptypb.Empty{}}
	}

	return result
}

func nonexistentHandlerInvocation() *kitchensink.HandlerInvocation {
	return &kitchensink.HandlerInvocation{
		Name: "nonexistent",
	}
}

// Helper functions
func makeClientSignalAction(actions []*kitchensink.Action) *kitchensink.ClientAction {
	return &kitchensink.ClientAction{
		Variant: &kitchensink.ClientAction_DoSignal{
			DoSignal: &kitchensink.DoSignal{
				Variant: &kitchensink.DoSignal_DoSignalActions_{
					DoSignalActions: &kitchensink.DoSignal_DoSignalActions{
						Variant: &kitchensink.DoSignal_DoSignalActions_DoActionsInMain{
							DoActionsInMain: &kitchensink.ActionSet{
								Actions:    actions,
								Concurrent: false,
							},
						},
					},
				},
				WithStart: false,
			},
		},
	}
}

func toProtoPayload(msg proto.Message, typeName string) *common.Payload {
	data, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}

	return &common.Payload{
		Metadata: map[string][]byte{
			"encoding":    []byte("binary/protobuf"),
			"messageType": []byte(typeName),
		},
		Data: data,
	}
}

func emptyPayload() *common.Payload {
	return &common.Payload{
		Metadata: map[string][]byte{
			"encoding": []byte("binary/null"),
		},
		Data: []byte{},
	}
}

func copyStringMap(src map[string]string) map[string]string {
	dst := make(map[string]string)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// GenerateExampleKitchenSinkInput generates the standard example test input
func GenerateExampleKitchenSinkInput() *kitchensink.TestInput {
	return &kitchensink.TestInput{
		ClientSequence: &kitchensink.ClientSequence{
			ActionSets: []*kitchensink.ClientActionSet{
				{
					Actions: []*kitchensink.ClientAction{
						makeClientSignalAction([]*kitchensink.Action{
							{
								Variant: &kitchensink.Action_Timer{
									Timer: &kitchensink.TimerAction{
										Milliseconds: 100,
									},
								},
							},
						}),
					},
					Concurrent:                     false,
					WaitAtEnd:                      durationpb.New(time.Second),
					WaitForCurrentRunToFinishAtEnd: false,
				},
				{
					Actions: []*kitchensink.ClientAction{
						makeClientSignalAction([]*kitchensink.Action{
							{
								Variant: &kitchensink.Action_Timer{
									Timer: &kitchensink.TimerAction{
										Milliseconds: 100,
									},
								},
							},
							{
								Variant: &kitchensink.Action_ExecActivity{
									ExecActivity: &kitchensink.ExecuteActivityAction{
										ActivityType: &kitchensink.ExecuteActivityAction_Noop{
											Noop: &emptypb.Empty{},
										},
										StartToCloseTimeout: durationpb.New(time.Second),
									},
								},
							},
							{
								Variant: &kitchensink.Action_ReturnResult{
									ReturnResult: &kitchensink.ReturnResultAction{
										ReturnThis: emptyPayload(),
									},
								},
							},
						}),
					},
				},
			},
		},
	}
}

// GenerateRandomSeed generates a random seed for the generator
func GenerateRandomSeed() (uint64, error) {
	seedBytes := make([]byte, 8)
	_, err := cryptorand.Read(seedBytes)
	if err != nil {
		return 0, err
	}

	var seed uint64
	for i, b := range seedBytes {
		seed |= uint64(b) << (i * 8)
	}

	return seed, nil
}
