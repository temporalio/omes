package kitchensink

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/protobuf/proto"
)

var materializeNow = time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

// TestIterationFillsUpsertsThroughContinueAsNew checks that an upsert three continue-as-new levels down gets a value,
// and that each level gets its own draw.
func TestIterationFillsUpsertsThroughContinueAsNew(t *testing.T) {
	level3 := &WorkflowInput{InitialActions: []*ActionSet{generatedUpsert("VisEscalated", "VisWide")}}
	level2 := &WorkflowInput{InitialActions: []*ActionSet{
		generatedUpsert("VisEscalated", "VisWide"), continueAsNew(level3)}}
	level1 := &WorkflowInput{InitialActions: []*ActionSet{
		generatedUpsert("VisEscalated", "VisWide"), continueAsNew(level2)}}
	load := testLoadInput(1, &TestInput{WorkflowInput: level1})
	load.SearchAttributes = append(load.SearchAttributes,
		keywordDef("VisWide", 1000, PoolDistribution_POOL_DISTRIBUTION_UNIFORM))
	m, err := NewMaterializer(load)
	require.NoError(t, err)

	got, _, err := m.Iteration(newRand(), materializeNow)
	require.NoError(t, err)

	l1 := got.WorkflowInput
	l2 := decodeCAN(t, l1.InitialActions[1])
	l3 := decodeCAN(t, l2.InitialActions[1])
	var wide []string
	for _, level := range []*WorkflowInput{l1, l2, l3} {
		attrs := upsertOf(level.InitialActions[0]).SearchAttributes
		require.Contains(t, attrs, "VisEscalated")
		require.Contains(t, attrs, "VisWide")
		wide = append(wide, string(attrs["VisWide"].GetData()))
	}
	require.False(t, wide[0] == wide[1] && wide[1] == wide[2], "all levels drew the same value: %v", wide)
}

// TestIterationSkipsNilUpsert checks that an upsert action wrapper holding a nil message is left alone, not a panic.
func TestIterationSkipsNilUpsert(t *testing.T) {
	m := newTestMaterializer(t, &TestInput{WorkflowInput: &WorkflowInput{InitialActions: []*ActionSet{
		{Actions: []*Action{{Variant: &Action_UpsertSearchAttributes{}}}},
	}}})

	require.NotPanics(t, func() {
		_, _, err := m.Iteration(newRand(), materializeNow)
		require.NoError(t, err)
	})
}

// TestIterationFillsChildWorkflowInput checks that an upsert inside a child workflow's input gets a value.
func TestIterationFillsChildWorkflowInput(t *testing.T) {
	child := &WorkflowInput{InitialActions: []*ActionSet{generatedUpsert("VisEscalated")}}
	m := newTestMaterializer(t, &TestInput{WorkflowInput: &WorkflowInput{InitialActions: []*ActionSet{
		{Actions: []*Action{{Variant: &Action_ExecChildWorkflow{ExecChildWorkflow: &ExecuteChildWorkflowAction{
			Input: []*common.Payload{ConvertToPayload(child)}}}}}},
	}}})

	got, _, err := m.Iteration(newRand(), materializeNow)
	require.NoError(t, err)

	payload := got.WorkflowInput.InitialActions[0].Actions[0].GetExecChildWorkflow().Input[0]
	require.Contains(t, upsertOf(decodeWorkflowInput(t, payload).InitialActions[0]).SearchAttributes, "VisEscalated")
}

