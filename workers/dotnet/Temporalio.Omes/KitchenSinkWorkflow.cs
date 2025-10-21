using Temporal.Omes.KitchenSink;
using Temporalio.Activities;
using Temporalio.Api.Common.V1;
using Temporalio.Client;
using Temporalio.Common;
using Temporalio.Converters;
using Temporalio.Exceptions;
using Temporalio.Workflows;
using Priority = Temporalio.Api.Common.V1.Priority;
using RetryPolicy = Temporalio.Api.Common.V1.RetryPolicy;

namespace Temporalio.Omes;

[Workflow("kitchenSink")]
public class KitchenSinkWorkflow
{
    private readonly Queue<ActionSet> actionSetQueue = new();

    [WorkflowSignal("do_actions_signal")]
    public async Task DoActionsSignalAsync(DoSignal.Types.DoSignalActions doSignals)
    {
        if (doSignals.DoActionsInMain is { } inMain)
        {
            actionSetQueue.Enqueue(inMain);
        }
        else
        {
            await HandleActionSetAsync(doSignals.DoActions);
        }
    }

    [WorkflowUpdate("do_actions_update")]
    public async Task<object> DoActionsUpdateAsync(DoActionsUpdate actionsUpdate)
    {
        var retval = await HandleActionSetAsync(actionsUpdate.DoActions);
        if (retval != null)
        {
            return retval;
        }

        return CurrentWorkflowState;
    }

    [WorkflowUpdateValidator(nameof(DoActionsUpdateAsync))]
    public void DoActionsUpdateValidator(DoActionsUpdate actionsUpdate)
    {
        if (actionsUpdate.RejectMe != null)
        {
            throw new ApplicationFailureException("Rejected");
        }
    }

    [WorkflowQuery("report_state")]
    public WorkflowState CurrentWorkflowState { get; private set; } = new();

    [WorkflowRun]
    public async Task<Payload?> RunAsync(WorkflowInput? workflowInput)
    {
        // Run all initial input actions
        Payload? initialReturnValue = null;
        if (workflowInput?.InitialActions is { } actions)
        {
            foreach (var actionSet in actions)
            {
                var returnMe = await HandleActionSetAsync(actionSet);
                if (returnMe != null)
                {
                    // Store return value but continue to check signal deduplication
                    initialReturnValue = returnMe;
                    break;
                }
            }
        }

        // Check signal deduplication after initial actions
        // (if initial actions errored, we never reach here)
        if (workflowInput?.ExpectedSignalCount > 0)
        {
            throw new ApplicationFailureException("signal deduplication not implemented");
        }

        // If initial actions returned a value, return it now
        if (initialReturnValue != null)
        {
            return initialReturnValue;
        }

        // Run all actions from signals
        while (true)
        {
            await Workflow.WaitConditionAsync(() => actionSetQueue.Count > 0);
            var actionSet = actionSetQueue.Dequeue();
            var returnMe = await HandleActionSetAsync(actionSet);
            if (returnMe != null)
            {
                return returnMe;
            }
        }
    }

    private async Task<Payload?> HandleActionSetAsync(ActionSet actionSet)
    {
        Payload? returnMe = null;
        // If actions are non-concurrent, just execute and return if requested
        if (!actionSet.Concurrent)
        {
            foreach (var action in actionSet.Actions)
            {
                returnMe = await HandleActionAsync(action);
                if (returnMe != null)
                {
                    return returnMe;
                }
            }

            return returnMe;
        }

        // If actions are concurrent, run them all, returning early if any action wishes to return
        var tasks = new List<Task>();

        // Local async function to process each action
        async Task ProcessActionAsync(Temporal.Omes.KitchenSink.Action action)
        {
            var result = await HandleActionAsync(action);
            if (result != null)
            {
                returnMe = result;
            }
        }

        foreach (var action in actionSet.Actions)
        {
            tasks.Add(ProcessActionAsync(action));
        }

        var waitReturnSet = Workflow.WaitConditionAsync(() => returnMe != null);
        var allTasksDone = Task.WhenAll(tasks);
        await Workflow.WhenAnyAsync(waitReturnSet, allTasksDone);

        return returnMe;
    }

