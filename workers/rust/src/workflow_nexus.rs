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
        Err(error) => {
            let failure = error.operation_failure().ok_or_else(|| {
                wire::invalid(format!("cannot decode Nexus start failure: {error:?}"))
            })?;
            if cancelling && failure.as_cancelled().is_some() {
                return Ok(None);
            }
            return finish_failure(failure.failure().clone(), cancelling).map(|()| None);
        }
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

// SDK 1.1 returns a raw proto. Source SDKs expose these methods on typed errors.
#[allow(dead_code)]
trait LegacyNexusStartFailure {
    fn operation_failure(&self) -> Option<&Self>;
    fn failure(&self) -> &Failure;
    fn as_cancelled(&self) -> Option<()>;
}

impl LegacyNexusStartFailure for Failure {
    fn operation_failure(&self) -> Option<&Self> {
        Some(self)
    }

    fn failure(&self) -> &Failure {
        self
    }

    fn as_cancelled(&self) -> Option<()> {
        is_cancelled(self).then_some(())
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
