using System.CommandLine;
using Temporalio.Client;
using Temporalio.Omes.Projects.Harness;
using Xunit;

namespace Temporalio.Omes.Projects.Tests.HarnessTests;

public class WorkerTests
{
    [Theory]
    [InlineData(null, null, 10000)]
    [InlineData(null, 50000, 50000)]
    [InlineData(null, 0, 0)]
    [InlineData("throughput-stress-baseline-x8", null, 400)]
    [InlineData("throughput-stress-baseline-x8", 50000, 50000)]
    [InlineData("throughput-stress-baseline-x8", 0, 0)]
    public async Task CacheOverridePreservesOtherWorkerSettings(
        string? profile, int? cacheSize, int expectedCacheSize)
    {
        var args = new List<string> { "--max-concurrent-workflow-tasks", "17", "--log-level", "panic" };
        if (cacheSize is { } value)
        {
            args.Add($"--max-cached-workflows={value}");
        }

        var parseResult = WorkerHarness.CreateWorkerCommand().Parse(args.ToArray());
        Assert.Empty(parseResult.Errors);
        var options = WorkerHarness.ParseWorkerOptions(parseResult);
        var sharedClient = HarnessTestSupport.CreateStrictTemporalClientProbe();

        await WorkerHarness.RunCoreAsync(
            workerFactory: (client, context) =>
            {
                Assert.Equal(expectedCacheSize, context.WorkerOptions.MaxCachedWorkflows);
                Assert.Equal(profile is null ? 17 : 64, context.WorkerOptions.MaxConcurrentWorkflowTasks);
                Assert.Equal("omes", context.WorkerOptions.TaskQueue);
                return context.TaskQueue;
            },
            clientFactory: _ => Task.FromResult(sharedClient),
            options: options,
            runWorkersAsync: _ => Task.CompletedTask,
            workerProfile: profile);

        if (profile is not null)
        {
            Assert.Equal(400, WorkerProfiles.LookupWorkerProfile(profile).MaxCachedWorkflows);
        }
    }

    [Theory]
    [InlineData("--max-cached-workflows=-1")]
    [InlineData("--max-cached-workflows=not-a-number")]
    [InlineData("--max-cached-workflows=2147483648")]
    [InlineData("--max-cached-workflows")]
    public void InvalidCacheSizeIsRejected(string argument)
    {
        var parseResult = WorkerHarness.CreateWorkerCommand().Parse([argument]);
        Assert.NotEmpty(parseResult.Errors);
    }

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
            },
            workerProfile: null);

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