    private async Task<Payload?> HandleActionAsync(Temporal.Omes.KitchenSink.Action action)
    {
        var tokenSrc = CancellationTokenSource.CreateLinkedTokenSource(Workflow.CancellationToken);
        if (action.ReturnResult is { } rr)
        {
            return rr.ReturnThis;
        }
        else if (action.ReturnError is { } re)
        {
            throw new ApplicationFailureException(re.Failure.Message);
        }
        else if (action.ContinueAsNew is { } can)
        {
            var args = can.Arguments.Select(p => new RawValue(p)).ToArray();
            throw Workflow.CreateContinueAsNewException("kitchenSink", args);
        }
        else if (action.Timer is { } timer)
        {
            await HandleAwaitableChoiceAsync(
                ToBool(Workflow.DelayAsync((int)timer.Milliseconds, tokenSrc.Token)),
                tokenSrc,
                timer.AwaitableChoice);
        }
        else if (action.ExecActivity is { } execActivity)
        {
            await HandleAwaitableChoiceAsync(
                ToBool(LaunchActivity(execActivity, tokenSrc)),
                tokenSrc,
                execActivity.AwaitableChoice
            );
        }
        else if (action.ExecChildWorkflow is { } execChild)
        {
            var childType = string.IsNullOrEmpty(execChild.WorkflowType) ? "kitchenSink" : execChild.WorkflowType;
            var childId = string.IsNullOrEmpty(execChild.WorkflowId) ? null : execChild.WorkflowId;
            var tq = string.IsNullOrEmpty(execChild.TaskQueue) ? null : execChild.TaskQueue;
            var args = execChild.Input.Select(p => new RawValue(p)).ToArray();
            var options = new ChildWorkflowOptions
            {
                CancellationToken = tokenSrc.Token,
                Id = childId,
                ExecutionTimeout = execChild.WorkflowExecutionTimeout?.ToTimeSpan(),
                TaskTimeout = execChild.WorkflowTaskTimeout?.ToTimeSpan(),
                RunTimeout = execChild.WorkflowRunTimeout?.ToTimeSpan(),
                TypedSearchAttributes = execChild.SearchAttributes.Count == 0
                    ? SearchAttributeCollection.Empty
                    : SearchAttributeCollection.FromProto(new SearchAttributes { IndexedFields = { execChild.SearchAttributes } }),
            };
            var childTask = Workflow.StartChildWorkflowAsync(childType, args, options);
            await HandleAwaitableChoiceAsync(childTask, tokenSrc, execChild.AwaitableChoice,
                afterStartedFn: async t => await t,
                afterCompletedFn: async t =>
                {
                    var childHandle = await t;
                    await childHandle.GetResultAsync();
                });
        }
        else if (action.SetPatchMarker is { } setPatchMarker)
        {
            bool wasPatched;
            if (setPatchMarker.Deprecated)
            {
                Workflow.DeprecatePatch(setPatchMarker.PatchId);
                wasPatched = true;
            }
            else
            {
                wasPatched = Workflow.Patched(setPatchMarker.PatchId);
            }

            if (wasPatched)
            {
                return await HandleActionAsync(setPatchMarker.InnerAction);
            }
        }
        else if (action.SetWorkflowState is { } setWorkflowState)
        {
            CurrentWorkflowState = setWorkflowState;
        }
        else if (action.AwaitWorkflowState is { } awaitWorkflowState)
        {
            await Workflow.WaitConditionAsync(() =>
            {
                if (!CurrentWorkflowState.Kvs.TryGetValue(awaitWorkflowState.Key, out string value))
                {
                    return false;
                }

                return value == awaitWorkflowState.Value;
            });
        }
        else if (action.UpsertMemo is { } upsertMemo)
        {
            var memoUpdates = new List<MemoUpdate>();
            foreach (var keyval in upsertMemo.UpsertedMemo.Fields)
            {
                memoUpdates.Add(MemoUpdate.ValueSet(keyval.Key, keyval.Value));
            }

            Workflow.UpsertMemo(memoUpdates.ToArray());
        }
        else if (action.UpsertSearchAttributes is { } upsertSearchAttributes)
        {
            var saUpdates = new List<SearchAttributeUpdate>();
            foreach (var keyval in upsertSearchAttributes.SearchAttributes)
            {
                if (keyval.Key.Contains("Keyword"))
                {
                    saUpdates.Add(SearchAttributeUpdate.ValueSet(
                        SearchAttributeKey.CreateKeyword(keyval.Key),
                        keyval.Value.Data[0].ToString()));
                }
                else
                {
                    saUpdates.Add(SearchAttributeUpdate.ValueSet(
                        SearchAttributeKey.CreateDouble(keyval.Key), keyval.Value.Data[0]));
                }
            }

            Workflow.UpsertTypedSearchAttributes(saUpdates.ToArray());
        }
        else if (action.NestedActionSet is { } nestedActionSet)
        {
            return await HandleActionSetAsync(nestedActionSet);
        }
        else if (action.NexusOperation is { })
        {
            throw new ApplicationFailureException("ExecuteNexusOperation is not supported");
        }
        else
        {
            throw new ApplicationFailureException("Unrecognized action");
        }

        return null;
    }

