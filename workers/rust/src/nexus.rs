//! Kitchen Sink Nexus service name, input encoding and output checks for callers.
use crate::{proto as p, wire};
use temporalio_common::{error::ApplicationFailure, protos::temporal::api::common::v1::Payload};

pub const SERVICE_NAME: &str = "kitchen-sink";

/// A run's auto-created endpoint routes operations back to this worker's task
/// queue, but the SDK cannot serve Nexus operations, so they would never
/// complete. Fail fast instead of hanging the run.
pub fn check_endpoint(endpoint: &str) -> Result<(), ApplicationFailure> {
    if endpoint.starts_with("test-nexus-endpoint-") {
        return Err(wire::invalid(format!(
            "Nexus endpoint {endpoint} targets this Rust worker, which cannot handle Nexus \
             operations; pass --option nexus-enabled=false or use an external endpoint"
        )));
    }
    Ok(())
}

pub fn input(request: Option<p::NexusOperationRequest>) -> Payload {
    wire::encode(&request.unwrap_or_default())
        .payloads
        .remove(0)
}

/// Compares the entire payload, including metadata.
pub fn check_output(
    expected: Option<&Payload>,
    actual: &Payload,
) -> Result<(), ApplicationFailure> {
    if let Some(expected) = expected
        && expected != actual
    {
        return Err(ApplicationFailure::new(format!(
            "expected output {expected:?}, got {actual:?}"
        )));
    }
    Ok(())
}
