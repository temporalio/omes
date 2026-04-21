namespace Temporalio.Omes;

public static class App
{
    public static Task<int> Main(string[] args) => RunAsync(args);

    public static Task<int> RunAsync(string[] args) =>
        Temporalio.Omes.Projects.Harness.App.RunAsync(KitchenSinkApp.Create(), args);
}