    private async Task HandleAwaitableChoiceAsync<T>(
        Task<T> awaitableTask,
        CancellationTokenSource canceller,
        AwaitableChoice? choice,
        Func<Task<T>, Task>? afterStartedFn = null,
        Func<Task<T>, Task>? afterCompletedFn = null
    )
    {
        afterStartedFn ??= _ => Workflow.DelayAsync(1);
        afterCompletedFn ??= t => t;
        choice ??= new AwaitableChoice { WaitFinish = new() };

        var didCancel = false;
        try
        {
            if (choice.Abandon != null)
            {
                // Do nothing
            }
            else if (choice.CancelBeforeStarted != null)
            {
                canceller.Cancel();
                didCancel = true;
                await awaitableTask;
            }
            else if (choice.CancelAfterStarted != null)
            {
                await afterStartedFn(awaitableTask);
                canceller.Cancel();
                didCancel = true;
                await awaitableTask;
            }
            else if (choice.CancelAfterCompleted != null)
            {
                await afterCompletedFn(awaitableTask);
                canceller.Cancel();
                didCancel = true;
                await awaitableTask;
            }
            else
            {
                await afterCompletedFn(awaitableTask);
            }
        }
        catch (Exception e) when (TemporalException.IsCanceledException(e))
        {
            if (didCancel)
            {
                return;
            }
            throw;
        }
    }

    private Task LaunchActivity(ExecuteActivityAction eaa, CancellationTokenSource tokenSrc)
    {
        var actType = "noop";
        var args = new List<object>();
        if (eaa.Delay is { } delay)
        {
            actType = "delay";
            args.Add(delay);
        }
        else if (eaa.Payload is { } payload)
        {
            actType = "payload";
            var inputData = new byte[payload.BytesToReceive];
            for (int i = 0; i < inputData.Length; i++)
            {
                inputData[i] = (byte)(i % 256);
            }
            args.Add(inputData);
            args.Add(payload.BytesToReturn);
        }
        else if (eaa.Client is { } client)
        {
            actType = "client";
            args.Add(client);
        }
        else if (eaa.RetryableError is { } retryableError)
        {
            actType = "retryable_error";
            args.Add(retryableError);
        }
        else if (eaa.Timeout is { } timeout)
        {
            actType = "timeout";
            args.Add(timeout);
        }
        else if (eaa.Heartbeat is { } heartbeat)
        {
            actType = "heartbeat";
            args.Add(heartbeat);
        }

        if (eaa.IsLocal != null)
        {
            LocalActivityOptions opts = new()
            {
                ScheduleToCloseTimeout = eaa.ScheduleToCloseTimeout?.ToTimeSpan(),
                ScheduleToStartTimeout = eaa.ScheduleToStartTimeout?.ToTimeSpan(),
                StartToCloseTimeout = eaa.StartToCloseTimeout?.ToTimeSpan(),
                CancellationToken = tokenSrc.Token,
                RetryPolicy =
                    eaa.RetryPolicy != null ? RetryPolicyFromProto(eaa.RetryPolicy) : null
            };
            return Workflow.ExecuteLocalActivityAsync(actType, args, opts);
        }
        else
        {
            ActivityOptions opts = new()
            {
                TaskQueue = eaa.TaskQueue == "" ? null : eaa.TaskQueue,
                ScheduleToCloseTimeout = eaa.ScheduleToCloseTimeout?.ToTimeSpan(),
                ScheduleToStartTimeout = eaa.ScheduleToStartTimeout?.ToTimeSpan(),
                StartToCloseTimeout = eaa.StartToCloseTimeout?.ToTimeSpan(),
                CancellationToken = tokenSrc.Token,
                Priority =
                    eaa.Priority != null ? PriorityFromProto(eaa) : null,
                RetryPolicy =
                    eaa.RetryPolicy != null ? RetryPolicyFromProto(eaa.RetryPolicy) : null
            };
            return Workflow.ExecuteActivityAsync(actType, args, opts);
        }
    }

