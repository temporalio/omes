use crate::{kitchen_sink::duration, proto as p, wire};
use futures::{FutureExt, future::BoxFuture};
use std::{
    sync::{Arc, Mutex},
    time::Duration,
};
use temporalio_client::{
    Client, UntypedQuery, UntypedSignal, UntypedUpdate, WorkflowFetchHistoryOptions,
    WorkflowGetResultOptions, WorkflowHandle, WorkflowIdConflictPolicy, WorkflowStartOptions,
    WorkflowUpdateWithStartOptions,
};
use temporalio_common::{
    UntypedWorkflow, data_converters::RawValue, error::ApplicationFailure,
    protos::temporal::api::common::v1::Payload,
};
use temporalio_macros::activities;
use temporalio_sdk::activities::{ActivityContext, ActivityError};

pub struct Activities;

// Registration uses the macro's activity collection, not its generated typed
// call constants (Kitchen Sink schedules activities by their wire names).
#[activities]
impl Activities {
    #[activity(name = "noop")]
    #[allow(dead_code)]
    pub async fn noop(_ctx: ActivityContext) -> Result<(), ActivityError> {
        Ok(())
    }

    #[activity(name = "delay")]
    #[allow(dead_code)]
    pub async fn delay(ctx: ActivityContext, raw: RawValue) -> Result<(), ActivityError> {
        let [payload] = raw.payloads.as_slice() else {
            return Err(wire::invalid("delay requires nanoseconds").into());
        };
        let nanos: u64 =
            serde_json::from_slice(&payload.data).map_err(|e| wire::invalid(e.to_string()))?;
        sleep(&ctx, Duration::from_nanos(nanos)).await
    }

    #[activity(name = "resources")]
    #[allow(dead_code)]
    pub async fn resources(ctx: ActivityContext, raw: RawValue) -> Result<(), ActivityError> {
        crate::resource_activity::run(ctx, wire::decode(&raw)?).await
    }

    #[activity(name = "payload")]
    #[allow(dead_code)]
    pub async fn payload(_ctx: ActivityContext, raw: RawValue) -> Result<RawValue, ActivityError> {
        let [_, size] = raw.payloads.as_slice() else {
            return Err(wire::invalid("payload requires bytes and return size").into());
        };
        let size: usize =
            serde_json::from_slice(&size.data).map_err(|e| wire::invalid(e.to_string()))?;
        Ok(RawValue::new(vec![bytes_payload(size)]))
    }

    #[activity(name = "retryable_error")]
    #[allow(dead_code)]
    pub async fn retryable_error(ctx: ActivityContext, raw: RawValue) -> Result<(), ActivityError> {
        let config: p::execute_activity_action::RetryableErrorActivity = wire::decode(&raw)?;
        if i64::from(ctx.info().attempt) <= i64::from(config.fail_attempts) {
            return Err(ApplicationFailure::builder("retryable error")
                .type_name("RetryableError".into())
                .build()
                .into());
        }
        Ok(())
    }

    #[activity(name = "timeout")]
    #[allow(dead_code)]
    pub async fn timeout(ctx: ActivityContext, raw: RawValue) -> Result<(), ActivityError> {
        let config: p::execute_activity_action::TimeoutActivity = wire::decode(&raw)?;
        if i64::from(ctx.info().attempt) <= i64::from(config.fail_attempts) {
            ctx.cancelled().await;
            return Err(ActivityError::cancelled());
        }
        sleep(&ctx, duration(config.success_duration)?.unwrap_or_default()).await
    }

