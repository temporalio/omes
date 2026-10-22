//! Deterministic Kitchen Sink interpreter: no OS time, RNG or threads in workflows.
use crate::{proto as p, wire};
use futures::{FutureExt, future::LocalBoxFuture};
use std::{
    collections::{BTreeSet, VecDeque},
    task::Poll,
    time::Duration,
};
use temporalio_common::{
    ActivityCloseTimeouts, UntypedActivity, UntypedWorkflow, data_converters::RawValue,
    error::ApplicationFailure,
};
use temporalio_macros::{workflow, workflow_methods};
use temporalio_sdk::{
    ActivityOptions, ChildWorkflowOptions, ContinueAsNewOptions, LocalActivityOptions,
    TimerOptions, WorkflowCancellationToken, WorkflowContext, WorkflowContextView, WorkflowResult,
    WorkflowTermination,
};

type ActionResult = WorkflowResult<Option<RawValue>>;
type Pending = LocalBoxFuture<'static, WorkflowResult<()>>;

#[workflow]
#[derive(Default)]
pub struct KitchenSink {
    state: p::WorkflowState,
    expected: BTreeSet<i32>,
    received: BTreeSet<i32>,
    initialized: bool,
    main_actions: VecDeque<p::ActionSet>,
    // Keep the first ready terminal outcome until run consumes it, including
    // while initial actions are still running. Later signals must not replace it.
    signal_result: Option<WorkflowResult<RawValue>>,
    pending: Vec<Pending>,
}

impl KitchenSink {
    fn record_signal_result(&mut self, result: WorkflowResult<RawValue>) {
        self.signal_result.get_or_insert(result);
    }
}

#[workflow_methods]
impl KitchenSink {
    #[run(name = "kitchenSink")]
    pub async fn run(ctx: &mut WorkflowContext<Self>, input: RawValue) -> WorkflowResult<RawValue> {
        let input: p::WorkflowInput = if input.payloads.is_empty() {
            Default::default()
        } else {
            wire::decode(&input)?
        };
        ctx.state_mut(|s| {
            s.expected = if input.expected_signal_ids.is_empty() {
                (1..=input.expected_signal_count).collect()
            } else {
                input.expected_signal_ids.into_iter().collect()
            };
            s.received = input.received_signal_ids.into_iter().collect();
            s.expected.retain(|id| !s.received.contains(id));
            s.initialized = true;
        });
        let mut result = None;
        for set in input.initial_actions {
            if let Some(value) = action_set(ctx.clone(), set).await? {
                result = Some(value);
            }
        }
        let timeout = ctx
            .info()
            .execution_timeout()
            .and_then(|t| t.checked_sub(Duration::from_secs(10)))
            .filter(|t| !t.is_zero())
            .unwrap_or(Duration::from_secs(60));
        // The missing-signals deadline starts only once there is a result.
        let mut deadline: Option<Pending> = None;
        loop {
            // The first result wins; later signal outcomes are ignored.
            if result.is_none()
                && let Some(signal_result) = ctx.state_mut(|s| s.signal_result.take())
            {
                result = Some(signal_result?);
            }
            if let Some(set) = ctx.state_mut(|s| s.main_actions.pop_front()) {
                if let Some(value) = action_set(ctx.clone(), set).await? {
                    result = Some(value);
                }
                continue;
            }
            let has_result = result.is_some();
            if has_result {
                if ctx.state(|s| s.expected.is_empty()) {
                    break;
                }
                deadline.get_or_insert_with(|| {
                    let ctx = ctx.clone();
                    async move {
                        ctx.timer(timeout).await;
                        Ok(())
                    }
                    .boxed_local()
                });
            }
            let changed = ctx
                .wait_condition(|s| {
                    (!has_result && s.signal_result.is_some())
                        || !s.main_actions.is_empty()
                        || (has_result && s.expected.is_empty())
                })
                .fuse();
            let expired = async {
                match deadline.as_mut() {
                    Some(deadline) => deadline.await,
                    None => futures::future::pending().await,
                }
            }
            .fuse();
            futures::pin_mut!(changed, expired);
            futures::select_biased! {
                changed = changed => changed?,
                _ = expired => return Err(wire::invalid(format!("missing signals: {:?}", ctx.state(|s| s.expected.clone()))).into()),
            }
        }
        Ok(result.expect("completion requires a result"))
    }

