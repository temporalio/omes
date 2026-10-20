//! Operator commands (pause, reset, update options) against a standalone activity.
//! Raw RPCs go through the SDK client, whose connection supplies authentication,
//! namespace routing, deadlines and retries.
use crate::{proto as p, standalone_activity, wire};
use std::time::Duration;
use temporalio_client::{Client, grpc::WorkflowService};
use temporalio_common::{
    error::ApplicationFailure,
    protos::temporal::api::{
        activity::v1::{ActivityExecutionInfo, ActivityOptions},
        enums::v1::{ActivityExecutionStatus, PendingActivityState},
        workflowservice::v1::{
            DescribeActivityExecutionRequest, PauseActivityExecutionRequest,
            ResetActivityExecutionRequest, UnpauseActivityExecutionRequest,
            UpdateActivityExecutionOptionsRequest,
        },
    },
};
use temporalio_sdk::activities::{ActivityContext, ActivityError};
use tonic::IntoRequest;
use uuid::Uuid;

const IDENTITY: &str = "omes";
const REASON: &str = "omes standalone activity operator command workload";
const STATE_WAIT_TIMEOUT: Duration = Duration::from_secs(60);

fn error(context: &str, error: impl std::fmt::Display) -> ApplicationFailure {
    ApplicationFailure::new(format!("{context}: {error}"))
}

pub(crate) async fn execute(
    ctx: ActivityContext,
    action: p::DoStandaloneActivityOperatorCommands,
) -> Result<(), ActivityError> {
    tokio::select! {
        result = run(ctx.client(), action) => result,
        _ = ctx.cancelled() => Err(ActivityError::cancelled()),
    }
}

async fn run(
    mut client: Client,
    action: p::DoStandaloneActivityOperatorCommands,
) -> Result<(), ActivityError> {
    use p::do_standalone_activity_operator_commands::CommandType;
    let activity = action.activity.as_ref().ok_or_else(|| {
        wire::invalid("DoStandaloneActivityOperatorCommands.activity is required")
    })?;
    let command = CommandType::try_from(action.command_type)
        .ok()
        .filter(|c| *c != CommandType::Unspecified)
        .ok_or_else(|| {
            wire::invalid(format!(
                "unsupported standalone activity operator command type {}",
                action.command_type
            ))
        })?;
    let id = format!("standalone-activity-operator-{}", Uuid::new_v4());
    let handle = standalone_activity::start(&client, activity, id).await?;
    let target = Target {
        namespace: client.options().namespace.clone(),
        activity_id: handle.activity_id().to_owned(),
        run_id: handle
            .run_id()
            .ok_or_else(|| ApplicationFailure::new("standalone activity start returned no run ID"))?
            .to_owned(),
    };
    let started = target
        .wait(&mut client, running)
        .await
        .map_err(|e| error("wait for standalone activity to start", e))?;
    match command {
        CommandType::Pause => {
            client
                .pause_activity_execution(
                    PauseActivityExecutionRequest {
                        namespace: target.namespace.clone(),
                        activity_id: target.activity_id.clone(),
                        run_id: target.run_id.clone(),
                        identity: IDENTITY.into(),
                        reason: REASON.into(),
                        resource_id: target.resource_id(),
                        request_id: Uuid::new_v4().to_string(),
                        ..Default::default()
                    }
                    .into_request(),
                )
                .await
                .map_err(|e| error("pause standalone activity", e))?;
            target
                .wait(&mut client, |info| {
                    info.status == ActivityExecutionStatus::Paused as i32
                        && info.run_state == PendingActivityState::Paused as i32
                })
                .await
                .map_err(|e| error("verify standalone activity paused", e))?;
            client
                .unpause_activity_execution(
                    UnpauseActivityExecutionRequest {
                        namespace: target.namespace.clone(),
                        activity_id: target.activity_id.clone(),
                        run_id: target.run_id.clone(),
                        identity: IDENTITY.into(),
                        reason: REASON.into(),
                        resource_id: target.resource_id(),
                        request_id: Uuid::new_v4().to_string(),
                        ..Default::default()
                    }
                    .into_request(),
                )
                .await
                .map_err(|e| error("unpause standalone activity", e))?;
            target
                .wait(&mut client, running)
                .await
                .map_err(|e| error("verify standalone activity unpaused", e))?;
        }
        CommandType::Reset => {
            let previous = started.last_started_time.ok_or_else(|| {
                ApplicationFailure::new(
                    "reset standalone activity: started activity is missing last started time",
                )
            })?;
            client
                .reset_activity_execution(
                    ResetActivityExecutionRequest {
                        namespace: target.namespace.clone(),
                        activity_id: target.activity_id.clone(),
                        run_id: target.run_id.clone(),
                        identity: IDENTITY.into(),
                        resource_id: target.resource_id(),
                        request_id: Uuid::new_v4().to_string(),
                        reset_heartbeat: true,
                        ..Default::default()
                    }
                    .into_request(),
                )
                .await
                .map_err(|e| error("reset standalone activity", e))?;
            target
                .wait(&mut client, |info| {
                    running(info)
                        && info.attempt == 1
                        && info.last_started_time.is_some_and(|time| {
                            (time.seconds, time.nanos) > (previous.seconds, previous.nanos)
                        })
                })
                .await
                .map_err(|e| error("verify standalone activity reset", e))?;
        }
        CommandType::Update => {
            let heartbeat_timeout = prost_wkt_types::Duration {
                seconds: 4,
                nanos: 0,
            };
            let response = client
                .update_activity_execution_options(
                    UpdateActivityExecutionOptionsRequest {
                        namespace: target.namespace.clone(),
                        activity_id: target.activity_id.clone(),
                        run_id: target.run_id.clone(),
                        identity: IDENTITY.into(),
                        resource_id: target.resource_id(),
                        request_id: Uuid::new_v4().to_string(),
                        activity_options: Some(ActivityOptions {
                            heartbeat_timeout: Some(heartbeat_timeout),
                            ..Default::default()
                        }),
                        update_mask: Some(prost_wkt_types::FieldMask {
                            paths: vec!["heartbeat_timeout".into()],
                        }),
                        ..Default::default()
                    }
                    .into_request(),
                )
                .await
                .map_err(|e| error("update standalone activity options", e))?
                .into_inner();
            let actual = response.activity_options.and_then(|o| o.heartbeat_timeout);
            if actual != Some(heartbeat_timeout) {
                return Err(ApplicationFailure::new(format!(
                    "update standalone activity options returned heartbeat timeout {actual:?}, want 4s"
                ))
                .into());
            }
        }
        CommandType::Unspecified => unreachable!("validated before starting the activity"),
    }
    handle
        .result()
        .await
        .map_err(standalone_activity::result_error)?;
    Ok(())
}

