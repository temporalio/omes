package kitchensink

import (
	"testing"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	"go.temporal.io/api/notificationservice/v1"
)

func TestValidateWorkerCallback(t *testing.T) {
	t.Parallel()

	success := func(sc any, result any) *notificationservice.OnCompleteRequest {
		return &notificationservice.OnCompleteRequest{
			SourceContext: ConvertToPayload(sc),
			Result:        &notificationservice.OnCompleteRequest_Success{Success: ConvertToPayload(result)},
		}
	}
	failed := func(sc any, f *failurepb.Failure) *notificationservice.OnCompleteRequest {
		return &notificationservice.OnCompleteRequest{
			SourceContext: ConvertToPayload(sc),
			Result:        &notificationservice.OnCompleteRequest_Failure{Failure: f},
		}
	}
	timeoutFailure := &failurepb.Failure{
		Message: "operation timed out",
		FailureInfo: &failurepb.Failure_TimeoutFailureInfo{TimeoutFailureInfo: &failurepb.TimeoutFailureInfo{
			TimeoutType: enumspb.TIMEOUT_TYPE_SCHEDULE_TO_CLOSE,
		}},
	}
	// The server may wrap the timeout in an operation failure.
	wrappedTimeout := &failurepb.Failure{
		Message: "operation failed",
		FailureInfo: &failurepb.Failure_NexusOperationExecutionFailureInfo{
			NexusOperationExecutionFailureInfo: &failurepb.NexusOperationFailureInfo{},
		},
		Cause: timeoutFailure,
	}
	appFailure := &failurepb.Failure{
		Message:     "boom",
		FailureInfo: &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: &failurepb.ApplicationFailureInfo{}},
	}

	expectSuccess := WorkerCallbackSourceContext{OperationID: "op", Expect: WorkerCallbackExpectSuccess, EchoValue: "hello"}
	expectTimeout := WorkerCallbackSourceContext{OperationID: "op", Expect: WorkerCallbackExpectTimeout}

	cases := []struct {
		name    string
		req     *notificationservice.OnCompleteRequest
		wantErr string
	}{
		{name: "success matches", req: success(expectSuccess, "hello")},
		{name: "success mismatch", req: success(expectSuccess, "goodbye"), wantErr: "result mismatch"},
		{name: "success expected, failure delivered", req: failed(expectSuccess, appFailure), wantErr: "expected success"},
		{
			name: "success expected, no result",
			req: &notificationservice.OnCompleteRequest{
				SourceContext: ConvertToPayload(expectSuccess),
			},
			wantErr: "no result",
		},
		{name: "timeout matches", req: failed(expectTimeout, timeoutFailure)},
		{name: "wrapped timeout matches", req: failed(expectTimeout, wrappedTimeout)},
		{name: "timeout expected, other failure", req: failed(expectTimeout, appFailure), wantErr: "without a timeout cause"},
		{name: "timeout expected, success delivered", req: success(expectTimeout, "hello"), wantErr: "expected a timeout"},
		{
			name:    "unknown expectation",
			req:     success(WorkerCallbackSourceContext{OperationID: "op", Expect: "bogus"}, "hello"),
			wantErr: "unknown worker callback expectation",
		},
		{
			name:    "no source context",
			req:     &notificationservice.OnCompleteRequest{},
			wantErr: "no source context",
		},
		{
			name: "undecodable source context",
			req: &notificationservice.OnCompleteRequest{
				SourceContext: &commonpb.Payload{
					Metadata: map[string][]byte{"encoding": []byte("json/plain")},
					Data:     []byte("not json"),
				},
			},
			wantErr: "failed to decode worker callback source context",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ValidateWorkerCallback(tc.req)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}
