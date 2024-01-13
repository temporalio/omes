using Temporal.Omes.KitchenSink;
using Temporalio.Activities;
using Temporalio.Api.Common.V1;
using Temporalio.Exceptions;
using Temporalio.Workflows;

namespace Temporalio.Omes;

[Workflow("kitchenSink")]
public class KitchenSinkWorkflow
{
    [WorkflowRun]
    public async Task RunAsync(WorkflowInput? workflowInput)
    {
        // Run all initial input actions
        if (workflowInput?.InitialActions is { } actions)
        {
            foreach (var actionSet in actions)
            {
                var returnMe = await HandleActionSet(actionSet);
                if (returnMe != null)
                {
                    return;
                }
            }
        }
    }

    private async Task<Payload?> HandleActionSet(ActionSet actionSet)
    {
        Payload? returnMe = null;
        // If actions are non-concurrent, just execute and return if requested
        if (!actionSet.Concurrent)
        {
            foreach (var action in actionSet.Actions)
            {
                returnMe = await HandleAction(action);
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
            var task = new Task(async () => { returnMe = await HandleAction(action); });
            tasks.Add(task);
        }

        var waitReturnSet = Workflow.WaitConditionAsync(() => returnMe != null);
        var allTasksDone = Task.WhenAll(tasks);
        await Workflow.WhenAnyAsync(waitReturnSet, allTasksDone);


        return returnMe;
    }

    private async Task<Payload?> HandleAction(Temporal.Omes.KitchenSink.Action action)
    {
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
            throw Workflow.CreateContinueAsNewException("kitchenSink", can.Arguments);
        }
        else if (action.Timer is { } timer)
        {
            await Workflow.DelayAsync((int)timer.Milliseconds);
        }
        else if (action.ExecActivity is { } execActivity)
        {
            // TODO: Cancel handler
            await LaunchActivity(execActivity);
        }

        return null;
    }

    private Task LaunchActivity(ExecuteActivityAction eaa)
    {
        var actType = "noop";
        var args = new List<object>();
        if (eaa.Delay is { } delay)
        {
            actType = "delay";
            args.Add(delay);
        }

        if (eaa.IsLocal != null)
        {
            LocalActivityOptions opts = new()
            {
                ScheduleToCloseTimeout = eaa.ScheduleToCloseTimeout?.ToTimeSpan(),
                ScheduleToStartTimeout = eaa.ScheduleToStartTimeout?.ToTimeSpan(),
                StartToCloseTimeout = eaa.StartToCloseTimeout?.ToTimeSpan(),
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
                : proto.NonRetryableErrorTypes,
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
}