// TestIterationFillsClientActionSets checks that upserts inside signal and update action sets, the with-start action,
// and a client activity's sequence get a value.
func TestIterationFillsClientActionSets(t *testing.T) {
	signal := func() *DoSignal {
		return &DoSignal{Variant: &DoSignal_DoSignalActions_{DoSignalActions: &DoSignal_DoSignalActions{
			Variant: &DoSignal_DoSignalActions_DoActionsInMain{DoActionsInMain: generatedUpsert("VisEscalated")}}}}
	}
	update := func() *DoUpdate {
		return &DoUpdate{Variant: &DoUpdate_DoActions{DoActions: &DoActionsUpdate{
			Variant: &DoActionsUpdate_DoActions{DoActions: generatedUpsert("VisEscalated")}}}}
	}
	clientSeq := func(a *ClientAction) *ClientSequence {
		return &ClientSequence{ActionSets: []*ClientActionSet{{Actions: []*ClientAction{a}}}}
	}
	in := &TestInput{
		WorkflowInput: &WorkflowInput{InitialActions: []*ActionSet{{Actions: []*Action{{
			Variant: &Action_ExecActivity{ExecActivity: &ExecuteActivityAction{
				ActivityType: &ExecuteActivityAction_Client{Client: &ExecuteActivityAction_ClientActivity{
					ClientSequence: clientSeq(&ClientAction{Variant: &ClientAction_DoSignal{DoSignal: signal()}})}}}}}}}}},
		ClientSequence: &ClientSequence{ActionSets: []*ClientActionSet{{Actions: []*ClientAction{
			{Variant: &ClientAction_DoSignal{DoSignal: signal()}},
			{Variant: &ClientAction_DoUpdate{DoUpdate: update()}},
			{Variant: &ClientAction_NestedActions{NestedActions: &ClientActionSet{Actions: []*ClientAction{
				{Variant: &ClientAction_DoUpdate{DoUpdate: update()}}}}}},
		}}}},
		WithStartAction: &WithStartClientAction{Variant: &WithStartClientAction_DoSignal{DoSignal: signal()}},
	}
	m := newTestMaterializer(t, in)

	got, _, err := m.Iteration(newRand(), materializeNow)
	require.NoError(t, err)

	filled := func(set *ActionSet) {
		t.Helper()
		require.Contains(t, upsertOf(set).SearchAttributes, "VisEscalated")
	}
	signalSet := func(s *DoSignal) *ActionSet { return s.GetDoSignalActions().GetDoActionsInMain() }
	updateSet := func(u *DoUpdate) *ActionSet { return u.GetDoActions().GetDoActions() }
	activity := got.WorkflowInput.InitialActions[0].Actions[0].GetExecActivity().GetClient()
	filled(signalSet(activity.ClientSequence.ActionSets[0].Actions[0].GetDoSignal()))
	actions := got.ClientSequence.ActionSets[0].Actions
	filled(signalSet(actions[0].GetDoSignal()))
	filled(updateSet(actions[1].GetDoUpdate()))
	filled(updateSet(actions[2].GetNestedActions().Actions[0].GetDoUpdate()))
	filled(signalSet(got.WithStartAction.GetDoSignal()))
}

// TestIterationClearsGenerated checks that no action in the result still names a generated attribute.
func TestIterationClearsGenerated(t *testing.T) {
	inner := &WorkflowInput{InitialActions: []*ActionSet{generatedUpsert("VisEscalated", "VisTenant")}}
	m := newTestMaterializer(t, &TestInput{WorkflowInput: &WorkflowInput{InitialActions: []*ActionSet{
		generatedUpsert("VisTenant"),
		{Actions: []*Action{{Variant: &Action_NestedActionSet{NestedActionSet: generatedUpsert("VisEscalated")}}}},
		continueAsNew(inner),
	}}})

	got, _, err := m.Iteration(newRand(), materializeNow)
	require.NoError(t, err)

	require.Empty(t, generatedNames(t, got))
	for _, name := range []string{"VisEscalated", "VisTenant"} {
		require.Contains(t, upsertOf(decodeCAN(t, got.WorkflowInput.InitialActions[2]).InitialActions[0]).
			SearchAttributes, name)
	}
}

// TestIterationLeavesSourceUnchanged checks that Iteration does not modify the LoadInput it was built from.
func TestIterationLeavesSourceUnchanged(t *testing.T) {
	load := testLoadInput(1, &TestInput{WorkflowInput: &WorkflowInput{InitialActions: []*ActionSet{
		generatedUpsert("VisEscalated"),
		continueAsNew(&WorkflowInput{InitialActions: []*ActionSet{generatedUpsert("VisEscalated")}}),
	}}})
	want := proto.Clone(load)
	m, err := NewMaterializer(load)
	require.NoError(t, err)

	_, _, err = m.Iteration(newRand(), materializeNow)
	require.NoError(t, err)

	require.True(t, proto.Equal(want, load))
}

// TestIterationRepeatsForSameSeed checks that two calls with equal seeds produce equal inputs for non-Datetime
// attributes, including a call made after an earlier one.
func TestIterationRepeatsForSameSeed(t *testing.T) {
	m := newTestMaterializer(t, &TestInput{WorkflowInput: &WorkflowInput{InitialActions: []*ActionSet{
		generatedUpsert("VisEscalated", "VisTenant"),
	}}})

	first, _, err := m.Iteration(newRand(), materializeNow)
	require.NoError(t, err)
	_, _, err = m.Iteration(rand.New(rand.NewPCG(9, 9)), materializeNow)
	require.NoError(t, err)
	second, _, err := m.Iteration(newRand(), materializeNow)
	require.NoError(t, err)

	require.True(t, proto.Equal(first, second))
	require.NotEmpty(t, upsertOf(first.WorkflowInput.InitialActions[0]).SearchAttributes)
}

