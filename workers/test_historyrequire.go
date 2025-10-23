package workers

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	historypb "go.temporal.io/api/history/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

type event struct {
	Type       string
	Attributes map[string]any
}

func (e event) String(lineNum int) string {
	out := e.Type
	if len(e.Attributes) > 0 {
		b, _ := json.Marshal(e.Attributes)
		out += " " + string(b)
	}
	return fmt.Sprintf("%3d: %s\n", lineNum, out)
}

type eventList []event

func (el eventList) findAllByType(eventType string) []struct {
	Event *event
	Index int
} {
	var result []struct {
		Event *event
		Index int
	}
	for i, evt := range el {
		if evt.Type == eventType {
			result = append(result, struct {
				Event *event
				Index int
			}{
				Event: &evt,
				Index: i,
			})
		}
	}
	return result
}

func (el eventList) Strings() []string {
	result := make([]string, len(el))
	for i, evt := range el {
		result[i] = evt.String(i + 1)
	}
	return result
}

// HistoryMatcher defines the interface for history matching strategies
type HistoryMatcher interface {
	Match(t *testing.T, actualHistoryEvents []*historypb.HistoryEvent) error
}

// FullHistoryMatcher checks that the list of history events match the expected spec.
// The expected spec is a string with each line containing an (1) event type, (2) optional
// JSON literal matching the event attributes and (3) an optional comment.
//
// Example:
//
//	WorkflowExecutionStarted {"taskQueue":"foo-bar"}
//	WorkflowTaskScheduled
//	WorkflowTaskStarted
//	WorkflowTaskCompleted
//	TimerStarted {"startToFireTimeout":"1s"}  # TimerStarted event with a 1s timeout
//	TimerFired
//	WorkflowTaskScheduled
//	WorkflowTaskStarted
//	WorkflowTaskCompleted
//	WorkflowExecutionCompleted
type FullHistoryMatcher string

func (m FullHistoryMatcher) Match(t *testing.T, actualHistoryEvents []*historypb.HistoryEvent) error {
	t.Helper()

	actualEvents := parseEvents(actualHistoryEvents)
	expectedEvents := parseSpec(t, string(m))

	var diffs []string
	var actualDump strings.Builder
	var expectedDump strings.Builder
	maxLines := max(len(actualEvents), len(expectedEvents))

	for i := range maxLines {
		lineNumber := i + 1

		var expectedEvent event
		if i < len(expectedEvents) {
			expectedEvent = expectedEvents[i]
			expectedDump.WriteString(expectedEvent.String(lineNumber))
		}

		if i >= len(actualEvents) {
			actualDump.WriteString(fmt.Sprintf("%3d: <missing event>\n", lineNumber))
			diffs = append(diffs, fmt.Sprintf("line %d: missing event", lineNumber))
			continue
		}
		actualEvent := actualEvents[i]

		var fieldsToShow map[string]any
		if expectedEvent.Attributes != nil && len(actualEvent.Attributes) > 0 {
			fieldsToShow = actualEvent.Attributes
		}
		displayEvent := event{Type: actualEvent.Type, Attributes: fieldsToShow}
		actualDump.WriteString(displayEvent.String(lineNumber))

		if expectedEvent.Type == "" {
			diffs = append(diffs, fmt.Sprintf("line %d: unexpected event", lineNumber))
		} else if actualEvent.Type != expectedEvent.Type {
			diffs = append(diffs, fmt.Sprintf("line %d: type mismatch, expected %s got %s",
				lineNumber, expectedEvent.Type, actualEvent.Type))
		} else if !mapIsSuperset(actualEvent.Attributes, expectedEvent.Attributes) {
			diffs = append(diffs, fmt.Sprintf("line %d: field(s) mismatch: expected %#v got %#v",
				lineNumber, expectedEvent.Attributes, actualEvent.Attributes))
		}
	}

	if len(diffs) > 0 {
		logHistoryMismatch(t, string(m), actualEvents)

		t.Log("--- DIFFS ---\n")
		for _, d := range diffs {
			t.Log(d)
		}
		return fmt.Errorf("history match failed with %d diffs", len(diffs))
	}

	return nil
}

// PartialHistoryMatcher checks that the list of history events contains a subsequence
// that matches the expected spec. The expected spec is a string with each line containing an (1) event type,
// (2) optional JSON literal matching the event attributes and (3) an optional comment.
// Use "..." as an event type to skip any number of events.
//
// Example:
//
//	WorkflowExecutionStarted {"taskQueue":"foo-bar"}
//	...
//	WorkflowExecutionCompleted
type PartialHistoryMatcher string