    #[activity(name = "heartbeat")]
    #[allow(dead_code)]
    pub async fn heartbeat(ctx: ActivityContext, raw: RawValue) -> Result<(), ActivityError> {
        let config: p::execute_activity_action::HeartbeatTimeoutActivity = wire::decode(&raw)?;
        if i64::from(ctx.info().attempt) <= i64::from(config.fail_attempts) {
            ctx.cancelled().await;
            return Err(ActivityError::cancelled());
        }
        let run_for = duration(config.success_duration)?.unwrap_or_default();
        let interval = duration(config.heartbeat_interval)?.unwrap_or_default();
        if interval.is_zero() {
            sleep(&ctx, run_for).await?;
            ctx.record_heartbeat(()).await?;
            return Ok(());
        }
        let deadline = tokio::time::sleep(run_for);
        tokio::pin!(deadline);
        let mut ticker = tokio::time::interval(interval);
        loop {
            tokio::select! {
                _ = &mut deadline => return Ok(()),
                _ = ticker.tick() => ctx.record_heartbeat(()).await?,
                _ = ctx.cancelled() => return Err(ActivityError::cancelled()),
            }
        }
    }

    #[activity(name = "client")]
    #[allow(dead_code)]
    pub async fn client(ctx: ActivityContext, raw: RawValue) -> Result<(), ActivityError> {
        let config: p::execute_activity_action::ClientActivity = wire::decode(&raw)?;
        let ctx = ClientActionContext::new(ctx);
        for set in config.client_sequence.unwrap_or_default().action_sets {
            client_action_set(ctx.clone(), set).await?;
        }
        Ok(())
    }
}

async fn sleep(ctx: &ActivityContext, duration: Duration) -> Result<(), ActivityError> {
    tokio::select! {
        _ = tokio::time::sleep(duration) => Ok(()),
        _ = ctx.cancelled() => Err(ActivityError::cancelled()),
    }
}

/// A seeded xorshift stream keeps payloads incompressible without workflow OS randomness.
pub fn bytes_payload(size: usize) -> Payload {
    let mut state = (size as u64).wrapping_add(0x9e3779b97f4a7c15);
    let data = (0..size)
        .map(|_| {
            state ^= state << 13;
            state ^= state >> 7;
            state ^= state << 17;
            state as u8
        })
        .collect();
    Payload {
        metadata: [("encoding".into(), b"binary/plain".to_vec())].into(),
        data,
        ..Default::default()
    }
}

/// Nested and concurrent action sets share a run cursor, but each RPC gets a
/// snapshot. Never hold the cursor lock across a network await.
#[derive(Clone)]
struct ClientActionContext {
    activity: ActivityContext,
    run_id: Arc<Mutex<Option<String>>>,
}

impl ClientActionContext {
    fn new(activity: ActivityContext) -> Self {
        Self {
            run_id: Arc::new(Mutex::new(activity.info().workflow_run_id.clone())),
            activity,
        }
    }

    /// Client actions target the latest run; only waiting for the current run
    /// to finish uses the tracked run cursor.
    fn handle(
        &self,
        run_id: Option<String>,
    ) -> Result<WorkflowHandle<Client, UntypedWorkflow>, ActivityError> {
        let original = self
            .activity
            .workflow_handle::<UntypedWorkflow>()
            .ok_or_else(|| wire::invalid("client activity requires a workflow"))?;
        let mut info = original.info().clone();
        info.run_id = run_id;
        // The activity's original run is not the first run of a newly bound handle.
        info.first_execution_run_id = None;
        Ok(WorkflowHandle::new(self.activity.client(), info))
    }

