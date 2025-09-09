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

    public string? WorkflowId { get; set; }

    public ClientActionsExecutor(ITemporalClient client, string workflowId, string taskQueue)
    {
        _client = client;
        WorkflowId = workflowId;
        _taskQueue = taskQueue;
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
            throw new ApplicationFailureException("concurrent client actions are not supported", "UnsupportedOperation", nonRetryable: true);
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
        else if (action.DoSelfDescribe != null)
        {
            await ExecuteSelfDescribeAction(action.DoSelfDescribe);
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

    private async Task ExecuteSelfDescribeAction(DoSelfDescribe selfDescribe)
    {
        try
        {
            // Use current workflow ID if not specified
            var workflowId = string.IsNullOrEmpty(selfDescribe.WorkflowId) ? WorkflowId : selfDescribe.WorkflowId;
            
            // Get the current workflow execution details
            var resp = await _client.WorkflowService.DescribeWorkflowExecutionAsync(
                new Temporalio.Api.WorkflowService.V1.DescribeWorkflowExecutionRequest
                {
                    Namespace = selfDescribe.Namespace,
                    Execution = new Temporalio.Api.Common.V1.WorkflowExecution
                    {
                        WorkflowId = workflowId,
                        RunId = selfDescribe.RunId,
                    }
                });

            // Log the workflow execution details
            Console.WriteLine("Workflow Execution Details:");
            Console.WriteLine($"  Workflow ID: {resp.WorkflowExecutionInfo?.Execution?.WorkflowId}");
            Console.WriteLine($"  Run ID: {resp.WorkflowExecutionInfo?.Execution?.RunId}");
            Console.WriteLine($"  Type: {resp.WorkflowExecutionInfo?.Type?.Name}");
            Console.WriteLine($"  Status: {resp.WorkflowExecutionInfo?.Status}");
            Console.WriteLine($"  Start Time: {resp.WorkflowExecutionInfo?.StartTime}");
            if (resp.WorkflowExecutionInfo?.CloseTime != null)
            {
                Console.WriteLine($"  Close Time: {resp.WorkflowExecutionInfo.CloseTime}");
            }
            Console.WriteLine($"  History Length: {resp.WorkflowExecutionInfo?.HistoryLength}");
            Console.WriteLine($"  Task Queue: {resp.WorkflowExecutionInfo?.TaskQueue}");
        }
        catch (Exception ex)
        {
            throw new ApplicationFailureException($"Failed to describe workflow execution: {ex.Message}", "DescribeWorkflowExecutionFailed", nonRetryable: true);
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