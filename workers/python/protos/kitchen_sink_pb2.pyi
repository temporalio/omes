from google.protobuf import duration_pb2 as _duration_pb2
from google.protobuf import empty_pb2 as _empty_pb2
from temporalio.api.common.v1 import message_pb2 as _message_pb2
from temporalio.api.failure.v1 import message_pb2 as _message_pb2_1
from temporalio.api.enums.v1 import workflow_pb2 as _workflow_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ParentClosePolicy(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PARENT_CLOSE_POLICY_UNSPECIFIED: _ClassVar[ParentClosePolicy]
    PARENT_CLOSE_POLICY_TERMINATE: _ClassVar[ParentClosePolicy]
    PARENT_CLOSE_POLICY_ABANDON: _ClassVar[ParentClosePolicy]
    PARENT_CLOSE_POLICY_REQUEST_CANCEL: _ClassVar[ParentClosePolicy]

class VersioningIntent(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    UNSPECIFIED: _ClassVar[VersioningIntent]
    COMPATIBLE: _ClassVar[VersioningIntent]
    DEFAULT: _ClassVar[VersioningIntent]

class ChildWorkflowCancellationType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CHILD_WF_ABANDON: _ClassVar[ChildWorkflowCancellationType]
    CHILD_WF_TRY_CANCEL: _ClassVar[ChildWorkflowCancellationType]
    CHILD_WF_WAIT_CANCELLATION_COMPLETED: _ClassVar[ChildWorkflowCancellationType]
    CHILD_WF_WAIT_CANCELLATION_REQUESTED: _ClassVar[ChildWorkflowCancellationType]

class ActivityCancellationType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TRY_CANCEL: _ClassVar[ActivityCancellationType]
    WAIT_CANCELLATION_COMPLETED: _ClassVar[ActivityCancellationType]
    ABANDON: _ClassVar[ActivityCancellationType]
PARENT_CLOSE_POLICY_UNSPECIFIED: ParentClosePolicy
PARENT_CLOSE_POLICY_TERMINATE: ParentClosePolicy
PARENT_CLOSE_POLICY_ABANDON: ParentClosePolicy
PARENT_CLOSE_POLICY_REQUEST_CANCEL: ParentClosePolicy
UNSPECIFIED: VersioningIntent
COMPATIBLE: VersioningIntent
DEFAULT: VersioningIntent
CHILD_WF_ABANDON: ChildWorkflowCancellationType
CHILD_WF_TRY_CANCEL: ChildWorkflowCancellationType
CHILD_WF_WAIT_CANCELLATION_COMPLETED: ChildWorkflowCancellationType
CHILD_WF_WAIT_CANCELLATION_REQUESTED: ChildWorkflowCancellationType
TRY_CANCEL: ActivityCancellationType
WAIT_CANCELLATION_COMPLETED: ActivityCancellationType
ABANDON: ActivityCancellationType

class TestInput(_message.Message):
    __slots__ = ("workflow_input", "client_sequence", "with_start_action")
    WORKFLOW_INPUT_FIELD_NUMBER: _ClassVar[int]
    CLIENT_SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    WITH_START_ACTION_FIELD_NUMBER: _ClassVar[int]
    workflow_input: WorkflowInput
    client_sequence: ClientSequence
    with_start_action: WithStartClientAction
    def __init__(self, workflow_input: _Optional[_Union[WorkflowInput, _Mapping]] = ..., client_sequence: _Optional[_Union[ClientSequence, _Mapping]] = ..., with_start_action: _Optional[_Union[WithStartClientAction, _Mapping]] = ...) -> None: ...

class ClientSequence(_message.Message):
    __slots__ = ("action_sets",)
    ACTION_SETS_FIELD_NUMBER: _ClassVar[int]
    action_sets: _containers.RepeatedCompositeFieldContainer[ClientActionSet]
    def __init__(self, action_sets: _Optional[_Iterable[_Union[ClientActionSet, _Mapping]]] = ...) -> None: ...

class ClientActionSet(_message.Message):
    __slots__ = ("actions", "concurrent", "wait_at_end", "wait_for_current_run_to_finish_at_end")
    ACTIONS_FIELD_NUMBER: _ClassVar[int]
    CONCURRENT_FIELD_NUMBER: _ClassVar[int]
    WAIT_AT_END_FIELD_NUMBER: _ClassVar[int]
    WAIT_FOR_CURRENT_RUN_TO_FINISH_AT_END_FIELD_NUMBER: _ClassVar[int]
    actions: _containers.RepeatedCompositeFieldContainer[ClientAction]
    concurrent: bool
    wait_at_end: _duration_pb2.Duration
    wait_for_current_run_to_finish_at_end: bool
    def __init__(self, actions: _Optional[_Iterable[_Union[ClientAction, _Mapping]]] = ..., concurrent: bool = ..., wait_at_end: _Optional[_Union[_duration_pb2.Duration, _Mapping]] = ..., wait_for_current_run_to_finish_at_end: bool = ...) -> None: ...

class WithStartClientAction(_message.Message):
    __slots__ = ("do_signal", "do_update")
    DO_SIGNAL_FIELD_NUMBER: _ClassVar[int]
    DO_UPDATE_FIELD_NUMBER: _ClassVar[int]
    do_signal: DoSignal
    do_update: DoUpdate
    def __init__(self, do_signal: _Optional[_Union[DoSignal, _Mapping]] = ..., do_update: _Optional[_Union[DoUpdate, _Mapping]] = ...) -> None: ...

class ClientAction(_message.Message):
    __slots__ = ("do_signal", "do_query", "do_update", "nested_actions")
    DO_SIGNAL_FIELD_NUMBER: _ClassVar[int]
    DO_QUERY_FIELD_NUMBER: _ClassVar[int]
    DO_UPDATE_FIELD_NUMBER: _ClassVar[int]
    NESTED_ACTIONS_FIELD_NUMBER: _ClassVar[int]
    do_signal: DoSignal
    do_query: DoQuery
    do_update: DoUpdate
    nested_actions: ClientActionSet
    def __init__(self, do_signal: _Optional[_Union[DoSignal, _Mapping]] = ..., do_query: _Optional[_Union[DoQuery, _Mapping]] = ..., do_update: _Optional[_Union[DoUpdate, _Mapping]] = ..., nested_actions: _Optional[_Union[ClientActionSet, _Mapping]] = ...) -> None: ...

class DoSignal(_message.Message):
    __slots__ = ("do_signal_actions", "custom", "with_start")
    class DoSignalActions(_message.Message):
        __slots__ = ("do_actions", "do_actions_in_main", "signal_id")
        DO_ACTIONS_FIELD_NUMBER: _ClassVar[int]
        DO_ACTIONS_IN_MAIN_FIELD_NUMBER: _ClassVar[int]
        SIGNAL_ID_FIELD_NUMBER: _ClassVar[int]
        do_actions: ActionSet
        do_actions_in_main: ActionSet
        signal_id: int
        def __init__(self, do_actions: _Optional[_Union[ActionSet, _Mapping]] = ..., do_actions_in_main: _Optional[_Union[ActionSet, _Mapping]] = ..., signal_id: _Optional[int] = ...) -> None: ...
    DO_SIGNAL_ACTIONS_FIELD_NUMBER: _ClassVar[int]
    CUSTOM_FIELD_NUMBER: _ClassVar[int]
    WITH_START_FIELD_NUMBER: _ClassVar[int]
    do_signal_actions: DoSignal.DoSignalActions
    custom: HandlerInvocation
    with_start: bool
    def __init__(self, do_signal_actions: _Optional[_Union[DoSignal.DoSignalActions, _Mapping]] = ..., custom: _Optional[_Union[HandlerInvocation, _Mapping]] = ..., with_start: bool = ...) -> None: ...

class DoQuery(_message.Message):
    __slots__ = ("report_state", "custom", "failure_expected")
    REPORT_STATE_FIELD_NUMBER: _ClassVar[int]
    CUSTOM_FIELD_NUMBER: _ClassVar[int]
    FAILURE_EXPECTED_FIELD_NUMBER: _ClassVar[int]
    report_state: _message_pb2.Payloads
    custom: HandlerInvocation
    failure_expected: bool
    def __init__(self, report_state: _Optional[_Union[_message_pb2.Payloads, _Mapping]] = ..., custom: _Optional[_Union[HandlerInvocation, _Mapping]] = ..., failure_expected: bool = ...) -> None: ...

class DoUpdate(_message.Message):
    __slots__ = ("do_actions", "custom", "with_start", "failure_expected")
    DO_ACTIONS_FIELD_NUMBER: _ClassVar[int]
    CUSTOM_FIELD_NUMBER: _ClassVar[int]
    WITH_START_FIELD_NUMBER: _ClassVar[int]
    FAILURE_EXPECTED_FIELD_NUMBER: _ClassVar[int]
    do_actions: DoActionsUpdate
    custom: HandlerInvocation
    with_start: bool
    failure_expected: bool
    def __init__(self, do_actions: _Optional[_Union[DoActionsUpdate, _Mapping]] = ..., custom: _Optional[_Union[HandlerInvocation, _Mapping]] = ..., with_start: bool = ..., failure_expected: bool = ...) -> None: ...

class DoActionsUpdate(_message.Message):
    __slots__ = ("do_actions", "reject_me")
    DO_ACTIONS_FIELD_NUMBER: _ClassVar[int]
    REJECT_ME_FIELD_NUMBER: _ClassVar[int]
    do_actions: ActionSet
    reject_me: _empty_pb2.Empty
    def __init__(self, do_actions: _Optional[_Union[ActionSet, _Mapping]] = ..., reject_me: _Optional[_Union[_empty_pb2.Empty, _Mapping]] = ...) -> None: ...

class HandlerInvocation(_message.Message):
    __slots__ = ("name", "args")
    NAME_FIELD_NUMBER: _ClassVar[int]
    ARGS_FIELD_NUMBER: _ClassVar[int]
    name: str
    args: _containers.RepeatedCompositeFieldContainer[_message_pb2.Payload]
    def __init__(self, name: _Optional[str] = ..., args: _Optional[_Iterable[_Union[_message_pb2.Payload, _Mapping]]] = ...) -> None: ...

class WorkflowState(_message.Message):
    __slots__ = ("kvs",)
    class KvsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    KVS_FIELD_NUMBER: _ClassVar[int]
    kvs: _containers.ScalarMap[str, str]
    def __init__(self, kvs: _Optional[_Mapping[str, str]] = ...) -> None: ...

class WorkflowInput(_message.Message):
    __slots__ = ("initial_actions", "expected_signal_count")
    INITIAL_ACTIONS_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_SIGNAL_COUNT_FIELD_NUMBER: _ClassVar[int]
    initial_actions: _containers.RepeatedCompositeFieldContainer[ActionSet]
    expected_signal_count: int
    def __init__(self, initial_actions: _Optional[_Iterable[_Union[ActionSet, _Mapping]]] = ..., expected_signal_count: _Optional[int] = ...) -> None: ...

class ActionSet(_message.Message):
    __slots__ = ("actions", "concurrent")
    ACTIONS_FIELD_NUMBER: _ClassVar[int]
    CONCURRENT_FIELD_NUMBER: _ClassVar[int]
    actions: _containers.RepeatedCompositeFieldContainer[Action]
    concurrent: bool
    def __init__(self, actions: _Optional[_Iterable[_Union[Action, _Mapping]]] = ..., concurrent: bool = ...) -> None: ...

class Action(_message.Message):
    __slots__ = ("timer", "exec_activity", "exec_child_workflow", "await_workflow_state", "send_signal", "cancel_workflow", "set_patch_marker", "upsert_search_attributes", "upsert_memo", "set_workflow_state", "return_result", "return_error", "continue_as_new", "nested_action_set", "nexus_operation")
    TIMER_FIELD_NUMBER: _ClassVar[int]
    EXEC_ACTIVITY_FIELD_NUMBER: _ClassVar[int]
    EXEC_CHILD_WORKFLOW_FIELD_NUMBER: _ClassVar[int]
    AWAIT_WORKFLOW_STATE_FIELD_NUMBER: _ClassVar[int]
    SEND_SIGNAL_FIELD_NUMBER: _ClassVar[int]
    CANCEL_WORKFLOW_FIELD_NUMBER: _ClassVar[int]
    SET_PATCH_MARKER_FIELD_NUMBER: _ClassVar[int]
    UPSERT_SEARCH_ATTRIBUTES_FIELD_NUMBER: _ClassVar[int]
    UPSERT_MEMO_FIELD_NUMBER: _ClassVar[int]
    SET_WORKFLOW_STATE_FIELD_NUMBER: _ClassVar[int]
    RETURN_RESULT_FIELD_NUMBER: _ClassVar[int]
    RETURN_ERROR_FIELD_NUMBER: _ClassVar[int]
    CONTINUE_AS_NEW_FIELD_NUMBER: _ClassVar[int]
    NESTED_ACTION_SET_FIELD_NUMBER: _ClassVar[int]
    NEXUS_OPERATION_FIELD_NUMBER: _ClassVar[int]
    timer: TimerAction
    exec_activity: ExecuteActivityAction
    exec_child_workflow: ExecuteChildWorkflowAction
    await_workflow_state: AwaitWorkflowState
    send_signal: SendSignalAction
    cancel_workflow: CancelWorkflowAction
    set_patch_marker: SetPatchMarkerAction
    upsert_search_attributes: UpsertSearchAttributesAction
    upsert_memo: UpsertMemoAction
    set_workflow_state: WorkflowState
    return_result: ReturnResultAction
    return_error: ReturnErrorAction
    continue_as_new: ContinueAsNewAction
    nested_action_set: ActionSet
    nexus_operation: ExecuteNexusOperation
    def __init__(self, timer: _Optional[_Union[TimerAction, _Mapping]] = ..., exec_activity: _Optional[_Union[ExecuteActivityAction, _Mapping]] = ..., exec_child_workflow: _Optional[_Union[ExecuteChildWorkflowAction, _Mapping]] = ..., await_workflow_state: _Optional[_Union[AwaitWorkflowState, _Mapping]] = ..., send_signal: _Optional[_Union[SendSignalAction, _Mapping]] = ..., cancel_workflow: _Optional[_Union[CancelWorkflowAction, _Mapping]] = ..., set_patch_marker: _Optional[_Union[SetPatchMarkerAction, _Mapping]] = ..., upsert_search_attributes: _Optional[_Union[UpsertSearchAttributesAction, _Mapping]] = ..., upsert_memo: _Optional[_Union[UpsertMemoAction, _Mapping]] = ..., set_workflow_state: _Optional[_Union[WorkflowState, _Mapping]] = ..., return_result: _Optional[_Union[ReturnResultAction, _Mapping]] = ..., return_error: _Optional[_Union[ReturnErrorAction, _Mapping]] = ..., continue_as_new: _Optional[_Union[ContinueAsNewAction, _Mapping]] = ..., nested_action_set: _Optional[_Union[ActionSet, _Mapping]] = ..., nexus_operation: _Optional[_Union[ExecuteNexusOperation, _Mapping]] = ...) -> None: ...

class AwaitableChoice(_message.Message):
    __slots__ = ("wait_finish", "abandon", "cancel_before_started", "cancel_after_started", "cancel_after_completed")
    WAIT_FINISH_FIELD_NUMBER: _ClassVar[int]
    ABANDON_FIELD_NUMBER: _ClassVar[int]
    CANCEL_BEFORE_STARTED_FIELD_NUMBER: _ClassVar[int]
    CANCEL_AFTER_STARTED_FIELD_NUMBER: _ClassVar[int]
    CANCEL_AFTER_COMPLETED_FIELD_NUMBER: _ClassVar[int]
    wait_finish: _empty_pb2.Empty
    abandon: _empty_pb2.Empty
    cancel_before_started: _empty_pb2.Empty
    cancel_after_started: _empty_pb2.Empty
    cancel_after_completed: _empty_pb2.Empty
    def __init__(self, wait_finish: _Optional[_Union[_empty_pb2.Empty, _Mapping]] = ..., abandon: _Optional[_Union[_empty_pb2.Empty, _Mapping]] = ..., cancel_before_started: _Optional[_Union[_empty_pb2.Empty, _Mapping]] = ..., cancel_after_started: _Optional[_Union[_empty_pb2.Empty, _Mapping]] = ..., cancel_after_completed: _Optional[_Union[_empty_pb2.Empty, _Mapping]] = ...) -> None: ...

class TimerAction(_message.Message):
    __slots__ = ("milliseconds", "awaitable_choice")
    MILLISECONDS_FIELD_NUMBER: _ClassVar[int]
    AWAITABLE_CHOICE_FIELD_NUMBER: _ClassVar[int]
    milliseconds: int
    awaitable_choice: AwaitableChoice
    def __init__(self, milliseconds: _Optional[int] = ..., awaitable_choice: _Optional[_Union[AwaitableChoice, _Mapping]] = ...) -> None: ...

class ExecuteActivityAction(_message.Message):
    __slots__ = ("generic", "delay", "noop", "resources", "payload", "client", "task_queue", "headers", "schedule_to_close_timeout", "schedule_to_start_timeout", "start_to_close_timeout", "heartbeat_timeout", "retry_policy", "is_local", "remote", "awaitable_choice", "priority", "fairness_key", "fairness_weight")
    class GenericActivity(_message.Message):
        __slots__ = ("type", "arguments")
        TYPE_FIELD_NUMBER: _ClassVar[int]
        ARGUMENTS_FIELD_NUMBER: _ClassVar[int]
        type: str
        arguments: _containers.RepeatedCompositeFieldContainer[_message_pb2.Payload]
        def __init__(self, type: _Optional[str] = ..., arguments: _Optional[_Iterable[_Union[_message_pb2.Payload, _Mapping]]] = ...) -> None: ...
    class ResourcesActivity(_message.Message):
        __slots__ = ("run_for", "bytes_to_allocate", "cpu_yield_every_n_iterations", "cpu_yield_for_ms")
        RUN_FOR_FIELD_NUMBER: _ClassVar[int]
        BYTES_TO_ALLOCATE_FIELD_NUMBER: _ClassVar[int]
        CPU_YIELD_EVERY_N_ITERATIONS_FIELD_NUMBER: _ClassVar[int]
        CPU_YIELD_FOR_MS_FIELD_NUMBER: _ClassVar[int]
        run_for: _duration_pb2.Duration
        bytes_to_allocate: int
        cpu_yield_every_n_iterations: int
        cpu_yield_for_ms: int
        def __init__(self, run_for: _Optional[_Union[_duration_pb2.Duration, _Mapping]] = ..., bytes_to_allocate: _Optional[int] = ..., cpu_yield_every_n_iterations: _Optional[int] = ..., cpu_yield_for_ms: _Optional[int] = ...) -> None: ...
    class PayloadActivity(_message.Message):
        __slots__ = ("bytes_to_receive", "bytes_to_return")
        BYTES_TO_RECEIVE_FIELD_NUMBER: _ClassVar[int]
        BYTES_TO_RETURN_FIELD_NUMBER: _ClassVar[int]
        bytes_to_receive: int
        bytes_to_return: int
        def __init__(self, bytes_to_receive: _Optional[int] = ..., bytes_to_return: _Optional[int] = ...) -> None: ...
    class ClientActivity(_message.Message):
        __slots__ = ("client_sequence",)
        CLIENT_SEQUENCE_FIELD_NUMBER: _ClassVar[int]
        client_sequence: ClientSequence
        def __init__(self, client_sequence: _Optional[_Union[ClientSequence, _Mapping]] = ...) -> None: ...
    class HeadersEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _message_pb2.Payload
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_message_pb2.Payload, _Mapping]] = ...) -> None: ...
    GENERIC_FIELD_NUMBER: _ClassVar[int]
    DELAY_FIELD_NUMBER: _ClassVar[int]
    NOOP_FIELD_NUMBER: _ClassVar[int]
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    CLIENT_FIELD_NUMBER: _ClassVar[int]
    TASK_QUEUE_FIELD_NUMBER: _ClassVar[int]
    HEADERS_FIELD_NUMBER: _ClassVar[int]
    SCHEDULE_TO_CLOSE_TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    SCHEDULE_TO_START_TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    START_TO_CLOSE_TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    HEARTBEAT_TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    RETRY_POLICY_FIELD_NUMBER: _ClassVar[int]
    IS_LOCAL_FIELD_NUMBER: _ClassVar[int]
    REMOTE_FIELD_NUMBER: _ClassVar[int]
    AWAITABLE_CHOICE_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    FAIRNESS_KEY_FIELD_NUMBER: _ClassVar[int]
    FAIRNESS_WEIGHT_FIELD_NUMBER: _ClassVar[int]
    generic: ExecuteActivityAction.GenericActivity
    delay: _duration_pb2.Duration
    noop: _empty_pb2.Empty
    resources: ExecuteActivityAction.ResourcesActivity
    payload: ExecuteActivityAction.PayloadActivity
    client: ExecuteActivityAction.ClientActivity
    task_queue: str
    headers: _containers.MessageMap[str, _message_pb2.Payload]
    schedule_to_close_timeout: _duration_pb2.Duration
    schedule_to_start_timeout: _duration_pb2.Duration
    start_to_close_timeout: _duration_pb2.Duration
    heartbeat_timeout: _duration_pb2.Duration
    retry_policy: _message_pb2.RetryPolicy
    is_local: _empty_pb2.Empty
    remote: RemoteActivityOptions
    awaitable_choice: AwaitableChoice
    priority: _message_pb2.Priority
    fairness_key: str
    fairness_weight: float
    def __init__(self, generic: _Optional[_Union[ExecuteActivityAction.GenericActivity, _Mapping]] = ..., delay: _Optional[_Union[_duration_pb2.Duration, _Mapping]] = ..., noop: _Optional[_Union[_empty_pb2.Empty, _Mapping]] = ..., resources: _Optional[_Union[ExecuteActivityAction.ResourcesActivity, _Mapping]] = ..., payload: _Optional[_Union[ExecuteActivityAction.PayloadActivity, _Mapping]] = ..., client: _Optional[_Union[ExecuteActivityAction.ClientActivity, _Mapping]] = ..., task_queue: _Optional[str] = ..., headers: _Optional[_Mapping[str, _message_pb2.Payload]] = ..., schedule_to_close_timeout: _Optional[_Union[_duration_pb2.Duration, _Mapping]] = ..., schedule_to_start_timeout: _Optional[_Union[_duration_pb2.Duration, _Mapping]] = ..., start_to_close_timeout: _Optional[_Union[_duration_pb2.Duration, _Mapping]] = ..., heartbeat_timeout: _Optional[_Union[_duration_pb2.Duration, _Mapping]] = ..., retry_policy: _Optional[_Union[_message_pb2.RetryPolicy, _Mapping]] = ..., is_local: _Optional[_Union[_empty_pb2.Empty, _Mapping]] = ..., remote: _Optional[_Union[RemoteActivityOptions, _Mapping]] = ..., awaitable_choice: _Optional[_Union[AwaitableChoice, _Mapping]] = ..., priority: _Optional[_Union[_message_pb2.Priority, _Mapping]] = ..., fairness_key: _Optional[str] = ..., fairness_weight: _Optional[float] = ...) -> None: ...

