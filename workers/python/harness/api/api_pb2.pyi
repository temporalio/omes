from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ConnectOptions(_message.Message):
    __slots__ = ("namespace", "server_address", "auth_header", "enable_tls", "tls_cert_path", "tls_key_path", "tls_server_name", "disable_host_verification")
    NAMESPACE_FIELD_NUMBER: _ClassVar[int]
    SERVER_ADDRESS_FIELD_NUMBER: _ClassVar[int]
    AUTH_HEADER_FIELD_NUMBER: _ClassVar[int]
    ENABLE_TLS_FIELD_NUMBER: _ClassVar[int]
    TLS_CERT_PATH_FIELD_NUMBER: _ClassVar[int]
    TLS_KEY_PATH_FIELD_NUMBER: _ClassVar[int]
    TLS_SERVER_NAME_FIELD_NUMBER: _ClassVar[int]
    DISABLE_HOST_VERIFICATION_FIELD_NUMBER: _ClassVar[int]
    namespace: str
    server_address: str
    auth_header: str
    enable_tls: bool
    tls_cert_path: str
    tls_key_path: str
    tls_server_name: str
    disable_host_verification: bool
    def __init__(self, namespace: _Optional[str] = ..., server_address: _Optional[str] = ..., auth_header: _Optional[str] = ..., enable_tls: bool = ..., tls_cert_path: _Optional[str] = ..., tls_key_path: _Optional[str] = ..., tls_server_name: _Optional[str] = ..., disable_host_verification: bool = ...) -> None: ...

class InitRequest(_message.Message):
    __slots__ = ("execution_id", "run_id", "task_queue", "connect_options", "config_json", "register_search_attributes")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    TASK_QUEUE_FIELD_NUMBER: _ClassVar[int]
    CONNECT_OPTIONS_FIELD_NUMBER: _ClassVar[int]
    CONFIG_JSON_FIELD_NUMBER: _ClassVar[int]
    REGISTER_SEARCH_ATTRIBUTES_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    run_id: str
    task_queue: str
    connect_options: ConnectOptions
    config_json: bytes
    register_search_attributes: bool
    def __init__(self, execution_id: _Optional[str] = ..., run_id: _Optional[str] = ..., task_queue: _Optional[str] = ..., connect_options: _Optional[_Union[ConnectOptions, _Mapping]] = ..., config_json: _Optional[bytes] = ..., register_search_attributes: bool = ...) -> None: ...

class InitResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ExecuteRequest(_message.Message):
    __slots__ = ("iteration", "payload")
    ITERATION_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    iteration: int
    payload: bytes
    def __init__(self, iteration: _Optional[int] = ..., payload: _Optional[bytes] = ...) -> None: ...

class ExecuteResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
