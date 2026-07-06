namespace Temporalio.Omes;

public static class App
{
    public static Task<int> RunAsync(string[] args) => Apps.Registry.RunAsync(args);
}
