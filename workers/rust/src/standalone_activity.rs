//! Standalone activity options shared by client actions and operator commands.
//! Unlike workflow commands, these always wait for completion. Local execution,
//! workflow cancellation modes, eager dispatch and versioning intent do not apply.
use crate::{kitchen_sink, proto as p, wire};
use temporalio_client::{
    ActivityHandle, ActivityStartOptions, Client, errors::ActivityResultError,
};
use temporalio_common::{ActivityCloseTimeouts, UntypedActivity, error::ApplicationFailure};
use temporalio_sdk::activities::ActivityError;
use uuid::Uuid;

pub(crate) fn result_error(error: ActivityResultError) -> ActivityError {
    let message = match error {
        // Keep the remote failure as the cause; its retry metadata belongs to the
        // standalone activity, not the enclosing client activity.
        ActivityResultError::ActivityFailed(cause) => {
            return ApplicationFailure::builder(format!("get standalone activity: {cause}"))
                .cause(Box::new(cause))
                .build()
                .into();
        }
        // Not a cancellation of the enclosing client activity, so don't report one.
        ActivityResultError::Cancelled { .. } => "activity canceled".to_owned(),
        other => other.to_string(),
    };
    ApplicationFailure::new(format!("get standalone activity: {message}")).into()
}

/// All four timeouts, retries and priority/fairness are supported.
/// A missing task queue or close timeout is invalid (there is no workflow default).
pub(crate) fn options(
    activity: &p::ExecuteActivityAction,
    id: String,
) -> Result<ActivityStartOptions, ApplicationFailure> {
    use p::{awaitable_choice::Condition, execute_activity_action::Locality};
    if activity.task_queue.is_empty() {
        return Err(wire::invalid("standalone activity task_queue is required"));
    }
    match activity.locality {
        Some(Locality::IsLocal(_)) => {
            wire::unsupported(true, "standalone activity local execution")?
        }
        Some(Locality::Remote(remote)) => wire::unsupported(
            remote.cancellation_type != 0
                || remote.do_not_eagerly_execute
                || remote.versioning_intent != 0,
            "standalone activity workflow remote options",
        )?,
        None => {}
    }
    wire::unsupported(!activity.headers.is_empty(), "headers")?;
    wire::unsupported(
        !matches!(
            activity.awaitable_choice.and_then(|c| c.condition),
            None | Some(Condition::WaitFinish(_))
        ),
        "standalone activity awaitable_choice other than wait_finish",
    )?;
    // A zero timeout means unset.
    let timeout =
        |d| Ok::<_, ApplicationFailure>(kitchen_sink::duration(d)?.filter(|d| !d.is_zero()));
    let close = match (
        timeout(activity.schedule_to_close_timeout)?,
        timeout(activity.start_to_close_timeout)?,
    ) {
        (Some(schedule_to_close), Some(start_to_close)) => {
            ActivityCloseTimeouts::ScheduleAndStartToClose {
                schedule_to_close,
                start_to_close,
            }
        }
        (Some(d), None) => ActivityCloseTimeouts::ScheduleToClose(d),
        (None, Some(d)) => ActivityCloseTimeouts::StartToClose(d),
        _ => {
            return Err(wire::invalid(
                "standalone activity requires a close timeout",
            ));
        }
    };
    Ok(ActivityStartOptions::new(&activity.task_queue, id, close)
        .maybe_schedule_to_start_timeout(timeout(activity.schedule_to_start_timeout)?)
        .maybe_heartbeat_timeout(timeout(activity.heartbeat_timeout)?)
        .maybe_retry_policy(activity.retry_policy.clone())
        .priority(kitchen_sink::activity_priority(activity))
        .build())
}

pub(crate) async fn start(
    client: &Client,
    activity: &p::ExecuteActivityAction,
    id: String,
) -> Result<ActivityHandle<Client, UntypedActivity>, ActivityError> {
    let (definition, input) = kitchen_sink::activity_input(activity)?;
    let options = options(activity, id)?;
    client
        .start_activity(definition, input, options)
        .await
        .map_err(|e| {
            ApplicationFailure::new(format!("failed to start standalone activity: {e}")).into()
        })
}

/// Fresh even for retries/concurrent client actions; never derived from the enclosing activity ID.
pub(crate) fn fresh_id(workflow_id: &str) -> String {
    format!("standalone-activity-{workflow_id}-{}", Uuid::new_v4())
}