    async fn wait_for_current_run(&self) -> Result<(), ActivityError> {
        use temporalio_client::{HistoryEventFilterType, errors::WorkflowGetResultError};
        use temporalio_common::protos::temporal::api::history::v1::history_event::Attributes;

        let handle = self.handle(self.run_id.lock().unwrap().clone())?;
        let waited_run = handle.info().run_id.clone();
        let wait = async {
            match handle
                .get_result(
                    WorkflowGetResultOptions::builder()
                        .follow_runs(false)
                        .build(),
                )
                .await
            {
                Ok(_) => Ok(()),
                Err(WorkflowGetResultError::ContinuedAsNew) => {
                    // GetResult deliberately discards the successor ID. Read the
                    // pinned close event instead of describing "latest", which
                    // could skip an intervening run when continuations are fast.
                    let events = handle
                        .fetch_history(
                            WorkflowFetchHistoryOptions::builder()
                                .event_filter_type(HistoryEventFilterType::CloseEvent)
                                .build(),
                        )
                        .into_events()
                        .await
                        .map_err(activity_rpc_error)?;
                    let next = events
                        .into_iter()
                        .find_map(|event| match event.attributes {
                            Some(Attributes::WorkflowExecutionContinuedAsNewEventAttributes(
                                attrs,
                            )) => Some(attrs.new_execution_run_id),
                            _ => None,
                        })
                        .filter(|id| !id.is_empty())
                        .ok_or_else(|| {
                            wire::invalid("continue-as-new close event has no successor run")
                        })?;
                    advance_run(&self.run_id, &waited_run, next);
                    Ok(())
                }
                Err(error) => Err(activity_rpc_error(error)),
            }
        };
        // Remote cancellation is delivered through heartbeats. Only heartbeat
        // when requested: an abandoned activity may intentionally outlive its run.
        let heartbeat = async {
            let Some(timeout) = self
                .activity
                .info()
                .heartbeat_timeout
                .filter(|d| !d.is_zero())
            else {
                return futures::future::pending::<Result<(), ActivityError>>().await;
            };
            let mut ticker = tokio::time::interval((timeout / 2).max(Duration::from_millis(1)));
            loop {
                ticker.tick().await;
                self.activity.record_heartbeat(()).await?;
            }
        };
        tokio::select! {
            result = wait => result,
            result = heartbeat => result,
            _ = self.activity.cancelled() => Err(ActivityError::cancelled()),
        }
    }
}

fn advance_run(cursor: &Mutex<Option<String>>, waited: &Option<String>, next: String) {
    let mut current = cursor.lock().unwrap();
    // A slower concurrent waiter must not rewind a cursor advanced by a sibling.
    if *current == *waited {
        *current = Some(next);
    }
}

fn client_action_set(
    ctx: ClientActionContext,
    set: p::ClientActionSet,
) -> BoxFuture<'static, Result<(), ActivityError>> {
    async move {
        if set.concurrent {
            futures::future::try_join_all(
                set.actions
                    .into_iter()
                    .map(|a| client_action(ctx.clone(), a)),
            )
            .await?;
        } else {
            for action in set.actions {
                client_action(ctx.clone(), action).await?;
            }
        }
        if let Some(wait) = duration(set.wait_at_end)? {
            sleep(&ctx.activity, wait).await?;
        }
        if set.wait_for_current_run_to_finish_at_end {
            ctx.wait_for_current_run().await?;
        }
        Ok(())
    }
    .boxed()
}