class ExecuteChildWorkflowAction(_message.Message):
    __slots__ = ("namespace", "workflow_id", "workflow_type", "task_queue", "input", "workflow_execution_timeout", "workflow_run_timeout", "workflow_task_timeout", "parent_close_policy", "workflow_id_reuse_policy", "retry_policy", "cron_schedule", "headers", "memo", "search_attributes", "cancellation_type", "versioning_intent", "awaitable_choice")
    class HeadersEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _message_pb2.Payload
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_message_pb2.Payload, _Mapping]] = ...) -> None: ...
    class MemoEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _message_pb2.Payload
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_message_pb2.Payload, _Mapping]] = ...) -> None: ...
    class SearchAttributesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _message_pb2.Payload
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_message_pb2.Payload, _Mapping]] = ...) -> None: ...
    NAMESPACE_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_TYPE_FIELD_NUMBER: _ClassVar[int]
    TASK_QUEUE_FIELD_NUMBER: _ClassVar[int]
    INPUT_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_EXECUTION_TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_RUN_TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_TASK_TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    PARENT_CLOSE_POLICY_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_REUSE_POLICY_FIELD_NUMBER: _ClassVar[int]
    RETRY_POLICY_FIELD_NUMBER: _ClassVar[int]
    CRON_SCHEDULE_FIELD_NUMBER: _ClassVar[int]
    HEADERS_FIELD_NUMBER: _ClassVar[int]
    MEMO_FIELD_NUMBER: _ClassVar[int]
    SEARCH_ATTRIBUTES_FIELD_NUMBER: _ClassVar[int]
    CANCELLATION_TYPE_FIELD_NUMBER: _ClassVar[int]
    VERSIONING_INTENT_FIELD_NUMBER: _ClassVar[int]
    AWAITABLE_CHOICE_FIELD_NUMBER: _ClassVar[int]
    namespace: str
    workflow_id: str
    workflow_type: str
    task_queue: str
    input: _containers.RepeatedCompositeFieldContainer[_message_pb2.Payload]
    workflow_execution_timeout: _duration_pb2.Duration
    workflow_run_timeout: _duration_pb2.Duration
    workflow_task_timeout: _duration_pb2.Duration
    parent_close_policy: ParentClosePolicy
    workflow_id_reuse_policy: _workflow_pb2.WorkflowIdReusePolicy
    retry_policy: _message_pb2.RetryPolicy
    cron_schedule: str
    headers: _containers.MessageMap[str, _message_pb2.Payload]
    memo: _containers.MessageMap[str, _message_pb2.Payload]
    search_attributes: _containers.MessageMap[str, _message_pb2.Payload]
    cancellation_type: ChildWorkflowCancellationType
    versioning_intent: VersioningIntent
    awaitable_choice: AwaitableChoice
    def __init__(self, namespace: _Optional[str] = ..., workflow_id: _Optional[str] = ..., workflow_type: _Optional[str] = ..., task_queue: _Optional[str] = ..., input: _Optional[_Iterable[_Union[_message_pb2.Payload, _Mapping]]] = ..., workflow_execution_timeout: _Optional[_Union[_duration_pb2.Duration, _Mapping]] = ..., workflow_run_timeout: _Optional[_Union[_duration_pb2.Duration, _Mapping]] = ..., workflow_task_timeout: _Optional[_Union[_duration_pb2.Duration, _Mapping]] = ..., parent_close_policy: _Optional[_Union[ParentClosePolicy, str]] = ..., workflow_id_reuse_policy: _Optional[_Union[_workflow_pb2.WorkflowIdReusePolicy, str]] = ..., retry_policy: _Optional[_Union[_message_pb2.RetryPolicy, _Mapping]] = ..., cron_schedule: _Optional[str] = ..., headers: _Optional[_Mapping[str, _message_pb2.Payload]] = ..., memo: _Optional[_Mapping[str, _message_pb2.Payload]] = ..., search_attributes: _Optional[_Mapping[str, _message_pb2.Payload]] = ..., cancellation_type: _Optional[_Union[ChildWorkflowCancellationType, str]] = ..., versioning_intent: _Optional[_Union[VersioningIntent, str]] = ..., awaitable_choice: _Optional[_Union[AwaitableChoice, _Mapping]] = ...) -> None: ...