    #[signal(name = "do_actions_signal")]
    pub async fn signal(ctx: &mut WorkflowContext<Self>, raw: RawValue) {
        let signal = match wire::decode(&raw) {
            Ok(signal) => signal,
            Err(err) => {
                ctx.state_mut(|s| s.record_signal_result(Err(err.into())));
                return;
            }
        };
        // Signals delivered with the first task wait for run to load the expected
        // signal IDs, then execute concurrently with the initial actions.
        if ctx.wait_condition(|s| s.initialized).await.is_err() {
            return;
        }
        process_signal(ctx.clone(), signal).await;
    }

    #[query(name = "report_state")]
    pub fn report_state(&self, _ctx: &WorkflowContextView, _input: RawValue) -> RawValue {
        wire::encode(&self.state)
    }

    #[update_validator(update)]
    fn validate_update(
        &self,
        _ctx: &WorkflowContextView,
        raw: &RawValue,
    ) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        let update: p::DoActionsUpdate = wire::decode(raw)?;
        if matches!(
            update.variant,
            Some(p::do_actions_update::Variant::RejectMe(_))
        ) {
            return Err("rejected".into());
        }
        Ok(())
    }

    #[update(name = "do_actions_update")]
    pub async fn update(
        ctx: &mut WorkflowContext<Self>,
        raw: RawValue,
    ) -> Result<RawValue, Box<dyn std::error::Error + Send + Sync>> {
        let update: p::DoActionsUpdate = wire::decode(&raw)?;
        let Some(p::do_actions_update::Variant::DoActions(set)) = update.variant else {
            return Err(wire::invalid("do_actions_update has no action set").into());
        };
        let result = action_set(ctx.clone(), set).await.map_err(
            |err| -> Box<dyn std::error::Error + Send + Sync> {
                match err {
                    WorkflowTermination::Failed(err) => Box::new(err),
                    // Continue-as-new from an update fails the update; it
                    // doesn't close the workflow.
                    WorkflowTermination::ContinueAsNew(_) => "continue as new".into(),
                    // Pass cancellation and eviction through unchanged.
                    other => Box::new(other),
                }
            },
        )?;
        Ok(result.unwrap_or_else(|| ctx.state(|s| wire::encode(&s.state))))
    }
}

async fn process_signal(ctx: WorkflowContext<KitchenSink>, signal: p::do_signal::DoSignalActions) {
    let accept = ctx.state_mut(|s| {
        if signal.signal_id == 0 {
            true
        } else if s.expected.remove(&signal.signal_id) {
            s.received.insert(signal.signal_id);
            true
        } else {
            false
        }
    });
    if !accept {
        return;
    }
    match signal.variant {
        Some(p::do_signal::do_signal_actions::Variant::DoActions(set)) => {
            match action_set(ctx.clone(), set).await {
                Ok(Some(result)) => ctx.state_mut(|s| s.record_signal_result(Ok(result))),
                Err(err) => ctx.state_mut(|s| s.record_signal_result(Err(err))),
                Ok(None) => {}
            }
        }
        Some(p::do_signal::do_signal_actions::Variant::DoActionsInMain(set)) => {
            ctx.state_mut(|s| s.main_actions.push_back(set));
        }
        None => ctx.state_mut(|s| {
            s.record_signal_result(Err(wire::invalid("signal has no action set").into()));
        }),
    }
}

