using NexusRpc;
using NexusRpc.Handlers;
using Temporalio.Nexus;

namespace NexusSimpleWorkflowProject;

[NexusService]
public interface IStringService
{
    [NexusOperation]
    string DoSomething(string name);
}

[NexusServiceHandler(typeof(IStringService))]
public class StringServiceHandler
{
    [NexusOperationHandler]
    public IOperationHandler<string, string> DoSomething() =>
        WorkflowRunOperationHandler.FromHandleFactory(
            async (WorkflowRunOperationContext context, string input) =>
            {
                return await context.StartWorkflowAsync(
                    (HandlerWorkflow wf) => wf.RunAsync(input),
                    new() { Id = $"nexus-handler-{Guid.NewGuid()}" });
            });
}
