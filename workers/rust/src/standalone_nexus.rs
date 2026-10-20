//! Client-side standalone Nexus calls. Handler execution belongs to the endpoint's worker.
use crate::{nexus, proto as p, wire};
use std::time::Duration;
use temporalio_client::{NamespacedClient, grpc::WorkflowService};
use temporalio_common::{
    data_converters::{
        DataConverter, NexusSerializationContext, NoopDecodeHint, RawValue,
        SerializationContextData,
    },
    error::ApplicationFailure,
    protos::temporal::api::{
        common::v1::Payload,
        enums::v1::NexusOperationWaitStage,
        failure::v1::Failure,
        workflowservice::v1::{
            PollNexusOperationExecutionRequest, StartNexusOperationExecutionRequest,
            poll_nexus_operation_execution_response::Outcome,
        },
    },
};
use temporalio_sdk::activities::{ActivityContext, ActivityError};
use uuid::Uuid;

pub async fn execute(
    ctx: ActivityContext,
    action: p::DoStandaloneNexusOperation,
) -> Result<(), ActivityError> {
    let operation = action
        .operation
        .ok_or_else(|| wire::invalid("standalone Nexus operation requires operation"))?;
    if operation.awaitable_choice.as_ref().is_some_and(|choice| {
        !matches!(
            choice.condition,
            Some(p::awaitable_choice::Condition::WaitFinish(_))
        )
    }) {
        return Err(wire::invalid(
            "standalone Nexus operation only supports the wait_finish awaitable choice",
        )
        .into());
    }
    nexus::check_endpoint(&operation.endpoint)?;
    let mut client = ctx.client();
    let namespace = client.namespace();
    let operation_id = format!(
        "standalone-nexus-{}-{}",
        ctx.info().workflow_id.as_deref().unwrap_or_default(),
        Uuid::new_v4()
    );
    // Use the existing authenticated SDK client, not a fresh tonic connection.
    // The SDK supplies namespace routing headers and retries transient RPC errors.
    let request = StartNexusOperationExecutionRequest {
        namespace: namespace.clone(),
        identity: client.identity(),
        request_id: Uuid::new_v4().to_string(),
        operation_id: operation_id.clone(),
        endpoint: operation.endpoint,
        service: nexus::SERVICE_NAME.into(),
        operation: operation.operation,
        input: Some(nexus::input(operation.input)),
        schedule_to_close_timeout: Some(Duration::from_secs(90).try_into().unwrap()),
        ..Default::default()
    };
    // Cancelling the client activity stops waiting; it doesn't cancel the operation.
    let start = tokio::select! {
        result = client.start_nexus_operation_execution(tonic::Request::new(request)) =>
            result.map_err(rpc_error)?.into_inner(),
        _ = ctx.cancelled() => return Err(ActivityError::cancelled()),
    };
    let poll = PollNexusOperationExecutionRequest {
        namespace,
        operation_id,
        run_id: start.run_id,
        wait_stage: NexusOperationWaitStage::Closed as i32,
    };
    loop {
        let mut request = tonic::Request::new(poll.clone());
        // The SDK doesn't treat this RPC as a long poll, so its default deadline
        // would cut the poll short.
        request.set_timeout(Duration::from_secs(60));
        let result = tokio::select! {
            result = client.poll_nexus_operation_execution(request) =>
                result.map_err(rpc_error)?.into_inner(),
            _ = ctx.cancelled() => return Err(ActivityError::cancelled()),
        };
        match result.outcome {
            Some(Outcome::Result(payload)) => {
                return nexus::check_output(operation.expected_output.as_ref(), &payload)
                    .map_err(Into::into);
            }
            Some(Outcome::Failure(failure)) => {
                return Err(operation_error(client.data_converter(), failure));
            }
            // A long poll can return before closure. Preserve operation/run IDs.
            None => continue,
        }
    }
}

fn operation_error(converter: &DataConverter, failure: Failure) -> ActivityError {
    let cause = match converter.to_error(
        &SerializationContextData::Nexus(NexusSerializationContext::new()),
        failure,
        NoopDecodeHint,
    ) {
        Ok(cause) => cause,
        Err(error) => return error.into(),
    };
    // Retain the full server failure (details, encoded attributes and nested
    // causes), not only its outer handler-error message.
    ApplicationFailure::builder(format!("get standalone Nexus operation: {cause}"))
        .cause(Box::new(cause))
        .build()
        .into()
}

fn rpc_error(error: tonic::Status) -> ActivityError {
    let details = RawValue::new(vec![Payload {
        metadata: [("encoding".into(), b"binary/plain".to_vec())].into(),
        data: error.details().to_vec(),
        ..Default::default()
    }]);
    ApplicationFailure::builder(error)
        .details(details)
        .build()
        .into()
}