fn running(info: &ActivityExecutionInfo) -> bool {
    info.status == ActivityExecutionStatus::Running as i32
        && info.run_state == PendingActivityState::Started as i32
}

#[derive(Debug)]
struct Target {
    namespace: String,
    activity_id: String,
    run_id: String,
}

impl Target {
    fn resource_id(&self) -> String {
        format!("activity:{}", self.activity_id)
    }

    async fn wait(
        &self,
        client: &mut Client,
        matches: impl Fn(&ActivityExecutionInfo) -> bool,
    ) -> Result<ActivityExecutionInfo, ApplicationFailure> {
        let deadline = tokio::time::Instant::now() + STATE_WAIT_TIMEOUT;
        let mut token = vec![];
        let mut last_observed = None;
        loop {
            // Bound the whole retrying RPC, not just one transport attempt.
            let mut request = DescribeActivityExecutionRequest {
                namespace: self.namespace.clone(),
                activity_id: self.activity_id.clone(),
                run_id: self.run_id.clone(),
                long_poll_token: token,
                ..Default::default()
            }
            .into_request();
            request.set_timeout(deadline.saturating_duration_since(tokio::time::Instant::now()));
            let response = tokio::time::timeout_at(
                deadline,
                client.describe_activity_execution(request),
            )
            .await
            .map_err(|e| e.to_string())
            .and_then(|result| result.map_err(|e| e.to_string()))
            .map_err(|e| {
                error(
                    &format!(
                        "describe standalone activity {self:?}, last observed {last_observed:?}"
                    ),
                    e,
                )
            })?
            .into_inner();
            let info = response.info.ok_or_else(|| {
                ApplicationFailure::new(format!(
                    "DescribeActivityExecution returned no activity info for {self:?}"
                ))
            })?;
            if matches(&info) {
                return Ok(info);
            }
            token = response.long_poll_token;
            if token.is_empty() {
                return Err(ApplicationFailure::new(format!(
                    "standalone activity {self:?} reached terminal state before expected state: {info:?}"
                )));
            }
            last_observed = Some(info);
        }
    }
}