fn client_action(
    ctx: ClientActionContext,
    action: p::ClientAction,
) -> BoxFuture<'static, Result<(), ActivityError>> {
    async move {
        use p::client_action::Variant;
        let handle = ctx.handle(None)?;
        let info = ctx.activity.info();
        let workflow_id = info.workflow_id.clone().unwrap();
        let queue = info.task_queue.clone();
        let start_input = || wire::encode(&p::WorkflowInput::default());
        match action
            .variant
            .ok_or_else(|| wire::invalid("unrecognized client action"))?
        {
            Variant::NestedActions(set) => return client_action_set(ctx, set).await,
            Variant::DoDescribe(_) => {
                handle
                    .describe(Default::default())
                    .await
                    .map_err(activity_rpc_error)?;
            }
            Variant::DoSignal(signal) => {
                let (name, args) = match signal.variant {
                    Some(p::do_signal::Variant::DoSignalActions(actions)) => {
                        ("do_actions_signal".to_owned(), wire::encode(&actions))
                    }
                    Some(p::do_signal::Variant::Custom(custom)) => {
                        (custom.name, RawValue::new(custom.args))
                    }
                    None => return Err(wire::invalid("do_signal has no variant").into()),
                };
                let definition = UntypedSignal::<UntypedWorkflow>::new(name);
                if signal.with_start {
                    ctx.activity
                        .client()
                        .signal_with_start_workflow(
                            UntypedWorkflow::new("kitchenSink"),
                            start_input(),
                            definition,
                            args,
                            WorkflowStartOptions::new(queue, workflow_id)
                                .id_conflict_policy(WorkflowIdConflictPolicy::UseExisting)
                                .build(),
                        )
                        .await
                        .map_err(activity_rpc_error)?;
                } else {
                    handle
                        .signal(definition, args, Default::default())
                        .await
                        .map_err(activity_rpc_error)?;
                }
            }
            Variant::DoQuery(query) => {
                let (name, args) = match query.variant {
                    Some(p::do_query::Variant::ReportState(args)) => {
                        ("report_state".to_owned(), RawValue::new(args.payloads))
                    }
                    Some(p::do_query::Variant::Custom(custom)) => {
                        (custom.name, RawValue::new(custom.args))
                    }
                    None => return Err(wire::invalid("do_query has no variant").into()),
                };
                let result = handle
                    .query(
                        UntypedQuery::<UntypedWorkflow>::new(name),
                        args,
                        Default::default(),
                    )
                    .await;
                check_expected(
                    result.map(|_| ()).map_err(activity_rpc_error),
                    query.failure_expected,
                )?;
            }
            Variant::DoUpdate(update) => {
                let (name, args) = match update.variant {
                    Some(p::do_update::Variant::DoActions(actions)) => {
                        ("do_actions_update".to_owned(), wire::encode(&actions))
                    }
                    Some(p::do_update::Variant::Custom(custom)) => {
                        (custom.name, RawValue::new(custom.args))
                    }
                    None => return Err(wire::invalid("do_update has no variant").into()),
                };
                let definition = UntypedUpdate::<UntypedWorkflow>::new(name);
                let result = if update.with_start {
                    ctx.activity
                        .client()
                        .execute_update_with_start_workflow(
                            UntypedWorkflow::new("kitchenSink"),
                            start_input(),
                            definition,
                            args,
                            WorkflowUpdateWithStartOptions::new(
                                queue,
                                workflow_id,
                                WorkflowIdConflictPolicy::UseExisting,
                            )
                            .maybe_update_id(
                                (!update.update_id.is_empty()).then_some(update.update_id),
                            )
                            .build(),
                        )
                        .await
                        .map(|_| ())
                        .map_err(activity_rpc_error)
                } else {
                    handle
                        .execute_update(
                            definition,
                            args,
                            temporalio_client::WorkflowExecuteUpdateOptions::builder()
                                .maybe_update_id(
                                    (!update.update_id.is_empty()).then_some(update.update_id),
                                )
                                .build(),
                        )
                        .await
                        .map(|_| ())
                        .map_err(activity_rpc_error)
                };
                check_expected(result, update.failure_expected)?;
            }
            Variant::DoStandaloneNexusOperation(operation) => {
                crate::standalone_nexus::execute(ctx.activity.clone(), operation).await?;
            }
            Variant::DoStandaloneActivity(action) => {
                let activity = action
                    .activity
                    .ok_or_else(|| wire::invalid("DoStandaloneActivity.activity is required"))?;
                // Cancellation stops the caller's wait, not the independent activity.
                let client = ctx.activity.client();
                let run = async {
                    crate::standalone_activity::start(
                        &client,
                        &activity,
                        crate::standalone_activity::fresh_id(&workflow_id),
                    )
                    .await?
                    .result()
                    .await
                    .map_err(crate::standalone_activity::result_error)
                };
                tokio::select! {
                    result = run => { result?; }
                    _ = ctx.activity.cancelled() => return Err(ActivityError::cancelled()),
                }
            }
            Variant::DoStandaloneActivityOperatorCommands(action) => {
                crate::standalone_activity_operator::execute(ctx.activity.clone(), action).await?;
            }
        }
        Ok(())
    }
    .boxed()
}

fn activity_rpc_error(error: impl std::fmt::Display) -> ActivityError {
    ApplicationFailure::new(error.to_string()).into()
}

fn check_expected(
    result: Result<(), ActivityError>,
    failure_expected: bool,
) -> Result<(), ActivityError> {
    if failure_expected { Ok(()) } else { result }
}
