using Google.Protobuf.WellKnownTypes;
using Temporal.Omes.KitchenSink;

namespace Temporalio.Omes;

// Maps an ExecuteActivityAction to its registered activity name and args.
// Shared by the workflow-scheduled path and the standalone-activity path.
public static class ActivityDispatch
{
    public static (string, IReadOnlyCollection<object>) NameAndArgs(ExecuteActivityAction act)
    {
        if (act.Delay is { } delay)
        {
            return ("delay", new object[] { delay });
        }
        if (act.Payload is { } payload)
        {
            var inputData = new byte[payload.BytesToReceive];
            for (int i = 0; i < inputData.Length; i++)
            {
                inputData[i] = (byte)(i % 256);
            }
            return ("payload", new object[] { inputData, payload.BytesToReturn });
        }
        if (act.Client is { } client)
        {
            return ("client", new object[] { client });
        }
        if (act.RetryableError is { } retryableError)
        {
            return ("retryable_error", new object[] { retryableError });
        }
        if (act.Timeout is { } timeout)
        {
            return ("timeout", new object[] { timeout });
        }
        if (act.Heartbeat is { } heartbeat)
        {
            return ("heartbeat", new object[] { heartbeat });
        }
        return ("noop", Array.Empty<object>());
    }

    public static Temporalio.Common.RetryPolicy RetryPolicyFromProto(
        Temporalio.Api.Common.V1.RetryPolicy proto)
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
}
