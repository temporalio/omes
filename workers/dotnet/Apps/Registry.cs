using Temporalio.Omes.Apps.HelloWorld;
using Temporalio.Omes.Apps.Worker;
using HarnessApp = Temporalio.Omes.Projects.Harness.App;

namespace Temporalio.Omes.Apps;

public static class Registry
{
    private const string DefaultAppName = "worker";

    private static readonly IReadOnlyDictionary<string, HarnessApp> Apps = new Dictionary<string, HarnessApp>
    {
        ["helloworld"] = HelloWorldApp.Create(),
        ["worker"] = WorkerApp.Create(),
    };

    public static Task<int> RunAsync(string[] args)
    {
        var appName = DefaultAppName;
        var harnessArgs = args;
        if (args.Length >= 2 && args[0] == "--app")
        {
            appName = args[1];
            harnessArgs = args[2..];
        }

        if (appName.Length == 0 || !Apps.TryGetValue(appName, out var app))
        {
            Console.Error.WriteLine($"unknown .NET worker app {appName}");
            return Task.FromResult(1);
        }

        return HarnessApp.RunAsync(app, harnessArgs);
    }
}