// TestIterationReturnsStartAttributes checks that the start attributes hold one typed value per named attribute.
func TestIterationReturnsStartAttributes(t *testing.T) {
	load := testLoadInput(1, &TestInput{})
	load.StartSearchAttributes = []string{"VisTenant", "VisEscalated"}
	m, err := NewMaterializer(load)
	require.NoError(t, err)

	_, attrs, err := m.Iteration(newRand(), materializeNow)
	require.NoError(t, err)

	require.Equal(t, 2, attrs.Size())
	tenant, ok := attrs.GetKeyword(temporal.NewSearchAttributeKeyKeyword("VisTenant"))
	require.True(t, ok)
	require.Contains(t, poolOf("VisTenant", 10), tenant)
	_, ok = attrs.GetBool(temporal.NewSearchAttributeKeyBool("VisEscalated"))
	require.True(t, ok)
}

// TestIterationPicksByWeight checks that weights 1 and 3 pick the second input about 75% of 4000 draws (within 3%).
func TestIterationPicksByWeight(t *testing.T) {
	load := &LoadInput{
		SearchAttributes: testDefs(),
		Inputs: []*WeightedTestInput{
			{Weight: 1, Input: &TestInput{WorkflowInput: &WorkflowInput{ExpectedSignalCount: 1}}},
			{Weight: 3, Input: &TestInput{WorkflowInput: &WorkflowInput{ExpectedSignalCount: 2}}},
		},
	}
	m, err := NewMaterializer(load)
	require.NoError(t, err)

	const n = 4000
	r := newRand()
	second := 0
	for range n {
		got, _, err := m.Iteration(r, materializeNow)
		require.NoError(t, err)
		if got.WorkflowInput.ExpectedSignalCount == 2 {
			second++
		}
	}

	require.InDelta(t, 0.75, float64(second)/n, 0.03)
}

// TestNewMaterializerRejectsUnknownName checks that a generated name with no definition fails, naming it.
func TestNewMaterializerRejectsUnknownName(t *testing.T) {
	load := testLoadInput(1, &TestInput{WorkflowInput: &WorkflowInput{InitialActions: []*ActionSet{
		generatedUpsert("VisMissing"),
	}}})

	_, err := NewMaterializer(load)

	require.ErrorContains(t, err, "VisMissing")
}

// TestNewMaterializerRejectsUnknownNameInPayload checks that a generated name with no definition fails when it sits in
// a continue-as-new payload.
func TestNewMaterializerRejectsUnknownNameInPayload(t *testing.T) {
	inner := &WorkflowInput{InitialActions: []*ActionSet{generatedUpsert("VisMissing")}}
	load := testLoadInput(1, &TestInput{WorkflowInput: &WorkflowInput{InitialActions: []*ActionSet{
		continueAsNew(inner)}}})

	_, err := NewMaterializer(load)

	require.ErrorContains(t, err, "VisMissing")
}

// TestNewMaterializerRejectsUnknownStartName checks that a start attribute with no definition fails, naming it.
func TestNewMaterializerRejectsUnknownStartName(t *testing.T) {
	load := testLoadInput(1, &TestInput{})
	load.StartSearchAttributes = []string{"VisNope"}

	_, err := NewMaterializer(load)

	require.ErrorContains(t, err, "VisNope")
}

// TestNewMaterializerRejectsBadDefinitions checks that an invalid or duplicate definition fails, naming it.
func TestNewMaterializerRejectsBadDefinitions(t *testing.T) {
	bad := keywordDef("VisBad", 0, PoolDistribution_POOL_DISTRIBUTION_UNIFORM)
	tests := map[string][]*SearchAttributeDefinition{
		"invalid":   {bad},
		"duplicate": {intDef("VisDup", 1, 2), intDef("VisDup", 1, 2)},
	}
	for name, defs := range tests {
		t.Run(name, func(t *testing.T) {
			load := testLoadInput(1, &TestInput{})
			load.SearchAttributes = defs

			_, err := NewMaterializer(load)

			require.ErrorContains(t, err, defs[0].Name)
		})
	}
}

// TestNewMaterializerRejectsEmptyWeights checks that a LoadInput with no inputs or zero total weight fails.
func TestNewMaterializerRejectsEmptyWeights(t *testing.T) {
	tests := map[string]*LoadInput{
		"no inputs":   {},
		"zero weight": testLoadInput(0, &TestInput{}),
	}
	for name, load := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := NewMaterializer(load)

			require.Error(t, err)
		})
	}
}