fn action_set(
    ctx: WorkflowContext<KitchenSink>,
    set: p::ActionSet,
) -> LocalBoxFuture<'static, ActionResult> {
    async move {
        if set.concurrent {
            // FuturesUnordered introduces its own waker, which the SDK rejects as a
            // non-SDK wake. Poll a stable, declaration-ordered list with the SDK's
            // waker directly. Completion/failure must short-circuit, unlike join_all.
            let mut running: Vec<_> = set
                .actions
                .into_iter()
                .map(|a| Some(action(ctx.clone(), a)))
                .collect();
            return futures::future::poll_fn(move |cx| {
                let mut pending = false;
                for slot in &mut running {
                    if let Some(future) = slot {
                        match future.poll_unpin(cx) {
                            Poll::Ready(Ok(None)) => *slot = None,
                            Poll::Ready(result) => return Poll::Ready(result),
                            Poll::Pending => pending = true,
                        }
                    }
                }
                if pending {
                    Poll::Pending
                } else {
                    Poll::Ready(Ok(None))
                }
            })
            .await;
        } else {
            for a in set.actions {
                if let Some(value) = action(ctx.clone(), a).await? {
                    return Ok(Some(value));
                }
            }
        }
        Ok(None)
    }
    .boxed_local()
}

fn action(
    ctx: WorkflowContext<KitchenSink>,
    a: p::Action,
) -> LocalBoxFuture<'static, ActionResult> {
    async move {
        use p::action::Variant;
        match a
            .variant
            .ok_or_else(|| wire::invalid("unrecognized action"))?
        {
            Variant::ReturnResult(ret) => {
                return Ok(Some(RawValue::new(vec![
                    ret.return_this.unwrap_or_default(),
                ])));
            }
            Variant::ReturnError(ret) => {
                return Err(
                    ApplicationFailure::new(ret.failure.unwrap_or_default().message).into(),
                );
            }
            Variant::NestedActionSet(set) => return action_set(ctx, set).await,
            Variant::SetWorkflowState(state) => ctx.state_mut(|s| s.state = state),
            Variant::AwaitWorkflowState(state) => {
                ctx.wait_condition(|s| s.state.kvs.get(&state.key) == Some(&state.value))
                    .await?
            }
            Variant::Timer(timer) => {
                let token = ctx.cancellation_token().child_token();
                let timer_ctx = ctx.clone();
                let options = TimerOptions::builder(Duration::from_millis(timer.milliseconds))
                    .cancellation_token(token.clone())
                    .build();
                let completion =
                    async move { timer_result(timer_ctx.timer(options).await) }.boxed_local();
                await_choice(ctx, timer.awaitable_choice, token, completion).await?;
            }
            Variant::ExecActivity(activity) => execute_activity(ctx, activity).await?,
            Variant::ExecChildWorkflow(child) => execute_child(ctx, child).await?,
            Variant::ContinueAsNew(can) => {
                let mut options = ContinueAsNewOptions::builder()
                    .maybe_workflow_type(nonempty(can.workflow_type))
                    .maybe_task_queue(nonempty(can.task_queue))
                    .maybe_run_timeout(duration(can.workflow_run_timeout)?)
                    .maybe_task_timeout(duration(can.workflow_task_timeout)?)
                    .maybe_retry_policy(can.retry_policy)
                    .build();
                // Proto maps cannot distinguish empty from unset. Both inherit
                // the current metadata; a nonempty map replaces it in full.
                if !can.memo.is_empty() {
                    let mut memo = temporalio_common::MemoValues::new();
                    for (key, payload) in can.memo {
                        memo.insert(key, RawValue::new(vec![payload]));
                    }
                    options.memo = Some(memo);
                }
                if !can.search_attributes.is_empty() {
                    options.search_attributes = Some(
                        temporalio_common::search_attributes::SearchAttributes::from(
                            temporalio_common::protos::temporal::api::common::v1::SearchAttributes {
                                indexed_fields: can.search_attributes.into_iter().collect(),
                            },
                        ),
                    );
                }
                options.versioning_intent = Some(match can.versioning_intent {
                    0 => temporalio_sdk::VersioningIntent::Unspecified,
                    1 => temporalio_sdk::VersioningIntent::Compatible,
                    2 => temporalio_sdk::VersioningIntent::Default,
                    _ => {
                        return Err(
                            wire::invalid("unknown continue-as-new versioning intent").into()
                        );
                    }
                });
                wire::unsupported(!can.headers.is_empty(), "headers")?;
                return ctx
                    .continue_as_new(RawValue::new(can.arguments), options)
                    .map(|never| match never {});
            }
            Variant::SetPatchMarker(patch) => {
                let enabled = if patch.deprecated {
                    ctx.deprecate_patch(&patch.patch_id)
                } else {
                    ctx.patched(&patch.patch_id)
                };
                if enabled && let Some(inner) = patch.inner_action {
                    return action(ctx, *inner).await;
                }
            }
            Variant::AwaitPendingActions(_) => {
                let pending = ctx.state_mut(|s| std::mem::take(&mut s.pending));
                for future in pending {
                    future.await?;
                }
            }
            Variant::CancelWorkflow(cancel) => {
                ctx.external_workflow(cancel.workflow_id, nonempty(cancel.run_id))
                    .cancel(None)
                    .await?;
            }
            Variant::SendSignal(signal) => {
                let token = ctx.cancellation_token().child_token();
                let target = ctx.external_workflow(signal.workflow_id, nonempty(signal.run_id));
                wire::unsupported(!signal.headers.is_empty(), "headers")?;
                let completion = target.signal(
                    temporalio_client::UntypedSignal::<UntypedWorkflow>::new(signal.signal_name),
                    RawValue::new(signal.args),
                    temporalio_sdk::SignalWorkflowOptions::builder()
                        .cancellation_token(token.clone())
                        .build(),
                );
                await_choice_with_started(
                    ctx,
                    signal.awaitable_choice,
                    token,
                    async move { completion.await.map_err(Into::into) }.boxed_local(),
                    StartedWait::Completion,
                )
                .await?;
            }
            Variant::NexusOperation(operation) => {
                if let Some(completion) =
                    crate::workflow_nexus::execute(ctx.clone(), operation).await?
                {
                    ctx.state_mut(|s| s.pending.push(completion));
                }
            }
            Variant::UpsertSearchAttributes(_) => {
                wire::unsupported(true, "UpsertSearchAttributes")?;
            }
            Variant::UpsertMemo(upsert) => {
                let fields = upsert.upserted_memo.unwrap_or_default().fields;
                // Reject rather than emit an empty modify command.
                if fields.is_empty() {
                    return Err(wire::invalid("memo is empty").into());
                }
                // These are already encoded payloads, including the empty payload
                // deletion sentinel. Do not decode or re-encode their contents.
                ctx.upsert_memo(fields.into_iter().map(|(key, payload)| {
                    (
                        key,
                        Some(temporalio_common::MemoValue::new(RawValue::new(vec![
                            payload,
                        ]))),
                    )
                }))
                .map_err(|err| wire::invalid(format!("invalid memo: {err}")))?;
            }
        }
        Ok(None)
    }
    .boxed_local()
}

