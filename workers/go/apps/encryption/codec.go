package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
	"google.golang.org/protobuf/proto"
)

const (
	// encryptedEncoding is the payload encoding this codec stamps on everything it
	// writes.
	encryptedEncoding = "binary/encrypted"

	// keyIDMetadataKey records which key encrypted a payload, so a real
	// implementation could rotate keys and still decrypt old payloads.
	keyIDMetadataKey = "encryption-key-id"

	keyID = "omes-fake-test-key"
)

// testOnlyKey is a FAKE KEY, FOR TESTING ONLY.
var testOnlyKey = []byte("omes-fake-test-key-do-not-use!!!") // 32 bytes -> AES-256

// encryptionCodec encrypts whole payloads with AES-GCM. The codec sits below the
// data converter, so it sees payloads only after the payload converter has turned
// values into bytes: what it encrypts is the serialized payload proto, metadata
// included, which is why the original encoding is recoverable on decode.
type encryptionCodec struct {
	aead cipher.AEAD
}

var _ converter.PayloadCodec = (*encryptionCodec)(nil)

func newEncryptionCodec(key []byte) (*encryptionCodec, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	return &encryptionCodec{aead: aead}, nil
}

func (c *encryptionCodec) Encode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	result := make([]*commonpb.Payload, len(payloads))
	for i, payload := range payloads {
		plaintext, err := proto.Marshal(payload)
		if err != nil {
			return payloads, fmt.Errorf("failed to marshal payload: %w", err)
		}
		nonce := make([]byte, c.aead.NonceSize())
		if _, err := rand.Read(nonce); err != nil {
			return payloads, fmt.Errorf("failed to generate nonce: %w", err)
		}
		result[i] = &commonpb.Payload{
			Metadata: map[string][]byte{
				converter.MetadataEncoding: []byte(encryptedEncoding),
				keyIDMetadataKey:           []byte(keyID),
			},
			// The nonce is not secret, so it rides along as a prefix of the ciphertext.
			Data: c.aead.Seal(nonce, nonce, plaintext, nil),
		}
	}
	return result, nil
}

func (c *encryptionCodec) Decode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	result := make([]*commonpb.Payload, len(payloads))
	for i, payload := range payloads {
		// Payloads this codec did not write pass through untouched: a single history
		// can mix encrypted payloads with ones another client wrote in the clear.
		if string(payload.GetMetadata()[converter.MetadataEncoding]) != encryptedEncoding {
			result[i] = payload
			continue
		}
		if len(payload.GetData()) < c.aead.NonceSize() {
			return payloads, fmt.Errorf("encrypted payload is shorter than the %d byte nonce", c.aead.NonceSize())
		}
		nonce, ciphertext := payload.GetData()[:c.aead.NonceSize()], payload.GetData()[c.aead.NonceSize():]
		plaintext, err := c.aead.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			return payloads, fmt.Errorf("failed to decrypt payload: %w", err)
		}
		decoded := &commonpb.Payload{}
		if err := proto.Unmarshal(plaintext, decoded); err != nil {
			return payloads, fmt.Errorf("failed to unmarshal decrypted payload: %w", err)
		}
		result[i] = decoded
	}
	return result, nil
}