class AwaitWorkflowState(_message.Message):
    __slots__ = ("key", "value")
    KEY_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    key: str
    value: str
    def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...

class SendSignalAction(_message.Message):
    __slots__ = ("workflow_id", "run_id", "signal_name", "args", "headers", "awaitable_choice")
    class HeadersEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _message_pb2.Payload
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_message_pb2.Payload, _Mapping]] = ...) -> None: ...
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SIGNAL_NAME_FIELD_NUMBER: _ClassVar[int]
    ARGS_FIELD_NUMBER: _ClassVar[int]
    HEADERS_FIELD_NUMBER: _ClassVar[int]
    AWAITABLE_CHOICE_FIELD_NUMBER: _ClassVar[int]
    workflow_id: str
    run_id: str
    signal_name: str
    args: _containers.RepeatedCompositeFieldContainer[_message_pb2.Payload]
    headers: _containers.MessageMap[str, _message_pb2.Payload]
    awaitable_choice: AwaitableChoice
    def __init__(self, workflow_id: _Optional[str] = ..., run_id: _Optional[str] = ..., signal_name: _Optional[str] = ..., args: _Optional[_Iterable[_Union[_message_pb2.Payload, _Mapping]]] = ..., headers: _Optional[_Mapping[str, _message_pb2.Payload]] = ..., awaitable_choice: _Optional[_Union[AwaitableChoice, _Mapping]] = ...) -> None: ...

