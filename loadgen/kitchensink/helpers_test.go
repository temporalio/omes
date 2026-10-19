package kitchensink

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
	"google.golang.org/protobuf/proto"
)

func TestGenericActivityNameAndArgs(t *testing.T) {
	dc := converter.GetDefaultDataConverter()
	encoded, err := dc.ToPayloads("left", int32(23), []byte{0, 255, 7})
	require.NoError(t, err)
	action := &ExecuteActivityAction{ActivityType: &ExecuteActivityAction_Generic{
		Generic: &ExecuteActivityAction_GenericActivity{Type: "registered_probe", Arguments: encoded.Payloads},
	}}
	name, args := ActivityNameAndArgs(action)
	require.Equal(t, "registered_probe", name)
	roundTrip, err := dc.ToPayloads(args...)
	require.NoError(t, err)
	require.True(t, proto.Equal(encoded, roundTrip), "argument payloads and encoding metadata must survive unchanged")
	var text string
	var number int32
	var data []byte
	require.NoError(t, dc.FromPayloads(roundTrip, &text, &number, &data))
	require.Equal(t, "left", text)
	require.Equal(t, int32(23), number)
	require.Equal(t, []byte{0, 255, 7}, data)
}

func TestGenericNoopUnchanged(t *testing.T) {
	name, args := ActivityNameAndArgs(GenericActivity("noop", DefaultRemoteActivity).GetExecActivity())
	require.Equal(t, "noop", name)
	require.Empty(t, args)
}

func TestGenericActivityPreservesPayloadMetadata(t *testing.T) {
	payload := &common.Payload{Metadata: map[string][]byte{"encoding": []byte("custom/opaque")}, Data: []byte{1, 2, 3}}
	_, args := ActivityNameAndArgs(&ExecuteActivityAction{ActivityType: &ExecuteActivityAction_Generic{
		Generic: &ExecuteActivityAction_GenericActivity{Type: "opaque", Arguments: []*common.Payload{payload}},
	}})
	roundTrip, err := converter.GetDefaultDataConverter().ToPayloads(args...)
	require.NoError(t, err)
	require.Len(t, roundTrip.Payloads, 1)
	require.True(t, proto.Equal(payload, roundTrip.Payloads[0]))
}
