package scenarios

import "testing"

func TestVisibilityQueryUsesRunWorkflowIDPrefix(t *testing.T) {
	got := visibilityWorkflowQuery("run-1")
	want := `WorkflowId STARTS_WITH "w-run-1-"`
	if got != want {
		t.Fatalf("got query %q, want %q", got, want)
	}
}
