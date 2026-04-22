using System.Reflection;
using System.Runtime.CompilerServices;
using Temporalio.Client;
using Xunit;

internal static class HarnessTestSupport
{
    public static ITemporalClient CreateStrictTemporalClientProbe() =>
        DispatchProxy.Create<ITemporalClient, StrictTemporalClientDispatchProxy>();
}

file class StrictTemporalClientDispatchProxy : DispatchProxy
{
    protected override object? Invoke(MethodInfo? targetMethod, object?[]? args)
    {
        if (targetMethod is null)
        {
            return null;
        }

        if (targetMethod.DeclaringType == typeof(object))
        {
            return targetMethod.Name switch
            {
                nameof(ToString) => "StrictTemporalClientProbe",
                nameof(GetHashCode) => RuntimeHelpers.GetHashCode(this),
                nameof(Equals) => ReferenceEquals(this, args?[0]),
                _ => throw new InvalidOperationException($"Unsupported object method {targetMethod.Name} on temporal client probe"),
            };
        }

        throw new InvalidOperationException($"Unexpected temporal client call in test probe: {targetMethod.Name}");
    }
}