pub fn duration(
    value: Option<prost_types::Duration>,
) -> Result<Option<Duration>, ApplicationFailure> {
    value
        .map(|v| Duration::try_from(v).map_err(|e| wire::invalid(format!("invalid duration: {e}"))))
        .transpose()
}

fn nonempty(value: String) -> Option<String> {
    (!value.is_empty()).then_some(value)
}
fn timer_result(result: temporalio_sdk::TimerResult) -> WorkflowResult<()> {
    match result {
        temporalio_sdk::TimerResult::Fired => Ok(()),
        temporalio_sdk::TimerResult::Cancelled => Err(WorkflowTermination::cancelled()),
    }
}

async fn await_choice(
    ctx: WorkflowContext<KitchenSink>,
    choice: Option<p::AwaitableChoice>,
    token: WorkflowCancellationToken,
    completion: Pending,
) -> WorkflowResult<()> {
    await_choice_with_started(ctx, choice, token, completion, StartedWait::Yield).await
}

enum StartedWait {
    Yield,
    Completion,
}

async fn await_choice_with_started(
    ctx: WorkflowContext<KitchenSink>,
    choice: Option<p::AwaitableChoice>,
    token: WorkflowCancellationToken,
    mut completion: Pending,
    started_wait: StartedWait,
) -> WorkflowResult<()> {
    use p::awaitable_choice::Condition;
    // SDK calls are inside owned futures to avoid borrowing the context. Poll once
    // with the SDK's waker to issue the command before abandoning it or cancelling
    // in this workflow task (not FutureExt::now_or_never's synthetic noop waker).
    let immediate = futures::future::poll_fn(|cx| {
        Poll::Ready(match completion.poll_unpin(cx) {
            Poll::Ready(result) => Some(result),
            Poll::Pending => None,
        })
    })
    .await;
    if let Some(result) = immediate {
        return result;
    }
    match choice.and_then(|c| c.condition) {
        Some(Condition::Abandon(_)) => {
            drop(completion);
            Ok(())
        }
        Some(Condition::WaitStarted(_)) => {
            match started_wait {
                StartedWait::Yield => timer_result(ctx.timer(Duration::from_nanos(1)).await)?,
                StartedWait::Completion => {
                    // "Started" for a signal means the server accepted it, not that
                    // the recipient handled it. Keep a ready pending action rather
                    // than polling the SDK future twice.
                    completion.await?;
                    completion = futures::future::ready(Ok(())).boxed_local();
                }
            }
            ctx.state_mut(|s| s.pending.push(completion));
            Ok(())
        }
        Some(Condition::CancelBeforeStarted(_)) => {
            token.cancel();
            swallow_cancel(completion.await)
        }
        Some(Condition::CancelAfterStarted(_)) => {
            match started_wait {
                StartedWait::Yield => timer_result(ctx.timer(Duration::from_nanos(1)).await)?,
                StartedWait::Completion => {
                    completion.await?;
                    token.cancel();
                    return Ok(());
                }
            }
            token.cancel();
            swallow_cancel(completion.await)
        }
        Some(Condition::CancelAfterCompleted(_)) => {
            let result = completion.await;
            token.cancel();
            result
        }
        _ => completion.await,
    }
}

