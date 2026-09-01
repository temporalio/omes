package encryption

import (
	"testing"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
)

// testMessage stands in for the payload body the driver threads through a run.
const testMessage = "encryption-coverage-test"

func TestCodecRoundTrip(t *testing.T) {
	codec := newTestCodec(t)

	original, err := converter.GetDefaultDataConverter().ToPayloads(
		ActivityInput{Step: "echo", Message: testMessage},
		HeartbeatDetails{Step: "echo", Attempt: 1, Message: testMessage},
	)
	require.NoError(t, err, "failed to build payloads")

	encoded, err := codec.Encode(original.GetPayloads())
	require.NoError(t, err)
	for i, payload := range encoded {
		require.Equal(t, encryptedEncoding,
			string(payload.GetMetadata()[converter.MetadataEncoding]),
			"payload %d encoding", i)
	}

	decoded, err := codec.Decode(encoded)
	require.NoError(t, err)

	var input ActivityInput
	var heartbeat HeartbeatDetails
	require.NoError(t, converter.GetDefaultDataConverter().FromPayloads(
		&commonpb.Payloads{Payloads: decoded}, &input, &heartbeat))
	require.Equal(t, testMessage, input.Message)
	require.Equal(t, int32(1), heartbeat.Attempt)
}

func TestDataConverterRoundTrip(t *testing.T) {
	dataConverter, err := newDataConverter()
	require.NoError(t, err)

	payload, err := dataConverter.ToPayload(CoverageInput{Message: testMessage, Iteration: 1})
	require.NoError(t, err)

	var result CoverageInput
	require.NoError(t, dataConverter.FromPayload(payload, &result))
	require.Equal(t, testMessage, result.Message)
	require.Equal(t, int64(1), result.Iteration)
}

func newTestCodec(t *testing.T) *encryptionCodec {
	t.Helper()
	codec, err := newEncryptionCodec(testOnlyKey)
	require.NoError(t, err)
	return codec
}
