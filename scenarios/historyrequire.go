package scenarios

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

func RequireHistoryMatches(t *testing.T, events []*historypb.HistoryEvent, spec string) {
	t.Helper()

	var allErrs []string
	specLines := parseSpec(spec, &allErrs)

	var expectedDump strings.Builder
	var actualDump strings.Builder
	var diffs []string

	max := len(specLines)
	for i := 0; i < max; i++ {
		specLine := specLines[i]
		var actual *historypb.HistoryEvent
		if i < len(events) {
			actual = events[i]
		}

		expectedDump.WriteString(formatLine(specLine, specLine.Fields))

		if actual == nil {
			actualDump.WriteString("<missing>\n")
			diffs = append(diffs, fmt.Sprintf("line %d: missing actual event", i+1))
			continue
		}

		eType := stripPrefix(actual.GetEventType().String())
		flat, _ := flattenEvent(actual)
		actualDump.WriteString(formatLineFromMap(eType, specLine.Fields, flat))

		if !equalEventType(eType, specLine.Type) {
			diffs = append(diffs, fmt.Sprintf("line %d: type mismatch, expected %s got %s", i+1, specLine.Type, eType))
		}
		if !mapIsSuperset(flat, specLine.Fields) {
			diffs = append(diffs, fmt.Sprintf("line %d: field mismatch", i+1))
		}
	}

	t.Log("\n--- MATCH CHECK ---")
	if len(specLines) > 0 {
		t.Log("EXPECTED:\n" + expectedDump.String())
		t.Log("ACTUAL:\n" + actualDump.String())
	}

	if len(diffs) > 0 || len(allErrs) > 0 {
		t.Log("--- DIFFS ---")
		for _, d := range diffs {
			t.Log(d)
		}
		for _, err := range allErrs {
			t.Log(err)
		}
		require.Equal(t, 0, len(diffs)+len(allErrs), "history match failed")
	}
}

func parseSpec(input string, errs *[]string) []specLine {
	var lines []specLine
	for lineNum, raw := range strings.Split(strings.TrimSpace(input), "\n") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		tokens := strings.SplitN(raw, " ", 3)
		if len(tokens) < 1 {
			*errs = append(*errs, fmt.Sprintf("line %d: invalid line: %q", lineNum+1, raw))
			continue
		}

		line := specLine{Type: tokens[0]}
		if len(tokens) == 2 {
			if strings.HasPrefix(strings.TrimSpace(tokens[1]), "{") {
				err := json.Unmarshal([]byte(tokens[1]), &line.Fields)
				if err != nil {
					*errs = append(*errs, fmt.Sprintf("line %d: invalid JSON in line: %q", lineNum+1, raw))
					continue
				}
			} else {
				line.Type = tokens[1]
			}
		} else if len(tokens) == 3 {
			err := json.Unmarshal([]byte(tokens[2]), &line.Fields)
			if err != nil {
				*errs = append(*errs, fmt.Sprintf("line %d: invalid JSON in line: %q", lineNum+1, raw))
				continue
			}
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
	if err := json.Unmarshal(b, &raw); err != nil {
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

func stripPrefix(t string) string {
	const prefix = "EVENT_TYPE_"
	return strings.TrimPrefix(t, prefix)
}

func equalEventType(actual, expected string) bool {
	return strings.EqualFold(strings.ReplaceAll(actual, "_", ""), strings.ReplaceAll(expected, "_", ""))
}

func formatLine(line specLine, fields map[string]any) string {
	out := line.Type
	if len(fields) > 0 {
		b, _ := json.Marshal(fields)
		out += " " + string(b)
	}
	return out + "\n"
}

func formatLineFromMap(typ string, keys map[string]any, actual map[string]any) string {
	out := typ
	if len(keys) > 0 {
		sub := make(map[string]any)
		for k := range keys {
			if v, ok := actual[k]; ok {
				sub[k] = v
			}
		}
		b, _ := json.Marshal(sub)
		out += " " + string(b)
	}
	return out + "\n"
}