class CancelWorkflowAction(_message.Message):
    __slots__ = ("workflow_id", "run_id")
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    workflow_id: str
    run_id: str
    def __init__(self, workflow_id: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class SetPatchMarkerAction(_message.Message):
    __slots__ = ("patch_id", "deprecated", "inner_action")
    PATCH_ID_FIELD_NUMBER: _ClassVar[int]
    DEPRECATED_FIELD_NUMBER: _ClassVar[int]
    INNER_ACTION_FIELD_NUMBER: _ClassVar[int]
    patch_id: str
    deprecated: bool
    inner_action: Action
    def __init__(self, patch_id: _Optional[str] = ..., deprecated: bool = ..., inner_action: _Optional[_Union[Action, _Mapping]] = ...) -> None: ...

class UpsertSearchAttributesAction(_message.Message):
    __slots__ = ("search_attributes",)
    class SearchAttributesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _message_pb2.Payload
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_message_pb2.Payload, _Mapping]] = ...) -> None: ...
    SEARCH_ATTRIBUTES_FIELD_NUMBER: _ClassVar[int]
    search_attributes: _containers.MessageMap[str, _message_pb2.Payload]
    def __init__(self, search_attributes: _Optional[_Mapping[str, _message_pb2.Payload]] = ...) -> None: ...

