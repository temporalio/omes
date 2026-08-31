package encryption

import (
	"testing"

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
	if err != nil {
		t.Fatalf("failed to build payloads: %v", err)
	}

	encoded, err := codec.Encode(original.GetPayloads())
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	for i, payload := range encoded {
		if encoding := string(payload.GetMetadata()[converter.MetadataEncoding]); encoding != encryptedEncoding {
			t.Errorf("payload %d encoding = %q, want %q", i, encoding, encryptedEncoding)
		}
	}

	decoded, err := codec.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	var input ActivityInput
	var heartbeat HeartbeatDetails
	if err := converter.GetDefaultDataConverter().FromPayloads(
		&commonpb.Payloads{Payloads: decoded}, &input, &heartbeat); err != nil {
		t.Fatalf("failed to decode payloads: %v", err)
	}
	if input.Message != testMessage || heartbeat.Attempt != 1 {
		t.Errorf("round trip lost data: got %+v and %+v", input, heartbeat)
	}
}

func TestDataConverterRoundTrip(t *testing.T) {
	dataConverter, err := newDataConverter()
	if err != nil {
		t.Fatalf("newDataConverter failed: %v", err)
	}

	payload, err := dataConverter.ToPayload(CoverageInput{Message: testMessage, Iteration: 1})
	if err != nil {
		t.Fatalf("ToPayload failed: %v", err)
	}
	var result CoverageInput
	if err := dataConverter.FromPayload(payload, &result); err != nil {
		t.Fatalf("FromPayload failed: %v", err)
	}
	if result.Message != testMessage || result.Iteration != 1 {
		t.Errorf("round trip lost data: got %+v", result)
	}
}

func newTestCodec(t *testing.T) *encryptionCodec {
	t.Helper()
	codec, err := newEncryptionCodec(testOnlyKey)
	if err != nil {
		t.Fatalf("newEncryptionCodec failed: %v", err)
	}
	return codec
}
