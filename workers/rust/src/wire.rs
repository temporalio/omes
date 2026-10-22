//! Kitchen Sink messages travel as ProtoJSON (`json/protobuf` payloads).
use prost::{Message, Name};
use prost_reflect::{DescriptorPool, DynamicMessage};
use serde::Serialize;
use std::sync::{
    OnceLock,
    atomic::{AtomicBool, Ordering},
};
use temporalio_common::{
    data_converters::RawValue, error::ApplicationFailure,
    protos::temporal::api::common::v1::Payload,
};

fn descriptor<T: Name>() -> prost_reflect::MessageDescriptor {
    static POOL: OnceLock<DescriptorPool> = OnceLock::new();
    POOL.get_or_init(|| {
        DescriptorPool::decode(
            include_bytes!(concat!(env!("OUT_DIR"), "/kitchen_sink.bin")).as_slice(),
        )
        .expect("generated schema descriptor")
    })
    .get_message_by_name(&T::full_name())
    .expect("generated message descriptor")
}

pub fn decode<T: Name + Message + Default>(raw: &RawValue) -> Result<T, ApplicationFailure> {
    let [payload] = raw.payloads.as_slice() else {
        return Err(invalid("expected exactly one protobuf payload"));
    };
    if payload.metadata.get("encoding").map(Vec::as_slice) != Some(b"json/protobuf")
        || payload.metadata.get("messageType").map(Vec::as_slice) != Some(T::full_name().as_bytes())
    {
        return Err(invalid(format!(
            "expected json/protobuf {}",
            T::full_name()
        )));
    }
    let mut deserializer = serde_json::Deserializer::from_slice(&payload.data);
    let message = DynamicMessage::deserialize(descriptor::<T>(), &mut deserializer)
        .map_err(|e| invalid(format!("invalid ProtoJSON: {e}")))?;
    deserializer
        .end()
        .map_err(|e| invalid(format!("invalid ProtoJSON: {e}")))?;
    T::decode(message.encode_to_vec().as_slice())
        .map_err(|e| invalid(format!("invalid protobuf: {e}")))
}

pub fn encode<T: Name + Message>(value: &T) -> RawValue {
    let message = DynamicMessage::decode(descriptor::<T>(), value.encode_to_vec().as_slice())
        .expect("generated message matches schema");
    // API Payload metadata and reflection map fields may use HashMap. Canonicalize
    // JSON objects so replay never depends on a randomized map iteration order.
    let mut json = serde_json::to_value(&message).expect("generated protobuf JSON serialization");
    json.sort_all_objects();
    RawValue::new(vec![Payload {
        metadata: [
            ("encoding".into(), b"json/protobuf".to_vec()),
            ("messageType".into(), T::full_name().into_bytes()),
        ]
        .into(),
        data: serde_json::to_vec(&json).expect("generated protobuf JSON serialization"),
        ..Default::default()
    }])
}

pub fn json<T: Serialize>(value: &T) -> Payload {
    Payload {
        metadata: [("encoding".into(), b"json/plain".to_vec())].into(),
        data: serde_json::to_vec(value).expect("JSON serialization"),
        ..Default::default()
    }
}

pub fn invalid(message: impl Into<String>) -> ApplicationFailure {
    ApplicationFailure::non_retryable(message.into())
}

static ERR_ON_UNIMPLEMENTED: AtomicBool = AtomicBool::new(false);

pub fn set_err_on_unimplemented(value: bool) {
    ERR_ON_UNIMPLEMENTED.store(value, Ordering::Relaxed);
}

/// Skips an unsupported action or option with a warning, or fails when
/// `--err-on-unimplemented` is set.
pub fn unsupported(condition: bool, feature: &str) -> Result<(), ApplicationFailure> {
    if !condition {
        return Ok(());
    }
    if ERR_ON_UNIMPLEMENTED.load(Ordering::Relaxed) {
        return Err(invalid(format!("{feature} is not supported")));
    }
    tracing::warn!(feature, "Skipping unsupported feature");
    Ok(())
}
