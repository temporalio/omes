package shared

type KitchenSinkWorkflowParams struct {
	Actions      []*KitchenSinkAction `json:"actions"`
	ActionSignal string               `json:"action_signal"`
}

type KitchenSinkAction struct {
	Result          *ResultAction          `json:"result"`
	Error           *ErrorAction           `json:"error"`
	ContinueAsNew   *ContinueAsNewAction   `json:"continue_as_new"`
	Sleep           *SleepAction           `json:"sleep"`
	QueryHandler    *QueryHandlerAction    `json:"query_handler"`
	Signal          *SignalAction          `json:"signal"`
	ExecuteActivity *ExecuteActivityAction `json:"execute_activity"`
}

type ResultAction struct {
	Value interface{} `json:"value"`
	RunID bool        `json:"run_id"`
}

type ErrorAction struct {
	Message string      `json:"message"`
	Details interface{} `json:"details"`
	Attempt bool        `json:"attempt"`
}

type ContinueAsNewAction struct {
	WhileAboveZero int `json:"while_above_zero"`
}

type SleepAction struct {
	Millis int64 `json:"millis"`
}

type QueryHandlerAction struct {
	Name string `json:"name"`
}

type SignalAction struct {
	Name string `json:"name"`
}

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
