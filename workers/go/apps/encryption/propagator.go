package encryption

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/workflow"
)

const (
	// headerFieldName is the Temporal header field this project propagates onto
	// activities, child workflows, signals, and continue-as-new.
	headerFieldName = "omes-encryption-tenant"

	// defaultHeaderValue keeps a header field present even when nothing put a value
	// on the context, so the header paths are always exercised.
	defaultHeaderValue = "omes-encryption-plaintext-header"
)

type headerContextKey struct{}

// plaintextPropagator carries one string through the run, encoded with the
// *default* data converter, so header fields land in history in the clear.
type plaintextPropagator struct{}

var _ workflow.ContextPropagator = plaintextPropagator{}

func (plaintextPropagator) Inject(ctx context.Context, writer workflow.HeaderWriter) error {
	return writeHeader(writer, headerValue(ctx.Value(headerContextKey{})))
}

func (plaintextPropagator) InjectFromWorkflow(ctx workflow.Context, writer workflow.HeaderWriter) error {
	return writeHeader(writer, headerValue(ctx.Value(headerContextKey{})))
}

func (plaintextPropagator) Extract(ctx context.Context, reader workflow.HeaderReader) (context.Context, error) {
	value, found, err := readHeader(reader)
	if err != nil || !found {
		return ctx, err
	}
	return context.WithValue(ctx, headerContextKey{}, value), nil
}

func (plaintextPropagator) ExtractToWorkflow(ctx workflow.Context, reader workflow.HeaderReader) (workflow.Context, error) {
	value, found, err := readHeader(reader)
	if err != nil || !found {
		return ctx, err
	}
	return workflow.WithValue(ctx, headerContextKey{}, value), nil
}

func writeHeader(writer workflow.HeaderWriter, value string) error {
	payload, err := converter.GetDefaultDataConverter().ToPayload(value)
	if err != nil {
		return fmt.Errorf("failed to encode header %s: %w", headerFieldName, err)
	}
	writer.Set(headerFieldName, payload)
	return nil
}

func readHeader(reader workflow.HeaderReader) (string, bool, error) {
	payload, ok := reader.Get(headerFieldName)
	if !ok {
		return "", false, nil
	}
	var value string
	if err := converter.GetDefaultDataConverter().FromPayload(payload, &value); err != nil {
		return "", false, fmt.Errorf("failed to decode header %s: %w", headerFieldName, err)
	}
	return value, true, nil
}

// withHeaderValue puts the value the propagator will inject on the context.
func withHeaderValue(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, headerContextKey{}, value)
}

func headerValue(value any) string {
	if str, ok := value.(string); ok && str != "" {
		return str
	}
	return defaultHeaderValue
}