fn swallow_cancel(result: WorkflowResult<()>) -> WorkflowResult<()> {
    match result {
        Err(WorkflowTermination::Cancelled { .. }) => Ok(()),
        Err(WorkflowTermination::Failed(ref e)) if e.as_cancelled().is_some() => Ok(()),
        other => other,
    }
}

/// Shared definition and raw-argument conversion for workflow and standalone activities.
pub(crate) fn activity_input(
    activity: &p::ExecuteActivityAction,
) -> Result<(UntypedActivity, RawValue), ApplicationFailure> {
    use p::execute_activity_action::ActivityType;
    let (name, input): (String, RawValue) = match activity
        .activity_type
        .clone()
        .ok_or_else(|| wire::invalid("activity type is required"))?
    {
        ActivityType::Noop(_) => ("noop".to_owned(), RawValue::new(vec![])),
        ActivityType::Generic(generic) => (generic.r#type, RawValue::new(generic.arguments)),
        ActivityType::Delay(delay) => (
            "delay".into(),
            RawValue::new(vec![wire::json(
                &duration(Some(delay))?.unwrap().as_nanos(),
            )]),
        ),
        ActivityType::Payload(config) => {
            if config.bytes_to_receive < 0 || config.bytes_to_return < 0 {
                return Err(wire::invalid("negative payload size"));
            }
            (
                "payload".into(),
                RawValue::new(vec![
                    crate::activities::bytes_payload(config.bytes_to_receive as usize),
                    wire::json(&config.bytes_to_return),
                ]),
            )
        }
        ActivityType::Client(config) => ("client".into(), wire::encode(&config)),
        ActivityType::RetryableError(config) => ("retryable_error".into(), wire::encode(&config)),
        ActivityType::Timeout(config) => ("timeout".into(), wire::encode(&config)),
        ActivityType::Heartbeat(config) => ("heartbeat".into(), wire::encode(&config)),
        ActivityType::Resources(config) => ("resources".into(), wire::encode(&config)),
    };
    Ok((UntypedActivity::new(name), input))
}

pub(crate) fn activity_priority(
    activity: &p::ExecuteActivityAction,
) -> temporalio_common::Priority {
    let mut priority: temporalio_common::Priority = activity
        .priority
        .clone()
        .map(Into::into)
        .unwrap_or_default();
    if !activity.fairness_key.is_empty() {
        priority.fairness_key = Some(activity.fairness_key.clone());
    }
    if activity.fairness_weight != 0. {
        priority.fairness_weight = Some(activity.fairness_weight);
    }
    priority
}

async fn execute_activity(
    ctx: WorkflowContext<KitchenSink>,
    activity: p::ExecuteActivityAction,
) -> WorkflowResult<()> {
    use p::execute_activity_action::Locality;
    wire::unsupported(!activity.headers.is_empty(), "headers")?;
    let (definition, args) = activity_input(&activity)?;
    let token = ctx.cancellation_token().child_token();
    let schedule = duration(activity.schedule_to_close_timeout)?;
    let start = duration(activity.start_to_close_timeout)?;
    let completion: Pending = if matches!(activity.locality, Some(Locality::IsLocal(_))) {
        wire::unsupported(!activity.task_queue.is_empty(), "local activity task queue")?;
        wire::unsupported(
            activity.heartbeat_timeout.is_some(),
            "local activity heartbeat timeout",
        )?;
        wire::unsupported(
            activity.priority.is_some()
                || !activity.fairness_key.is_empty()
                || activity.fairness_weight != 0.,
            "local activity priority/fairness",
        )?;
        let options = LocalActivityOptions::builder()
            .maybe_schedule_to_close_timeout(schedule)
            .maybe_start_to_close_timeout(start)
            .maybe_schedule_to_start_timeout(duration(activity.schedule_to_start_timeout)?)
            .retry_policy(activity.retry_policy.map(Into::into).unwrap_or_default())
            .cancellation_token(token.clone())
            .build();
        let activity_ctx = ctx.clone();
        async move {
            activity_ctx
                .execute_local_activity(definition, args, options)
                .await
                .map(|_| ())
                .map_err(Into::into)
        }
        .boxed_local()
    } else {
        let close = match (schedule, start) {
            (Some(schedule_to_close), Some(start_to_close)) => {
                ActivityCloseTimeouts::ScheduleAndStartToClose {
                    schedule_to_close,
                    start_to_close,
                }
            }
            (Some(d), None) => ActivityCloseTimeouts::ScheduleToClose(d),
            (None, Some(d)) => ActivityCloseTimeouts::StartToClose(d),
            _ => return Err(wire::invalid("activity requires a close timeout").into()),
        };
        let priority = activity_priority(&activity);
        let mut options = ActivityOptions::with_close_timeouts(close)
            .maybe_task_queue(nonempty(activity.task_queue))
            .maybe_schedule_to_start_timeout(duration(activity.schedule_to_start_timeout)?)
            .maybe_heartbeat_timeout(duration(activity.heartbeat_timeout)?)
            .maybe_retry_policy(activity.retry_policy)
            .cancellation_token(token.clone())
            .build();
        if let Some(Locality::Remote(remote)) = activity.locality {
            wire::unsupported(remote.versioning_intent != 0, "activity versioning intent")?;
            options.cancellation_type = match remote.cancellation_type {
                0 => temporalio_sdk::ActivityCancellationType::TryCancel,
                1 => temporalio_sdk::ActivityCancellationType::WaitCancellationCompleted,
                2 => temporalio_sdk::ActivityCancellationType::Abandon,
                _ => return Err(wire::invalid("unknown activity cancellation type").into()),
            };
            options.do_not_eagerly_execute = remote.do_not_eagerly_execute;
        }
        options.priority = Some(priority);
        let activity_ctx = ctx.clone();
        async move {
            activity_ctx
                .execute_activity(definition, args, options)
                .await
                .map(|_| ())
                .map_err(Into::into)
        }
        .boxed_local()
    };
    await_choice(ctx, activity.awaitable_choice, token, completion).await
}

async fn execute_child(
    ctx: WorkflowContext<KitchenSink>,
    child: p::ExecuteChildWorkflowAction,
) -> WorkflowResult<()> {
    wire::unsupported(
        !child.namespace.is_empty() && child.namespace != ctx.namespace(),
        "cross-namespace child",
    )?;
    wire::unsupported(!child.memo.is_empty(), "child memo")?;
    wire::unsupported(child.versioning_intent != 0, "child versioning intent")?;
    wire::unsupported(!child.headers.is_empty(), "headers")?;
    let token = ctx.cancellation_token().child_token();
    let mut options = ChildWorkflowOptions::builder()
        .maybe_workflow_id(nonempty(child.workflow_id))
        .maybe_task_queue(nonempty(child.task_queue))
        .maybe_execution_timeout(duration(child.workflow_execution_timeout)?)
        .maybe_run_timeout(duration(child.workflow_run_timeout)?)
        .maybe_task_timeout(duration(child.workflow_task_timeout)?)
        .maybe_cron_schedule(nonempty(child.cron_schedule))
        .maybe_retry_policy(child.retry_policy.map(Into::into))
        .cancellation_token(token.clone())
        .build();
    if !child.search_attributes.is_empty() {
        options.search_attributes = Some(
            temporalio_common::search_attributes::SearchAttributes::from(
                temporalio_common::protos::temporal::api::common::v1::SearchAttributes {
                    indexed_fields: child.search_attributes.into_iter().collect(),
                },
            ),
        );
    }
    options.cancel_type = match child.cancellation_type {
        0 => temporalio_sdk::ChildWorkflowCancellationType::Abandon,
        1 => temporalio_sdk::ChildWorkflowCancellationType::TryCancel,
        2 => temporalio_sdk::ChildWorkflowCancellationType::WaitCancellationCompleted,
        3 => temporalio_sdk::ChildWorkflowCancellationType::WaitCancellationRequested,
        _ => return Err(wire::invalid("unknown child cancellation type").into()),
    };
    options.parent_close_policy = match child.parent_close_policy {
        0 => temporalio_sdk::ParentClosePolicy::Unspecified,
        1 => temporalio_sdk::ParentClosePolicy::Terminate,
        2 => temporalio_sdk::ParentClosePolicy::Abandon,
        3 => temporalio_sdk::ParentClosePolicy::RequestCancel,
        _ => return Err(wire::invalid("unknown parent close policy").into()),
    };
    options.id_reuse_policy = match child.workflow_id_reuse_policy {
        0 => temporalio_sdk::WorkflowIdReusePolicy::Unspecified,
        1 => temporalio_sdk::WorkflowIdReusePolicy::AllowDuplicate,
        2 => temporalio_sdk::WorkflowIdReusePolicy::AllowDuplicateFailedOnly,
        3 => temporalio_sdk::WorkflowIdReusePolicy::RejectDuplicate,
        4 => temporalio_sdk::WorkflowIdReusePolicy::TerminateIfRunning,
        _ => return Err(wire::invalid("unknown child workflow ID reuse policy").into()),
    };
    let workflow_type = nonempty(child.workflow_type).unwrap_or_else(|| "kitchenSink".into());
    let future = ctx.start_child_workflow(
        UntypedWorkflow::new(workflow_type),
        RawValue::new(child.input),
        options,
    );
    use p::awaitable_choice::Condition;
    let choice = child.awaitable_choice.and_then(|c| c.condition);
    if matches!(choice, Some(Condition::CancelBeforeStarted(_))) {
        token.cancel();
    }
    if matches!(choice, Some(Condition::Abandon(_))) {
        drop(future);
        return Ok(());
    }
    let started = match future.await {
        Ok(started) => started,
        Err(err) => {
            let result = Err(WorkflowTermination::Failed(err.into()));
            return if matches!(choice, Some(Condition::CancelBeforeStarted(_))) {
                swallow_cancel(result)
            } else {
                result
            };
        }
    };
    if matches!(choice, Some(Condition::CancelAfterStarted(_))) {
        token.cancel();
    }
    let completion =
        async move { started.result().await.map(|_| ()).map_err(Into::into) }.boxed_local();
    if matches!(choice, Some(Condition::WaitStarted(_))) {
        ctx.state_mut(|s| s.pending.push(completion));
        return Ok(());
    }
    let result = completion.await;
    if matches!(choice, Some(Condition::CancelAfterCompleted(_))) {
        token.cancel();
    }
    if matches!(
        choice,
        Some(Condition::CancelBeforeStarted(_) | Condition::CancelAfterStarted(_))
    ) {
        swallow_cancel(result)
    } else {
        result
    }
}
