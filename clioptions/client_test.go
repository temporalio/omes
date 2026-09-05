package clioptions

import (
	"testing"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"google.golang.org/protobuf/proto"
)

func TestPassThroughPayloadConverterRoundTripsPayload(t *testing.T) {
	t.Parallel()

	expected := &commonpb.Payload{
		Metadata: map[string][]byte{"encoding": []byte("json/plain")},
		Data:     []byte(`"hello"`),
	}
	encoded, err := OmesDataConverter().ToPayload(expected)
	require.NoError(t, err)

	var actual commonpb.Payload
	require.NoError(t, OmesDataConverter().FromPayload(encoded, &actual))
	require.True(t, proto.Equal(expected, &actual))
}
