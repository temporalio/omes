//! Workflow-issued operations against an existing, externally served endpoint.
use crate::{kitchen_sink::KitchenSink, nexus, proto as p, wire};
use futures::{FutureExt, future::LocalBoxFuture};
use temporalio_common::{
    data_converters::{
        DefaultFailureConverter, FailureConverter, PayloadConverter, SerializationContextData,
    },
    error::ApplicationFailure,
    protos::{
        coresdk::nexus::nexus_operation_result::Status,
        temporal::api::failure::v1::{Failure, failure::FailureInfo},
    },
};
use temporalio_sdk::{CancellableFuture, NexusOperationOptions, WorkflowContext, WorkflowResult};

type Pending = LocalBoxFuture<'static, WorkflowResult<()>>;

pub async fn execute(
    ctx: WorkflowContext<KitchenSink>,
    operation: p::ExecuteNexusOperation,
) -> WorkflowResult<Option<Pending>> {
    use p::awaitable_choice::Condition;
    nexus::check_endpoint(&operation.endpoint)?;
    let choice = operation.awaitable_choice.and_then(|c| c.condition);
    // The action has no timeout fields, so impose no caller-side timeout.
    let start = ctx.start_nexus_operation(
        NexusOperationOptions::builder()
            .endpoint(operation.endpoint)
            .service(nexus::SERVICE_NAME)
            .operation(operation.operation)
            .input(nexus::input(operation.input))
            .build(),
    );
    if matches!(choice, Some(Condition::Abandon(_))) {
        drop(start);
        return Ok(None);
    }
    let cancelling = matches!(
        choice,
        Some(Condition::CancelBeforeStarted(_) | Condition::CancelAfterStarted(_))
    );
    if matches!(choice, Some(Condition::CancelBeforeStarted(_))) {
        start.cancel();
    }
    let started = match start.await {
        Ok(started) => started,
        Err(failure) => return finish_failure(failure, cancelling).map(|()| None),
    };
    if matches!(choice, Some(Condition::CancelAfterStarted(_))) {
        started.cancel();
    }
    let wait_started = matches!(choice, Some(Condition::WaitStarted(_)));
    let completion = async move {
        let result = match started.result().await.status {
            Some(Status::Completed(payload)) => {
                // Only a direct wait for completion checks the expected output;
                // pending and cancelled operations just need to finish.
                nexus::check_output(
                    if wait_started || cancelling {
                        None
                    } else {
                        operation.expected_output.as_ref()
                    },
                    &payload,
                )
                .map_err(Into::into)
            }
            Some(Status::Failed(f) | Status::Cancelled(f) | Status::TimedOut(f)) => {
                finish_failure(f, cancelling)
            }
            None => Err(wire::invalid("Nexus operation result has no status").into()),
        };
        if matches!(choice, Some(Condition::CancelAfterCompleted(_))) {
            started.cancel();
        }
        result
    }
    .boxed_local();
    if wait_started {
        Ok(Some(completion))
    } else {
        completion.await.map(|()| None)
    }
}

fn is_cancelled(failure: &Failure) -> bool {
    matches!(
        failure.failure_info,
        Some(FailureInfo::CanceledFailureInfo(_))
    ) || failure.cause.as_deref().is_some_and(is_cancelled)
}

fn finish_failure(failure: Failure, intentional_cancel: bool) -> WorkflowResult<()> {
    if intentional_cancel && is_cancelled(&failure) {
        return Ok(());
    }
    // The SDK exposes raw Nexus failure protos but has no workflow Nexus error
    // type, so keep the entire remote failure as the cause.
    let message = failure.message.clone();
    let cause = DefaultFailureConverter::default()
        .to_error(
            failure,
            &PayloadConverter::default(),
            &SerializationContextData::None,
        )
        .map_err(|error| wire::invalid(format!("cannot decode Nexus failure: {error}")))?;
    Err(ApplicationFailure::builder(message)
        .cause(Box::new(cause))
        .build()
        .into())
}
