package kitchensink

// WorkflowParams is the single input for the Kitchen Sink Workflow (KSW).
type WorkflowParams struct {
	Actions      []*Action `json:"actions"`
	ActionSignal string    `json:"action_signal"`
}

// Action represent a single action to be taken by the Kitchen Sink Workflow.
type Action struct {
	Result          *ResultAction          `json:"result"`
	Error           *ErrorAction           `json:"error"`
	ContinueAsNew   *ContinueAsNewAction   `json:"continue_as_new"`
	Sleep           *SleepAction           `json:"sleep"`
	QueryHandler    *QueryHandlerAction    `json:"query_handler"`
	Signal          *SignalAction          `json:"signal"`
	ExecuteActivity *ExecuteActivityAction `json:"execute_activity"`
}

// ResultAction instructs the KSW to return the given result.
type ResultAction struct {
	// Value to return (ignored if RunID is true).
	Value interface{} `json:"value"`
	// Whether to return the current run ID.
	RunID bool `json:"run_id"`
}

// ErrorAction instructs the KSW to return an error.
type ErrorAction struct {
	// If present, the KSW will fail with an ApplicationFailure with the given message (ignored if Attempt is true).
	Message string `json:"message"`
	// Details to attach to the ApplicationFailure (ignored if Attempt is true).
	Details interface{} `json:"details"`
	// Controls whether to include the attempt in the error message
	Attempt bool `json:"attempt"`
}

// ContinueAsNewAction instructs the KSW to continue-as-new.
type ContinueAsNewAction struct {
	// The Workflow will continue as new as long as this value is greater than 0.
	// The value is decremented every time continue-as-new is called.
	WhileAboveZero int `json:"while_above_zero"`
}

// SleepAction instructs the KSW to sleep.
type SleepAction struct {
	// Number of milliseconds to sleep for.
	Millis int64 `json:"millis"`
}

// QueryHandlerAction instructs the KSW to set up an "echo" query handler.
type QueryHandlerAction struct {
	// Name of the query to use for the handler.
	Name string `json:"name"`
}

// SignalAction instructs the KSW to block on a signal.
type SignalAction struct {
	// Name of the signal to block on.
	Name string `json:"name"`
}

// ExecuteActivityAction instructs the KSW to execute an activity and await its completion.
type ExecuteActivityAction struct {
	Name                     string        `json:"name"`
	TaskQueue                string        `json:"task_queue"`
	Args                     []interface{} `json:"args"`
	Count                    int           `json:"count"` // 0 same as 1
	IndexAsArg               bool          `json:"index_as_arg"`
	ScheduleToCloseTimeoutMS int64         `json:"schedule_to_close_timeout_ms"`
	StartToCloseTimeoutMS    int64         `json:"start_to_close_timeout_ms"`
	ScheduleToStartTimeoutMS int64         `json:"schedule_to_start_timeout_ms"`
	CancelAfterMS            int64         `json:"cancel_after_ms"`
	WaitForCancellation      bool          `json:"wait_for_cancellation"`
	HeartbeatTimeoutMS       int64         `json:"heartbeat_timeout_ms"`
	RetryMaxAttempts         int           `json:"retry_max_attempts"` // 0 same as 1
	NonRetryableErrorTypes   []string      `json:"non_retryable_error_types"`
}
