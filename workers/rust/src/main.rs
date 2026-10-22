// Workflow-facing helpers return the SDK's native ApplicationFailure, which is
// large; boxing it would only add conversions at every SDK boundary.
#![allow(clippy::result_large_err)]

mod activities;
mod apps;
mod harness;
mod kitchen_sink;
mod nexus;
mod proto;
mod resource_activity;
mod standalone_activity;
mod standalone_activity_operator;
mod standalone_nexus;
mod wire;
mod workflow_nexus;

use anyhow::Result;
use clap::{Parser, Subcommand};
use harness::{project, worker};

#[derive(Parser)]
struct Cli {
    /// Registered application: worker (Kitchen Sink) or helloworld.
    #[arg(long, default_value = "worker")]
    app: apps::App,
    #[command(subcommand)]
    command: Command,
}

#[derive(Subcommand)]
enum Command {
    Worker(Box<worker::Options>),
    ProjectServer(project::Options),
}

#[tokio::main]
async fn main() -> Result<()> {
    let cli = Cli::parse();
    match cli.command {
        Command::Worker(options) => worker::run(cli.app, *options).await,
        Command::ProjectServer(options) => project::run(cli.app, options).await,
    }
}
