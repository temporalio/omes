//! Hello-world sample workflow and project callbacks.
use super::{ExecuteContext, InitContext};
use anyhow::{Result, ensure};
use temporalio_client::{Client, WorkflowStartOptions};
use temporalio_macros::{workflow, workflow_methods};
use temporalio_sdk::{WorkflowContext, WorkflowResult};

pub(super) async fn execute(
    client: &Client,
    run: &InitContext,
    context: ExecuteContext,
) -> Result<()> {
    let handle = client
        .start_workflow(
            HelloWorld::run,
            "World".to_owned(),
            WorkflowStartOptions::new(
                context.task_queue,
                format!("{}-{}", run.execution_id, context.iteration),
            )
            .build(),
        )
        .await?;
    let result = handle.get_result(Default::default()).await?;
    ensure!(
        result == "Hello World",
        "unexpected helloworld result: {result:?}"
    );
    println!("{result}");
    Ok(())
}

#[workflow]
#[derive(Default)]
pub(super) struct HelloWorld;

#[workflow_methods]
impl HelloWorld {
    #[run(name = "HelloWorldWorkflow")]
    pub async fn run(_ctx: &mut WorkflowContext<Self>, name: String) -> WorkflowResult<String> {
        Ok(format!("Hello {name}"))
    }
}