func (m PartialHistoryMatcher) Match(t *testing.T, actualHistoryEvents []*historypb.HistoryEvent) error {
	t.Helper()

	actualEvents := parseEvents(actualHistoryEvents)
	expectedEvents := parseSpec(t, string(m))

	var matchFromPosition func(actualIdx, expectedIdx int) bool
	matchFromPosition = func(actualIdx, expectedIdx int) bool {
		if expectedIdx >= len(expectedEvents) {
			return true
		}
		if actualIdx >= len(actualEvents) {
			return false
		}

		expectedEvent := expectedEvents[expectedIdx]

		if expectedEvent.Type == "..." {
			for i := actualIdx; i <= len(actualEvents); i++ {
				if matchFromPosition(i, expectedIdx+1) {
					return true
				}
			}
			return false
		}

		actualEvent := actualEvents[actualIdx]
		if actualEvent.Type == expectedEvent.Type && mapIsSuperset(actualEvent.Attributes, expectedEvent.Attributes) {
			return matchFromPosition(actualIdx+1, expectedIdx+1)
		}
		return matchFromPosition(actualIdx+1, expectedIdx)
	}

	if matchFromPosition(0, 0) {
		return nil
	}

	logHistoryMismatch(t, string(m), actualEvents)
	return fmt.Errorf("partialHistory matcher failed: expected sequence not found in history")
}

func parseSpec(t *testing.T, input string) eventList {
	var events eventList
	for lineNum, raw := range strings.Split(strings.TrimSpace(input), "\n") {
		raw = strings.TrimSpace(raw)

		if commentIdx := strings.Index(raw, "#"); commentIdx >= 0 {
			raw = strings.TrimSpace(raw[:commentIdx])
		}

		tokens := strings.SplitN(raw, " ", 2)
		require.NotEmptyf(t, tokens, "line %d: empty line in spec", lineNum+1)

		evt := event{Type: tokens[0]}
		if len(tokens) > 1 {
			err := json.Unmarshal([]byte(tokens[1]), &evt.Attributes)
			require.NoErrorf(t, err, "line %d: invalid JSON in line: %q", lineNum+1, raw)
		}
		events = append(events, evt)
	}
	return events
}

func parseEvents(events []*historypb.HistoryEvent) eventList {
	var result eventList
	for _, histEvent := range events {
		eventType := strings.TrimPrefix(histEvent.GetEventType().String(), "EVENT_TYPE_")
		fields := extractFields(histEvent)
		result = append(result, event{
			Type:       eventType,
			Attributes: fields,
		})
	}
	return result
}

func extractFields(event *historypb.HistoryEvent) map[string]any {
	b, err := protojson.Marshal(event)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal protobuf event: %v", err))
	}

	var raw map[string]any
	if err = json.Unmarshal(b, &raw); err != nil {
		panic(fmt.Sprintf("failed to unmarshal JSON: %v", err))
	}
	result := make(map[string]any)

	for k, v := range raw {
		if strings.HasSuffix(k, "Attributes") {
			if inner, ok := v.(map[string]any); ok {
				maps.Copy(result, inner)
			}
		} else if k != "eventId" && k != "eventType" {
			result[k] = v
		}
	}
	return result
}

func mapIsSuperset(actual, expected map[string]any) bool {
	for k, v := range expected {
		av, ok := actual[k]
		if !ok || !looselyEqual(av, v) {
			return false
		}
	}
	return true
}

func looselyEqual(x, y any) bool {
	switch x := x.(type) {
	case float64:
		switch y := y.(type) {
		case float64:
			return x == y
		case int, int32, int64:
			return x == float64(reflect.ValueOf(y).Int())
		}
	case string:
		return fmt.Sprint(x) == fmt.Sprint(y)
	case map[string]any:
		if yMap, ok := y.(map[string]any); ok {
			return mapIsSuperset(x, yMap)
		}
		return false
	}
	return reflect.DeepEqual(x, y)
}

func logHistoryMismatch(t *testing.T, expectedSpec string, actualEvents eventList) {
	t.Helper()

	t.Logf("\n--- MISMATCH ---\n\n")

	if expectedSpec != "" {
		expectedEvents := parseSpec(t, expectedSpec)
		t.Log("EXPECTED:")
		for _, expectedEvent := range expectedEvents {
			out := expectedEvent.Type
			if len(expectedEvent.Attributes) > 0 {
				b, _ := json.Marshal(expectedEvent.Attributes)
				out += " " + string(b)
			}
			t.Logf("  %s\n", out)
		}
	}

	t.Log("ACTUAL:")
	logHistoryEvents(t, actualEvents)
}

func LogHistoryEvents(t *testing.T, events []*historypb.HistoryEvent) {
	t.Helper()
	logHistoryEvents(t, parseEvents(events))
}

func logHistoryEvents(t *testing.T, actualEvents eventList) {
	t.Helper()
	for _, actualEvent := range actualEvents {
		out := actualEvent.Type
		if len(actualEvent.Attributes) > 0 {
			b, _ := json.Marshal(actualEvent.Attributes)
			out += " " + string(b)
		}
		t.Logf("  %s\n", out)
	}
}
