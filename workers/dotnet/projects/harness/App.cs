namespace Temporalio.Omes.Projects.Harness;

public sealed record App(
    WorkerFactory Worker,
    ClientFactory ClientFactory,
    ProjectHandlers? Project = null)
{
    public static Task<int> RunAsync(App app, string[] args)
    {
        if (args.Length == 0 || args[0] == "worker")
        {
            var workerArgs = args.Length > 0 && args[0] == "worker" ? args[1..] : args;
            return WorkerHarness.RunAsync(app, workerArgs);
        }

        if (args[0] == "project-server")
        {
            if (app.Project is null)
            {
                Console.Error.WriteLine("Wanted project-server but no project handlers registered for this app");
                return Task.FromResult(1);
            }

            return ProjectHarness.RunAsync(app, args[1..]);
        }

        Console.Error.WriteLine($"Unknown command: ['{args[0]}']. Expected 'worker' or 'project-server'");
        return Task.FromResult(1);
    }
}
