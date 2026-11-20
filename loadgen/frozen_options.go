package loadgen

import (
	"encoding/json"
	"errors"
	"sort"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

type frozenAttribute struct {
	Name string
	Type enumspb.IndexedValueType
	Data json.RawMessage
}
type frozenStartOptions struct {
	Options    client.StartWorkflowOptions
	Attributes []frozenAttribute
}

func freezeStartOptions(options client.StartWorkflowOptions) (frozenStartOptions, error) {
	out := frozenStartOptions{Options: options}
	if options.VersioningOverride != nil {
		return out, errors.New("journaled starts do not support versioning overrides")
	}
	for k, v := range options.TypedSearchAttributes.GetUntypedValues() {
		data, err := json.Marshal(v)
		if err != nil {
			return out, err
		}
		out.Attributes = append(out.Attributes, frozenAttribute{k.GetName(), k.GetValueType(), data})
	}
	sort.Slice(out.Attributes, func(i, j int) bool { return out.Attributes[i].Name < out.Attributes[j].Name })
	out.Options.TypedSearchAttributes = temporal.NewSearchAttributes()
	return out, nil
}
func (s frozenStartOptions) restore() (client.StartWorkflowOptions, error) {
	attrs := []temporal.SearchAttributeUpdate{}
	for _, a := range s.Attributes {
		switch a.Type {
		case enumspb.INDEXED_VALUE_TYPE_TEXT:
			var v string
			if err := json.Unmarshal(a.Data, &v); err != nil {
				return s.Options, err
			}
			attrs = append(attrs, temporal.NewSearchAttributeKeyString(a.Name).ValueSet(v))
		case enumspb.INDEXED_VALUE_TYPE_KEYWORD:
			var v string
			if err := json.Unmarshal(a.Data, &v); err != nil {
				return s.Options, err
			}
			attrs = append(attrs, temporal.NewSearchAttributeKeyKeyword(a.Name).ValueSet(v))
		case enumspb.INDEXED_VALUE_TYPE_INT:
			var v int64
			if err := json.Unmarshal(a.Data, &v); err != nil {
				return s.Options, err
			}
			attrs = append(attrs, temporal.NewSearchAttributeKeyInt64(a.Name).ValueSet(v))
		case enumspb.INDEXED_VALUE_TYPE_DOUBLE:
			var v float64
			if err := json.Unmarshal(a.Data, &v); err != nil {
				return s.Options, err
			}
			attrs = append(attrs, temporal.NewSearchAttributeKeyFloat64(a.Name).ValueSet(v))
		case enumspb.INDEXED_VALUE_TYPE_BOOL:
			var v bool
			if err := json.Unmarshal(a.Data, &v); err != nil {
				return s.Options, err
			}
			attrs = append(attrs, temporal.NewSearchAttributeKeyBool(a.Name).ValueSet(v))
		case enumspb.INDEXED_VALUE_TYPE_DATETIME:
			var v time.Time
			if err := json.Unmarshal(a.Data, &v); err != nil {
				return s.Options, err
			}
			attrs = append(attrs, temporal.NewSearchAttributeKeyTime(a.Name).ValueSet(v))
		case enumspb.INDEXED_VALUE_TYPE_KEYWORD_LIST:
			var v []string
			if err := json.Unmarshal(a.Data, &v); err != nil {
				return s.Options, err
			}
			attrs = append(attrs, temporal.NewSearchAttributeKeyKeywordList(a.Name).ValueSet(v))
		default:
			return s.Options, errors.New("unsupported frozen search attribute type")
		}
	}
	s.Options.TypedSearchAttributes = temporal.NewSearchAttributes(attrs...)
	return s.Options, nil
}
