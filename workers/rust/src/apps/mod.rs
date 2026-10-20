//! Explicit application registry. Kitchen Sink remains the default worker.
mod helloworld;

use anyhow::{Result, ensure};
use clap::ValueEnum;
use std::time::Duration;
use temporalio_client::Client;
use temporalio_sdk::WorkerOptions;

#[derive(Clone, Copy, Debug, PartialEq, Eq, ValueEnum)]
pub(crate) enum App {
    Worker,
    Helloworld,
}

// Apps receive the full project context even though helloworld uses little of it.
#[allow(dead_code)]
pub(crate) struct InitContext {
    pub execution_id: String,
    pub run_id: String,
    pub task_queue: String,
    pub config_json: Vec<u8>,
}

#[allow(dead_code)]
pub(crate) struct ExecuteContext {
    pub iteration: i64,
    pub task_queue: String,
    pub payload: Vec<u8>,
}

impl App {
    pub(crate) fn worker_options(self, queue: &str) -> Result<WorkerOptions> {
        let options =
            WorkerOptions::new(queue.to_owned()).graceful_shutdown_period(Duration::from_secs(5));
        Ok(match self {
            Self::Worker => options
                .register_activities(crate::activities::Activities)
                .register_workflow::<crate::kitchen_sink::KitchenSink>()?
                .build(),
            Self::Helloworld => options
                .register_workflow::<helloworld::HelloWorld>()?
                .build(),
        })
    }

    pub(crate) fn require_project(self) -> Result<()> {
        ensure!(
            self == Self::Helloworld,
            "no project handlers registered for app {self:?}"
        );
        Ok(())
    }

    /// Per-run project setup; helloworld needs none.
    pub(crate) async fn init(self, _client: &Client, _context: &InitContext) -> Result<()> {
        match self {
            Self::Helloworld => Ok(()),
            Self::Worker => unreachable!("project::run requires a project app"),
        }
    }

    pub(crate) async fn execute(
        self,
        client: &Client,
        run: &InitContext,
        context: ExecuteContext,
    ) -> Result<()> {
        match self {
            Self::Helloworld => helloworld::execute(client, run, context).await,
            Self::Worker => unreachable!("project::run requires a project app"),
        }
    }
}
