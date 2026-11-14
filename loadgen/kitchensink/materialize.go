package kitchensink

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	jsonProtobufEncoding = "json/protobuf"
)

// Materializer turns a LoadInput into one concrete TestInput per iteration. It is safe for concurrent use, because
// Iteration only reads the LoadInput and works on a copy.
type Materializer struct {
	inputs     []*TestInput
	cumulative []uint64
	total      uint64
	defs       map[string]*SearchAttributeDefinition
	startAttrs []string
}

// NewMaterializer validates the LoadInput. Every definition must be valid and uniquely named, and every generated name
// (at any depth, including inside encoded workflow inputs) and start search attribute must have a definition.
func NewMaterializer(in *LoadInput) (*Materializer, error) {
	m := &Materializer{
		defs:       map[string]*SearchAttributeDefinition{},
		startAttrs: in.GetStartSearchAttributes(),
	}
	for _, def := range in.GetSearchAttributes() {
		if err := validateDefinition(def); err != nil {
			return nil, err
		}
		if _, dup := m.defs[def.GetName()]; dup {
			return nil, fmt.Errorf("search attribute %q is defined more than once", def.GetName())
		}
		m.defs[def.GetName()] = def
	}
	for _, name := range m.startAttrs {
		if _, ok := m.defs[name]; !ok {
			return nil, fmt.Errorf("start search attribute %q has no definition", name)
		}
	}
	if len(in.GetInputs()) == 0 {
		return nil, errors.New("load input has no inputs")
	}
	for i, w := range in.GetInputs() {
		if w.GetInput() == nil {
			return nil, fmt.Errorf("input %d is unset", i)
		}
		m.total += uint64(w.GetWeight())
		m.cumulative = append(m.cumulative, m.total)
		m.inputs = append(m.inputs, w.GetInput())
		check := proto.Clone(w.GetInput()).(*TestInput)
		err := walkTestInput(check, func(u *UpsertSearchAttributesAction) error {
			for _, name := range u.GetGenerated() {
				if _, ok := m.defs[name]; !ok {
					return fmt.Errorf("input %d: generated search attribute %q has no definition", i, name)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	if m.total == 0 {
		return nil, errors.New("load input weights sum to 0")
	}
	return m, nil
}

// Iteration picks an input with probability weight / total weight, copies it, replaces every generated name with a
// fresh value, and returns the copy with the typed start attributes to merge into the start options.
func (m *Materializer) Iteration(r *rand.Rand, now time.Time) (*TestInput, temporal.SearchAttributes, error) {
	draw := r.Uint64N(m.total)
	idx := 0
	for m.cumulative[idx] <= draw {
		idx++
	}
	in := proto.Clone(m.inputs[idx]).(*TestInput)

	updates := make([]temporal.SearchAttributeUpdate, 0, len(m.startAttrs))
	for _, name := range m.startAttrs {
		def := m.defs[name]
		v, err := generateValue(def, r, now)
		if err != nil {
			return nil, temporal.SearchAttributes{}, err
		}
		u, err := typedSearchAttributeUpdate(def, v)
		if err != nil {
			return nil, temporal.SearchAttributes{}, err
		}
		updates = append(updates, u)
	}

	err := walkTestInput(in, func(u *UpsertSearchAttributesAction) error {
		if len(u.GetGenerated()) > 0 && u.SearchAttributes == nil {
			u.SearchAttributes = map[string]*common.Payload{}
		}
		for _, name := range u.GetGenerated() {
			def := m.defs[name]
			v, err := generateValue(def, r, now)
			if err != nil {
				return err
			}
			p, err := searchAttributePayload(def, v)
			if err != nil {
				return err
			}
			u.SearchAttributes[name] = p
		}
		u.Generated = nil
		return nil
	})
	if err != nil {
		return nil, temporal.SearchAttributes{}, err
	}
	return in, temporal.NewSearchAttributes(updates...), nil
}

// walkTestInput calls fn on every UpsertSearchAttributesAction reachable from the input, and edits the input in place.
// Workflow inputs encoded in continue-as-new arguments and child workflow inputs are decoded, walked, and re-encoded.
func walkTestInput(in *TestInput, fn func(*UpsertSearchAttributesAction) error) error {
	if err := walkWorkflowInput(in.GetWorkflowInput(), fn); err != nil {
		return err
	}
	if err := walkClientSequence(in.GetClientSequence(), fn); err != nil {
		return err
	}
	switch v := in.GetWithStartAction().GetVariant().(type) {
	case *WithStartClientAction_DoSignal:
		return walkDoSignal(v.DoSignal, fn)
	case *WithStartClientAction_DoUpdate:
		return walkDoUpdate(v.DoUpdate, fn)
	default:
		return nil
	}
}

func walkWorkflowInput(in *WorkflowInput, fn func(*UpsertSearchAttributesAction) error) error {
	for _, set := range in.GetInitialActions() {
		if err := walkActionSet(set, fn); err != nil {
			return err
		}
	}
	return nil
}

func walkActionSet(set *ActionSet, fn func(*UpsertSearchAttributesAction) error) error {
	for _, a := range set.GetActions() {
		if err := walkAction(a, fn); err != nil {
			return err
		}
	}
	return nil
}

func walkAction(a *Action, fn func(*UpsertSearchAttributesAction) error) error {
	switch v := a.GetVariant().(type) {
	case *Action_UpsertSearchAttributes:
		if v.UpsertSearchAttributes == nil {
			return nil
		}
		return fn(v.UpsertSearchAttributes)
	case *Action_NestedActionSet:
		return walkActionSet(v.NestedActionSet, fn)
	case *Action_ExecChildWorkflow:
		return walkPayloads(v.ExecChildWorkflow.GetInput(), fn)
	case *Action_ContinueAsNew:
		return walkPayloads(v.ContinueAsNew.GetArguments(), fn)
	case *Action_ExecActivity:
		return walkActivity(v.ExecActivity, fn)
	case *Action_NexusOperation:
		return walkNexusOperation(v.NexusOperation, fn)
	default:
		return nil
	}
}

func walkActivity(a *ExecuteActivityAction, fn func(*UpsertSearchAttributesAction) error) error {
	if c := a.GetClient(); c != nil {
		return walkClientSequence(c.GetClientSequence(), fn)
	}
	return nil
}

func walkNexusOperation(op *ExecuteNexusOperation, fn func(*UpsertSearchAttributesAction) error) error {
	switch v := op.GetInput().GetAction().(type) {
	case *NexusOperationRequest_StartActivity:
		return walkActivity(v.StartActivity, fn)
	case *NexusOperationRequest_WorkflowAction:
		wa := v.WorkflowAction
		if err := walkWorkflowInput(wa.GetStartOptions().GetWorkflowInput(), fn); err != nil {
			return err
		}
		switch a := wa.GetAction().(type) {
		case *NexusWorkflowAction_Signal:
			return walkDoSignal(a.Signal, fn)
		case *NexusWorkflowAction_Update:
			return walkDoUpdate(a.Update, fn)
		default:
			return nil
		}
	default:
		return nil
	}
}

func walkClientSequence(seq *ClientSequence, fn func(*UpsertSearchAttributesAction) error) error {
	for _, set := range seq.GetActionSets() {
		if err := walkClientActionSet(set, fn); err != nil {
			return err
		}
	}
	return nil
}

func walkClientActionSet(set *ClientActionSet, fn func(*UpsertSearchAttributesAction) error) error {
	for _, a := range set.GetActions() {
		var err error
		switch v := a.GetVariant().(type) {
		case *ClientAction_DoSignal:
			err = walkDoSignal(v.DoSignal, fn)
		case *ClientAction_DoUpdate:
			err = walkDoUpdate(v.DoUpdate, fn)
		case *ClientAction_NestedActions:
			err = walkClientActionSet(v.NestedActions, fn)
		case *ClientAction_DoStandaloneNexusOperation:
			err = walkNexusOperation(v.DoStandaloneNexusOperation.GetOperation(), fn)
		case *ClientAction_DoStandaloneActivity:
			err = walkActivity(v.DoStandaloneActivity.GetActivity(), fn)
		case *ClientAction_DoStandaloneActivityOperatorCommands:
			err = walkActivity(v.DoStandaloneActivityOperatorCommands.GetActivity(), fn)
		default:
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func walkDoSignal(s *DoSignal, fn func(*UpsertSearchAttributesAction) error) error {
	switch v := s.GetDoSignalActions().GetVariant().(type) {
	case *DoSignal_DoSignalActions_DoActions:
		return walkActionSet(v.DoActions, fn)
	case *DoSignal_DoSignalActions_DoActionsInMain:
		return walkActionSet(v.DoActionsInMain, fn)
	default:
		return nil
	}
}

func walkDoUpdate(u *DoUpdate, fn func(*UpsertSearchAttributesAction) error) error {
	if set := u.GetDoActions().GetDoActions(); set != nil {
		return walkActionSet(set, fn)
	}
	return nil
}

// walkPayloads walks and re-encodes each payload that holds a WorkflowInput, and leaves other payloads untouched.
func walkPayloads(payloads []*common.Payload, fn func(*UpsertSearchAttributesAction) error) error {
	for i, p := range payloads {
		if !isWorkflowInputPayload(p) {
			continue
		}
		var wi WorkflowInput
		if err := protojson.Unmarshal(p.GetData(), &wi); err != nil {
			return fmt.Errorf("decode workflow input payload: %w", err)
		}
		if err := walkWorkflowInput(&wi, fn); err != nil {
			return err
		}
		payloads[i] = ConvertToPayload(&wi)
	}
	return nil
}

func isWorkflowInputPayload(p *common.Payload) bool {
	return string(p.GetMetadata()["encoding"]) == jsonProtobufEncoding &&
		string(p.GetMetadata()["messageType"]) == string((&WorkflowInput{}).ProtoReflect().Descriptor().FullName())
}
