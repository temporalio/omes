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
        if (workflowInput?.InitialActions is { } actions)
        {
            foreach (var actionSet in actions)
            {
                var returnMe = await HandleActionSetAsync(actionSet);
                if (returnMe != null)
                {
                    return null;
                }
            }
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
        foreach (var action in actionSet.Actions)
        {
            async void Action()
            {
                returnMe = await HandleActionAsync(action);
            }

            var task = new Task(Action);
            task.Start();
            tasks.Add(task);
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
                Workflow.DelayAsync((int)timer.Milliseconds, tokenSrc.Token)
                    .ContinueWith(_ => true),
                tokenSrc,
                timer.AwaitableChoice);
        }
        else if (action.ExecActivity is { } execActivity)
        {
            await HandleAwaitableChoiceAsync(
                LaunchActivity(execActivity, tokenSrc).ContinueWith(_ => true),
                tokenSrc,
                execActivity.AwaitableChoice);
        }
        else if (action.ExecChildWorkflow is { } execChild)
        {
            var childType = execChild.WorkflowType ?? "kitchenSink";
            var args = execChild.Input.Select(p => new RawValue(p)).ToArray();
            var options = new ChildWorkflowOptions
            {
                CancellationToken = tokenSrc.Token,
                Id = execChild.WorkflowId == "" ? null : execChild.WorkflowId,
                TaskQueue = execChild.TaskQueue,
                ExecutionTimeout = execChild.WorkflowExecutionTimeout?.ToTimeSpan(),
                TaskTimeout = execChild.WorkflowTaskTimeout?.ToTimeSpan(),
                RunTimeout = execChild.WorkflowRunTimeout?.ToTimeSpan()
            };
            var childTask = Workflow.StartChildWorkflowAsync(childType, args, options);
            await HandleAwaitableChoiceAsync(childTask, tokenSrc, execChild.AwaitableChoice,
                afterStartedFn: t => t, afterCompletedFn: async t =>
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
                await awaitableTask;
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

    [Activity("client")]
    public static async Task Client(ExecuteActivityAction.Types.ClientActivity clientActivity)
    {
        // For now, skip the client activity functionality that requires application context
        await Task.CompletedTask;
        throw new NotImplementedException("Client activity not implemented - application context not available in this version");
    }

    private class ClientActionsExecutor
    {
        private readonly ITemporalClient _client;
        private string? _workflowId;
        private string? _runId;
        private string _workflowType = "kitchenSink";
        private object? _workflowInput;

        public ClientActionsExecutor(ITemporalClient client)
        {
            _client = client;
            _workflowInput = null; // Explicitly assign to avoid warning
        }

        public async Task ExecuteClientSequenceAsync(ClientSequence clientSeq)
        {
            foreach (var actionSet in clientSeq.ActionSets)
            {
                await ExecuteClientActionSetAsync(actionSet);
            }
        }

        private async Task ExecuteClientActionSetAsync(ClientActionSet actionSet)
        {
            if (actionSet.Concurrent)
            {
                throw new ArgumentException("Concurrent client actions are not supported in .NET worker");
            }

            foreach (var action in actionSet.Actions)
            {
                await ExecuteClientActionAsync(action);
            }

            if (actionSet.WaitForCurrentRunToFinishAtEnd)
            {
                if (!string.IsNullOrEmpty(_workflowId) && !string.IsNullOrEmpty(_runId))
                {
                    var handle = _client.GetWorkflowHandle(_workflowId, _runId);
                    try
                    {
                        await handle.GetResultAsync();
                    }
                    catch (Exception)
                    {
                    }
                }
            }
        }

        private async Task ExecuteClientActionAsync(ClientAction action)
        {
            if (action.DoSignal != null)
            {
                await ExecuteSignalActionAsync(action.DoSignal);
            }
            else if (action.DoUpdate != null)
            {
                await ExecuteUpdateActionAsync(action.DoUpdate);
            }
            else if (action.DoQuery != null)
            {
                await ExecuteQueryActionAsync(action.DoQuery);
            }
            else if (action.NestedActions != null)
            {
                await ExecuteClientActionSetAsync(action.NestedActions);
            }
            else
            {
                throw new ArgumentException("Client action must have a recognized variant");
            }
        }

        private async Task ExecuteSignalActionAsync(DoSignal signal)
        {
            string signalName;
            object? signalArgs;

            if (signal.DoSignalActions != null)
            {
                signalName = "do_actions_signal";
                signalArgs = signal.DoSignalActions;
            }
            else if (signal.Custom != null)
            {
                signalName = signal.Custom.Name;
                signalArgs = signal.Custom.Args.Count > 0 ? signal.Custom.Args : null;
            }
            else
            {
                throw new ArgumentException("DoSignal must have a recognizable variant");
            }

            if (signal.WithStart)
            {
                var workflowId = _workflowId ?? Guid.NewGuid().ToString();
                var options = new WorkflowOptions
                {
                    Id = workflowId
                };
                var handle = await _client.StartWorkflowAsync(_workflowType, new[] { _workflowInput }, options);
                await handle.SignalAsync(signalName, new[] { signalArgs });
                _workflowId = handle.Id;
                _runId = handle.ResultRunId;
            }
            else
            {
                var handle = _client.GetWorkflowHandle(_workflowId!);
                await handle.SignalAsync(signalName, new[] { signalArgs });
            }
        }

        private async Task ExecuteUpdateActionAsync(DoUpdate update)
        {
            string updateName;
            object? updateArgs;

            if (update.DoActions != null)
            {
                updateName = "do_actions_update";
                updateArgs = update.DoActions;
            }
            else if (update.Custom != null)
            {
                updateName = update.Custom.Name;
                updateArgs = update.Custom.Args.Count > 0 ? update.Custom.Args : null;
            }
            else
            {
                throw new ArgumentException("DoUpdate must have a recognizable variant");
            }

            try
            {
                if (update.WithStart)
                {
                    var workflowId = _workflowId ?? Guid.NewGuid().ToString();
                    var options = new WorkflowOptions
                    {
                        Id = workflowId
                    };
                    var handle = await _client.StartWorkflowAsync(_workflowType, new[] { _workflowInput }, options);
                    await handle.ExecuteUpdateAsync(updateName, new[] { updateArgs });
                    _workflowId = handle.Id;
                    _runId = handle.ResultRunId;
                }
                else
                {
                    var handle = _client.GetWorkflowHandle(_workflowId!);
                    await handle.ExecuteUpdateAsync(updateName, new[] { updateArgs });
                }
            }
            catch (Exception)
            {
                if (!update.FailureExpected)
                {
                    throw;
                }
            }
        }

        private async Task ExecuteQueryActionAsync(DoQuery query)
        {
            try
            {
                if (query.ReportState != null)
                {
                    var handle = _client.GetWorkflowHandle(_workflowId!);
                    await handle.QueryAsync<object>("report_state", Array.Empty<object>());
                }
                else if (query.Custom != null)
                {
                    var handle = _client.GetWorkflowHandle(_workflowId!);
                    var queryArgs = query.Custom.Args.Count > 0 ? query.Custom.Args.ToArray() : null;
                    await handle.QueryAsync<object>(query.Custom.Name, queryArgs ?? Array.Empty<object>());
                }
                else
                {
                    throw new ArgumentException("DoQuery must have a recognizable variant");
                }
            }
            catch (Exception)
            {
                if (!query.FailureExpected)
                {
                    throw;
                }
            }
        }
    }
}