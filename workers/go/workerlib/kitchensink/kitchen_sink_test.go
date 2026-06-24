package kitchensink

import (
	"bytes"
	"testing"

	loadgenks "github.com/temporalio/omes/loadgen/kitchensink"
)

func TestPayloadNexusHandlerWorkflowReturnsRequestedBytes(t *testing.T) {
	output, err := PayloadNexusHandlerWorkflow(nil, &loadgenks.PayloadNexusInput{
		Input:         []byte("ignored"),
		BytesToReturn: 4,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(output, []byte{0, 1, 2, 3}) {
		t.Fatalf("unexpected output: %v", output)
	}
}
