package loadgen

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	historypb "go.temporal.io/api/history/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

type specLine struct {
	EventID int64
	Type    string
	Fields  map[string]any
}

func requireHistoryMatches(t *testing.T, actualEvents []*historypb.HistoryEvent, expectedSpec string) {
	t.Helper()

	var diffs []string
	var actualDump strings.Builder
	var expectedDump strings.Builder
	expectedLines := parseSpec(t, expectedSpec)
	maxLines := max(len(actualEvents), len(expectedLines))

	for i := 0; i < maxLines; i++ {
		lineNumber := i + 1

		var expectedLine specLine
		if i < len(expectedLines) {
			expectedLine = expectedLines[i]
			expectedDump.WriteString(formatExpectedLine(lineNumber, expectedLine, expectedLine.Fields))
		}

		if i >= len(actualEvents) {
			actualDump.WriteString(fmt.Sprintf("%3d: <missing event>\n", lineNumber))
			diffs = append(diffs, fmt.Sprintf("line %d: missing event", lineNumber))
			continue
		}
		actualEvent := actualEvents[i]

		// Transform event for comparison.
		actualType := strings.TrimPrefix(actualEvent.GetEventType().String(), "EVENT_TYPE_")
		actualFields, _ := flattenEvent(actualEvent)
		actualDump.WriteString(formatActualLine(lineNumber, actualType, expectedLine.Fields, actualFields))

		// Compare types and fields.
		if expectedLine.Type == "" {
			diffs = append(diffs, fmt.Sprintf("line %d: unexpected event", lineNumber))
		} else if actualType != expectedLine.Type {
			diffs = append(diffs, fmt.Sprintf("line %d: type mismatch, expected %s got %s",
				lineNumber, expectedLine.Type, actualType))
		} else if !mapIsSuperset(actualFields, expectedLine.Fields) {
			diffs = append(diffs, fmt.Sprintf("line %d: field(s) mismatch: expected %#v got %#v",
				lineNumber, expectedLine.Fields, actualFields))
		}
	}

	if len(diffs) > 0 {
		t.Log("--- MISMATCH ---\n")
		t.Log("EXPECTED:\n" + expectedDump.String())
		t.Log("ACTUAL:\n" + actualDump.String())

		t.Log("--- DIFFS ---")
		for _, d := range diffs {
			t.Log(d)
		}
		require.Equal(t, 0, len(diffs), "history match failed")
	}
}

func parseSpec(t *testing.T, input string) []specLine {
	t.Helper()

	var lines []specLine
	for lineNum, raw := range strings.Split(strings.TrimSpace(input), "\n") {
		raw = strings.TrimSpace(raw)

		if commentIdx := strings.Index(raw, "#"); commentIdx >= 0 {
			raw = strings.TrimSpace(raw[:commentIdx])
		}

		tokens := strings.SplitN(raw, " ", 2)
		require.NotEmptyf(t, tokens, "line %d: empty line in spec", lineNum+1)

		line := specLine{Type: tokens[0]}
		if len(tokens) > 1 {
			err := json.Unmarshal([]byte(tokens[1]), &line.Fields)
			require.NoErrorf(t, err, "line %d: invalid JSON in line: %q", lineNum+1, raw)
		}
		lines = append(lines, line)
	}
	return lines
}

func flattenEvent(event *historypb.HistoryEvent) (map[string]any, error) {
	b, err := protojson.Marshal(event)
	if err != nil {
		return nil, err
	}

	var raw map[string]any
	if err = json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	result := make(map[string]any)

	for k, v := range raw {
		if strings.HasSuffix(k, "Attributes") {
			if inner, ok := v.(map[string]any); ok {
				for ik, iv := range inner {
					result[ik] = iv
				}
			}
		} else if k != "eventId" && k != "eventType" {
			result[k] = v
		}
	}
	return result, nil
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
	}
	return reflect.DeepEqual(x, y)
}

func formatExpectedLine(lineNum int, line specLine, fields map[string]any) string {
	out := fmt.Sprintf("%3d: %s", lineNum, line.Type)
	if len(fields) > 0 {
		b, _ := json.Marshal(fields)
		out += " " + string(b)
	}
	return out + "\n"
}

func formatActualLine(lineNum int, typ string, expectedFields map[string]any, actualFields map[string]any) string {
	out := fmt.Sprintf("%3d: %s", lineNum, typ)
	if expectedFields != nil && len(actualFields) > 0 {
		b, _ := json.Marshal(actualFields)
		out += " " + string(b)
	}
	return out + "\n"
}