    private static async Task<bool> ToBool(Task task)
    {
        await task;
        return true;
    }

    // Duped for now, if exposed by SDK use it from there.
    private static Temporalio.Common.RetryPolicy RetryPolicyFromProto(RetryPolicy proto)
    {
        return new()
        {
            InitialInterval = proto.InitialInterval.ToTimeSpan(),
            BackoffCoefficient = (float)proto.BackoffCoefficient,
            MaximumInterval = proto.MaximumInterval?.ToTimeSpan(),
            MaximumAttempts = proto.MaximumAttempts,
            NonRetryableErrorTypes = proto.NonRetryableErrorTypes.Count == 0
                ? null
                : proto.NonRetryableErrorTypes
        };
    }

    private static Temporalio.Common.Priority PriorityFromProto(ExecuteActivityAction eaa)
    {
        if (eaa.FairnessKey != null)
        {
            throw new ApplicationFailureException("FairnessKey is not supported yet");
        }
        if (eaa.FairnessWeight > 0)
        {
            throw new ApplicationFailureException("FairnessWeight is not supported yet");
        }
        return new()
        {
            PriorityKey = eaa.Priority.PriorityKey
        };
    }

    [Activity("noop")]
    public static void Noop()
    {
    }

    [Activity("delay")]
    public static async Task Delay(Google.Protobuf.WellKnownTypes.Duration delayFor)
    {
        await Task.Delay(delayFor.ToTimeSpan());
    }

    [Activity("payload")]
    public static byte[] Payload(byte[] inputData, int bytesToReturn)
    {
        var output = new byte[bytesToReturn];
        new Random().NextBytes(output);
        return output;
    }

    [Activity("retryable_error")]
    public static void RetryableError(ExecuteActivityAction.Types.RetryableErrorActivity config)
    {
        var info = ActivityExecutionContext.Current.Info;
        if (info.Attempt <= config.FailAttempts)
        {
            throw new ApplicationFailureException("retryable error");
        }
    }

    [Activity("timeout")]
    public static async Task Timeout(ExecuteActivityAction.Types.TimeoutActivity config)
    {
        var info = ActivityExecutionContext.Current.Info;
        var duration = config.SuccessDuration;
        // Failure case: run failure duration (exceeds activity timeout)
        if (info.Attempt <= config.FailAttempts)
        {
            duration = config.FailureDuration;
        }

        // Sleep for failure/success timeout duration.
        // In failure case, this will throw a TaskCancelledException.
        await Task.Delay(duration.ToTimeSpan(), ActivityExecutionContext.Current.CancellationToken);
    }

    [Activity("heartbeat")]
    public static async Task Heartbeat(ExecuteActivityAction.Types.HeartbeatTimeoutActivity config)
    {
        var info = ActivityExecutionContext.Current.Info;
        var shouldSendHeartbeats = info.Attempt > config.FailAttempts;
        var duration = config.SuccessDuration;
        // Failure case: run failure duration (exceeds heartbeat timeout)
        if (!shouldSendHeartbeats)
        {
            duration = config.FailureDuration;
        }
        // Sleep for failure/success timeout duration.
        // In failure case, this will throw a TaskCancelledException.
        await Task.Delay(duration.ToTimeSpan(), ActivityExecutionContext.Current.CancellationToken);
        // If successful, heartbeat
        ActivityExecutionContext.Current.Heartbeat();
    }
}

public class ClientActivitiesImpl
{
    private readonly ITemporalClient _client;
    private readonly bool _errOnUnimplemented;

    public ClientActivitiesImpl(ITemporalClient client, bool errOnUnimplemented = false)
    {
        _client = client;
        _errOnUnimplemented = errOnUnimplemented;
    }

    [Activity("client")]
    public async Task Client(ExecuteActivityAction.Types.ClientActivity clientActivity)
    {
        var activityInfo = ActivityExecutionContext.Current.Info;
        var workflowId = activityInfo.WorkflowId;
        var taskQueue = activityInfo.TaskQueue;

        var executor = new ClientActionsExecutor(_client, workflowId, taskQueue, _errOnUnimplemented);
        await executor.ExecuteClientSequence(clientActivity.ClientSequence);
    }
}