class UpsertMemoAction(_message.Message):
    __slots__ = ("upserted_memo",)
    UPSERTED_MEMO_FIELD_NUMBER: _ClassVar[int]
    upserted_memo: _message_pb2.Memo
    def __init__(self, upserted_memo: _Optional[_Union[_message_pb2.Memo, _Mapping]] = ...) -> None: ...

class ReturnResultAction(_message.Message):
    __slots__ = ("return_this",)
    RETURN_THIS_FIELD_NUMBER: _ClassVar[int]
    return_this: _message_pb2.Payload
    def __init__(self, return_this: _Optional[_Union[_message_pb2.Payload, _Mapping]] = ...) -> None: ...

class ReturnErrorAction(_message.Message):
    __slots__ = ("failure",)
    FAILURE_FIELD_NUMBER: _ClassVar[int]
    failure: _message_pb2_1.Failure
    def __init__(self, failure: _Optional[_Union[_message_pb2_1.Failure, _Mapping]] = ...) -> None: ...

class ContinueAsNewAction(_message.Message):
    __slots__ = ("workflow_type", "task_queue", "arguments", "workflow_run_timeout", "workflow_task_timeout", "memo", "headers", "search_attributes", "retry_policy", "versioning_intent")
    class MemoEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _message_pb2.Payload
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_message_pb2.Payload, _Mapping]] = ...) -> None: ...
    class HeadersEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _message_pb2.Payload
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_message_pb2.Payload, _Mapping]] = ...) -> None: ...
    class SearchAttributesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _message_pb2.Payload
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_message_pb2.Payload, _Mapping]] = ...) -> None: ...
    WORKFLOW_TYPE_FIELD_NUMBER: _ClassVar[int]
    TASK_QUEUE_FIELD_NUMBER: _ClassVar[int]
    ARGUMENTS_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_RUN_TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_TASK_TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    MEMO_FIELD_NUMBER: _ClassVar[int]
    HEADERS_FIELD_NUMBER: _ClassVar[int]
    SEARCH_ATTRIBUTES_FIELD_NUMBER: _ClassVar[int]
    RETRY_POLICY_FIELD_NUMBER: _ClassVar[int]
    VERSIONING_INTENT_FIELD_NUMBER: _ClassVar[int]
    workflow_type: str
    task_queue: str
    arguments: _containers.RepeatedCompositeFieldContainer[_message_pb2.Payload]
    workflow_run_timeout: _duration_pb2.Duration
    workflow_task_timeout: _duration_pb2.Duration
    memo: _containers.MessageMap[str, _message_pb2.Payload]
    headers: _containers.MessageMap[str, _message_pb2.Payload]
    search_attributes: _containers.MessageMap[str, _message_pb2.Payload]
    retry_policy: _message_pb2.RetryPolicy
    versioning_intent: VersioningIntent
    def __init__(self, workflow_type: _Optional[str] = ..., task_queue: _Optional[str] = ..., arguments: _Optional[_Iterable[_Union[_message_pb2.Payload, _Mapping]]] = ..., workflow_run_timeout: _Optional[_Union[_duration_pb2.Duration, _Mapping]] = ..., workflow_task_timeout: _Optional[_Union[_duration_pb2.Duration, _Mapping]] = ..., memo: _Optional[_Mapping[str, _message_pb2.Payload]] = ..., headers: _Optional[_Mapping[str, _message_pb2.Payload]] = ..., search_attributes: _Optional[_Mapping[str, _message_pb2.Payload]] = ..., retry_policy: _Optional[_Union[_message_pb2.RetryPolicy, _Mapping]] = ..., versioning_intent: _Optional[_Union[VersioningIntent, str]] = ...) -> None: ...