// TestIterationLeavesOtherPayloadsUntouched checks that a continue-as-new argument that is not a WorkflowInput stays
// byte-identical.
func TestIterationLeavesOtherPayloadsUntouched(t *testing.T) {
	other := ConvertToPayload(&WorkflowState{Kvs: map[string]string{"a": "b"}})
	plain := ConvertToPayload("hello")
	m := newTestMaterializer(t, &TestInput{WorkflowInput: &WorkflowInput{InitialActions: []*ActionSet{
		{Actions: []*Action{{Variant: &Action_ContinueAsNew{ContinueAsNew: &ContinueAsNewAction{
			Arguments: []*common.Payload{other, plain}}}}}},
	}}})

	got, _, err := m.Iteration(newRand(), materializeNow)
	require.NoError(t, err)

	args := got.WorkflowInput.InitialActions[0].Actions[0].GetContinueAsNew().Arguments
	require.True(t, proto.Equal(other, args[0]))
	require.True(t, proto.Equal(plain, args[1]))
}

// TestIterationConcurrent checks that concurrent iterations over one Materializer are safe; run with -race.
func TestIterationConcurrent(t *testing.T) {
	m := newTestMaterializer(t, &TestInput{WorkflowInput: &WorkflowInput{InitialActions: []*ActionSet{
		generatedUpsert("VisEscalated"),
		continueAsNew(&WorkflowInput{InitialActions: []*ActionSet{generatedUpsert("VisTenant")}}),
	}}})

	errs := make(chan error, 8)
	for i := range 8 {
		go func() {
			r := rand.New(rand.NewPCG(uint64(i), 1))
			for range 50 {
				if _, _, err := m.Iteration(r, materializeNow); err != nil {
					errs <- err
					return
				}
			}
			errs <- nil
		}()
	}
	for range 8 {
		require.NoError(t, <-errs)
	}
}

func testDefs() []*SearchAttributeDefinition {
	return []*SearchAttributeDefinition{
		keywordDef("VisTenant", 10, PoolDistribution_POOL_DISTRIBUTION_UNIFORM),
		boolDef("VisEscalated", 0.5),
	}
}

func testLoadInput(weight uint32, in *TestInput) *LoadInput {
	return &LoadInput{
		SearchAttributes: testDefs(),
		Inputs:           []*WeightedTestInput{{Weight: weight, Input: in}},
	}
}

func newTestMaterializer(t *testing.T, in *TestInput) *Materializer {
	t.Helper()
	m, err := NewMaterializer(testLoadInput(1, in))
	require.NoError(t, err)
	return m
}

func generatedUpsert(names ...string) *ActionSet {
	return &ActionSet{Actions: []*Action{{Variant: &Action_UpsertSearchAttributes{
		UpsertSearchAttributes: &UpsertSearchAttributesAction{Generated: names}}}}}
}

func continueAsNew(in *WorkflowInput) *ActionSet {
	return &ActionSet{Actions: []*Action{{Variant: &Action_ContinueAsNew{ContinueAsNew: &ContinueAsNewAction{
		Arguments: []*common.Payload{ConvertToPayload(in)}}}}}}
}

func upsertOf(set *ActionSet) *UpsertSearchAttributesAction {
	return set.Actions[0].GetUpsertSearchAttributes()
}

func decodeCAN(t *testing.T, set *ActionSet) *WorkflowInput {
	t.Helper()
	return decodeWorkflowInput(t, set.Actions[0].GetContinueAsNew().Arguments[0])
}

func decodeWorkflowInput(t *testing.T, p *common.Payload) *WorkflowInput {
	t.Helper()
	var in WorkflowInput
	require.NoError(t, converter.GetDefaultDataConverter().FromPayload(p, &in))
	return &in
}

// generatedNames returns every generated name left anywhere in the input, including encoded payloads.
func generatedNames(t *testing.T, in *TestInput) []string {
	t.Helper()
	var names []string
	walkUpserts(t, in, func(u *UpsertSearchAttributesAction) { names = append(names, u.Generated...) })
	return names
}

// walkUpserts visits every upsert action in the input using the production walker, on a clone.
func walkUpserts(t *testing.T, in *TestInput, fn func(*UpsertSearchAttributesAction)) {
	t.Helper()
	err := walkTestInput(proto.Clone(in).(*TestInput), func(u *UpsertSearchAttributesAction) error {
		fn(u)
		return nil
	})
	require.NoError(t, err)
}
