using Temporalio.Client;
using Temporalio.Exceptions;
using Temporal.Omes.KitchenSink;
using Temporalio.Api.Enums.V1;

namespace Temporalio.Omes;

public class ClientActionsExecutor
{
    private readonly ITemporalClient _client;
    private readonly string _workflowType = "kitchenSink";
    private readonly string _taskQueue;
    private object? _workflowInput = null;
    private string _runId = "";
    private readonly bool _errOnUnimplemented;

    public string? WorkflowId { get; set; }

    public ClientActionsExecutor(ITemporalClient client, string workflowId, string taskQueue, bool errOnUnimplemented = false)
    {
        _client = client;
        WorkflowId = workflowId;
        _taskQueue = taskQueue;
        _errOnUnimplemented = errOnUnimplemented;
    }

    public async Task ExecuteClientSequence(ClientSequence clientSeq)
    {
        foreach (var actionSet in clientSeq.ActionSets)
        {
            await ExecuteClientActionSet(actionSet);
        }
    }

    private async Task ExecuteClientActionSet(ClientActionSet actionSet)
    {
        if (actionSet.Concurrent)
        {
            if (_errOnUnimplemented)
            {
                throw new ApplicationFailureException("concurrent client actions are not supported", "UnsupportedOperation", nonRetryable: true);
            }
            // Skip concurrent actions when not erroring on unimplemented
            Console.WriteLine("Skipping concurrent client actions (not implemented)");
            return;
        }

        foreach (var action in actionSet.Actions)
        {
            await ExecuteClientAction(action);
        }
    }

    private async Task ExecuteClientAction(ClientAction action)
    {
        if (action.DoSignal != null)
        {
            await ExecuteSignalAction(action.DoSignal);
        }
        else if (action.DoUpdate != null)
        {
            await ExecuteUpdateAction(action.DoUpdate);
        }
        else if (action.DoQuery != null)
        {
            await ExecuteQueryAction(action.DoQuery);
        }
        else if (action.NestedActions != null)
        {
            await ExecuteClientActionSet(action.NestedActions);
        }
        else
        {
            throw new ArgumentException("Client action must have a recognized variant");
        }
    }

    private async Task ExecuteSignalAction(DoSignal signal)
    {
        string signalName;
        object? signalArgs = null;

        if (signal.DoSignalActions != null)
        {
            signalName = "do_actions_signal";
            signalArgs = signal.DoSignalActions;
        }
        else if (signal.Custom != null)
        {
            signalName = signal.Custom.Name;
            signalArgs = signal.Custom.Args?.ToArray();
        }
        else
        {
            throw new ArgumentException("DoSignal must have a recognizable variant");
        }

        var args = NormalizeArgsToArray(signalArgs);
        if (signal.WithStart)
        {
            var options = new WorkflowOptions(id: WorkflowId!, taskQueue: _taskQueue);
            options.SignalWithStart(signalName, args);

            var handle = await _client.StartWorkflowAsync(
                _workflowType,
                NormalizeArgsToArray(_workflowInput),
                options);

            WorkflowId = handle.Id;
            _runId = handle.RunId ?? "";
        }
        else
        {
            var handle = _client.GetWorkflowHandle(WorkflowId!);
            await handle.SignalAsync(signalName, args);
        }
    }

    private async Task ExecuteUpdateAction(DoUpdate update)
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
            var args = NormalizeArgsToArray(updateArgs);
            if (update.WithStart)
            {
                var startOperation = WithStartWorkflowOperation.Create(
                    _workflowType,
                    NormalizeArgsToArray(_workflowInput),
                    new(id: WorkflowId!, taskQueue: _taskQueue)
                    {
                        IdConflictPolicy = WorkflowIdConflictPolicy.UseExisting
                    });

                await _client.ExecuteUpdateWithStartWorkflowAsync(
                    updateName,
                    args,
                    new(startOperation));

                var handle = await startOperation.GetHandleAsync();
                WorkflowId = handle.Id;
                _runId = handle.RunId ?? "";
            }
            else
            {
                var handle = _client.GetWorkflowHandle(WorkflowId!);
                await handle.ExecuteUpdateAsync(updateName, args);
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

    private async Task ExecuteQueryAction(DoQuery query)
    {
        try
        {
            if (query.ReportState != null)
            {
                var handle = _client.GetWorkflowHandle(WorkflowId!);
                await handle.QueryAsync<WorkflowState>("report_state", new object[] { query.ReportState });
            }
            else if (query.Custom != null)
            {
                var handle = _client.GetWorkflowHandle(WorkflowId!);
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

    private static object[] NormalizeArgsToArray(object? args)
    {
        return args switch
        {
            null => Array.Empty<object>(),
            object[] array => array,
            _ => new[] { args }
        };
    }
}