class RemoteActivityOptions(_message.Message):
    __slots__ = ("cancellation_type", "do_not_eagerly_execute", "versioning_intent")
    CANCELLATION_TYPE_FIELD_NUMBER: _ClassVar[int]
    DO_NOT_EAGERLY_EXECUTE_FIELD_NUMBER: _ClassVar[int]
    VERSIONING_INTENT_FIELD_NUMBER: _ClassVar[int]
    cancellation_type: ActivityCancellationType
    do_not_eagerly_execute: bool
    versioning_intent: VersioningIntent
    def __init__(self, cancellation_type: _Optional[_Union[ActivityCancellationType, str]] = ..., do_not_eagerly_execute: bool = ..., versioning_intent: _Optional[_Union[VersioningIntent, str]] = ...) -> None: ...

class ExecuteNexusOperation(_message.Message):
    __slots__ = ("endpoint", "operation", "input", "headers", "awaitable_choice", "expected_output")
    class HeadersEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    INPUT_FIELD_NUMBER: _ClassVar[int]
    HEADERS_FIELD_NUMBER: _ClassVar[int]
    AWAITABLE_CHOICE_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_OUTPUT_FIELD_NUMBER: _ClassVar[int]
    endpoint: str
    operation: str
    input: str
    headers: _containers.ScalarMap[str, str]
    awaitable_choice: AwaitableChoice
    expected_output: str
    def __init__(self, endpoint: _Optional[str] = ..., operation: _Optional[str] = ..., input: _Optional[str] = ..., headers: _Optional[_Mapping[str, str]] = ..., awaitable_choice: _Optional[_Union[AwaitableChoice, _Mapping]] = ..., expected_output: _Optional[str] = ...) -> None: ...
