using System.CommandLine;
using Temporalio.Client;
using Temporalio.Omes.Projects.Harness;
using Xunit;

namespace Temporalio.Omes.Projects.Tests.HarnessTests;

public class WorkerTests
{
    [Fact]
    public async Task RunPassesSharedClientAndContextToEachWorkerFactory()
    {
        var sharedClient = HarnessTestSupport.CreateStrictTemporalClientProbe();
        ClientConfig? capturedConfig = null;
        var createdWorkers = new List<string>();
        IReadOnlyList<string>? runWorkersInput = null;
        var seenClients = new List<ITemporalClient>();
        var parseResult = WorkerHarness.CreateWorkerCommand().Parse(
            [
                "--task-queue",
                "omes",
                "--task-queue-suffix-index-start",
                "1",
                "--task-queue-suffix-index-end",
                "2",
                "--log-level",
                "panic",
            ]);
        var options = WorkerHarness.ParseWorkerOptions(parseResult);

        await WorkerHarness.RunCoreAsync(
            workerFactory: (client, context) =>
            {
                seenClients.Add(client);
                createdWorkers.Add(context.TaskQueue);
                return context.TaskQueue;
            },
            clientFactory: config =>
            {
                capturedConfig = config;
                return Task.FromResult(sharedClient);
            },
            options: options,
            runWorkersAsync: workers =>
            {
                runWorkersInput = workers;
                return Task.CompletedTask;
            });

        Assert.All(seenClients, client => Assert.Same(sharedClient, client));
        Assert.Equal(["omes-1", "omes-2"], createdWorkers);
        Assert.Equal(createdWorkers, runWorkersInput);
        Assert.NotNull(capturedConfig);
        Assert.Equal("localhost:7233", capturedConfig!.ServerAddress);
        Assert.Equal("default", capturedConfig.Namespace);
        Assert.Null(capturedConfig.ApiKey);
        Assert.Null(capturedConfig.Tls);
    }

    [Fact]
    public async Task WorkerModeCancelsRemainingWorkersWhenOneFails()
    {
        var waitingWorkerSawCancellation = false;
        var failingWorker = new FakeHarnessWorker(
            _ => Task.FromException(new InvalidOperationException("boom")));
        var waitingWorker = new FakeHarnessWorker(
            async cancellationToken =>
            {
                try
                {
                    await Task.Delay(Timeout.InfiniteTimeSpan, cancellationToken);
                }
                catch (OperationCanceledException)
                {
                    waitingWorkerSawCancellation = true;
                }
            });

        var error = await Assert.ThrowsAsync<InvalidOperationException>(
            () =>
                WorkerHarness.RunWorkersAsync(
                    [failingWorker, waitingWorker],
                    static (worker, cancellationToken) => worker.RunAsync(cancellationToken),
                    static worker => worker.Dispose()));

        Assert.Equal("boom", error.Message);
        Assert.Equal(1, failingWorker.RunCalls);
        Assert.Equal(1, waitingWorker.RunCalls);
        Assert.True(waitingWorkerSawCancellation);
        Assert.Equal(1, failingWorker.DisposeCalls);
        Assert.Equal(1, waitingWorker.DisposeCalls);
    }

    [Fact]
    public async Task WorkerModeWaitsForAllWorkersToCompleteWhenNoneFail()
    {
        var releaseWorker = new TaskCompletionSource(TaskCreationOptions.RunContinuationsAsynchronously);
        var blockingWorkerStarted = new TaskCompletionSource(TaskCreationOptions.RunContinuationsAsynchronously);
        var quickWorker = new FakeHarnessWorker(_ => Task.CompletedTask);
        var blockingWorkerSawCancellation = false;
        var blockingWorker = new FakeHarnessWorker(
            async cancellationToken =>
            {
                blockingWorkerStarted.SetResult();
                try
                {
                    await releaseWorker.Task.WaitAsync(cancellationToken);
                }
                catch (OperationCanceledException)
                {
                    blockingWorkerSawCancellation = true;
                    throw;
                }
            });

        var runTask = WorkerHarness.RunWorkersAsync(
            [quickWorker, blockingWorker],
            static (worker, cancellationToken) => worker.RunAsync(cancellationToken),
            static worker => worker.Dispose());

        await blockingWorkerStarted.Task;
        var completedTask = await Task.WhenAny(runTask, Task.Delay(TimeSpan.FromMilliseconds(200)));

        Assert.NotSame(runTask, completedTask);
        Assert.False(blockingWorkerSawCancellation);

        releaseWorker.SetResult();
        await runTask;

        Assert.Equal(1, quickWorker.RunCalls);
        Assert.Equal(1, blockingWorker.RunCalls);
        Assert.Equal(1, quickWorker.DisposeCalls);
        Assert.Equal(1, blockingWorker.DisposeCalls);
    }
}

file sealed class FakeHarnessWorker(Func<CancellationToken, Task> runAsync)
{
    public int RunCalls { get; private set; }

    public int DisposeCalls { get; private set; }

    public async Task RunAsync(CancellationToken cancellationToken)
    {
        RunCalls++;
        await runAsync(cancellationToken);
    }

    public void Dispose() => DisposeCalls++;
}
