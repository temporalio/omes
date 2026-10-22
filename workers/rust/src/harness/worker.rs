//! Worker CLI options, telemetry, tuning, and multi-queue lifecycle.
use super::{client::ConnectOptions, metrics, tuning, versioning};
use crate::apps::App;
use anyhow::{Result, bail};
use clap::Args;
use futures::{StreamExt, stream::FuturesUnordered};
use std::sync::Arc;
use temporalio_common::telemetry::TelemetryOptions;
use temporalio_sdk::{Runtime, Worker, runtime::RuntimeOptions};
use tracing_subscriber::EnvFilter;

#[derive(Args)]
#[group(id = "worker-options")]
pub(crate) struct Options {
    #[arg(short = 'q', long, default_value = "omes")]
    task_queue: String,
    #[arg(long, default_value_t = 0)]
    task_queue_suffix_index_start: u32,
    #[arg(long, default_value_t = 0)]
    task_queue_suffix_index_end: u32,
    #[command(flatten)]
    pub(super) connect: ConnectOptions,
    #[arg(long, default_value = "info")]
    log_level: String,
    #[arg(long, default_value = "console", value_parser = ["console", "json"])]
    log_encoding: String,
    #[arg(long)]
    prom_listen_address: Option<String>,
    #[command(flatten)]
    tuning: tuning::Options,
    #[command(flatten)]
    versioning: versioning::Options,
    #[arg(long, default_value_t = false, action = clap::ArgAction::Set)]
    err_on_unimplemented: bool,
}

impl Options {
    fn queues(&self) -> Result<Vec<String>> {
        anyhow::ensure!(!self.task_queue.is_empty(), "task queue is required");
        anyhow::ensure!(
            self.task_queue_suffix_index_start <= self.task_queue_suffix_index_end,
            "task queue suffix start exceeds end"
        );
        Ok(if self.task_queue_suffix_index_end == 0 {
            vec![self.task_queue.clone()]
        } else {
            (self.task_queue_suffix_index_start..=self.task_queue_suffix_index_end)
                .map(|i| format!("{}-{i}", self.task_queue))
                .collect()
        })
    }
}

pub(crate) async fn run(app: App, options: Options) -> Result<()> {
    let queues = options.queues()?;
    crate::wire::set_err_on_unimplemented(options.err_on_unimplemented);
    let profile = std::env::var("OMES_WORKER_PROFILE").unwrap_or_default();
    let deployment_options = options.versioning.deployment_options(&profile)?;
    let tls = options.connect.tls_options()?;
    let builder = tracing_subscriber::fmt()
        .with_env_filter(log_filter(&options.log_level)?)
        .with_ansi(false);
    let subscriber: Arc<dyn tracing::Subscriber + Send + Sync> = if options.log_encoding == "json" {
        Arc::new(builder.json().finish())
    } else {
        Arc::new(builder.finish())
    };
    tracing::subscriber::set_global_default(subscriber.clone())?;
    let exporter = match options.prom_listen_address.as_deref() {
        Some(address) if !address.is_empty() => Some(metrics::start(address).await?),
        _ => None,
    };
    let telemetry_options = TelemetryOptions::builder()
        .subscriber_override(subscriber)
        .maybe_metrics(exporter.as_ref().map(|e| e.meter.clone()))
        .build();
    // Runtime doesn't expose its telemetry instance. Initialize the client's from
    // the same options and exporter so client and worker metrics share a prefix
    // and labels.
    let client_telemetry = temporalio_common::telemetry::telemetry_init(telemetry_options.clone())?;
    let runtime = Runtime::from_current_tokio(
        RuntimeOptions::builder()
            .telemetry_options(telemetry_options)
            .build()
            .map_err(anyhow::Error::msg)?,
    )?;
    let client = options
        .connect
        .client_with_tls(tls, client_telemetry.get_temporal_metric_meter())
        .await?;
    let mut workers = Vec::new();
    let mut shutdown = Vec::new();
    for queue in queues {
        let mut config = app.worker_options(&queue)?;
        options.tuning.apply(&mut config, &profile)?;
        if let Some(deployment_options) = &deployment_options {
            config.deployment_options = deployment_options.clone();
        }
        tracing::info!(task_queue = queue, namespace = options.connect.namespace,
            profile, tuner = ?config.tuner, cache = config.max_cached_workflows,
            workflow_pollers = ?config.workflow_task_poller_behavior,
            activity_pollers = ?config.activity_task_poller_behavior,
            activities_per_second = ?config.max_worker_activities_per_second,
            ?app, "Starting Rust worker");
        let worker = Worker::new(&runtime, client.clone(), config)?;
        shutdown.push(worker.shutdown_handle());
        workers.push(worker);
    }
    // SDK worker futures are intentionally !Send. Poll all queues together on
    // this task rather than spawning them onto Tokio's thread pool.
    let mut tasks = FuturesUnordered::new();
    for mut worker in workers {
        tasks.push(async move { worker.run().await });
    }
    let mut failure = None;
    let signal = tokio::select! {
        signal = super::shutdown_signal() => signal,
        result = tasks.next() => {
            failure = result;
            Ok(())
        }
    };
    for stop in shutdown {
        stop();
    }
    while let Some(result) = tasks.next().await {
        if result.is_err() && failure.is_none() {
            failure = Some(result);
        }
    }
    signal?;
    if let Some(result) = failure {
        result?;
        bail!("worker exited unexpectedly");
    }
    drop(exporter);
    Ok(())
}

/// Tracing has no panic/fatal levels, and every tracing level is below them.
fn log_filter(value: &str) -> Result<EnvFilter> {
    let level = value.to_ascii_lowercase();
    let directive = match level.as_str() {
        "panic" | "fatal" => "off",
        "trace" | "debug" | "info" | "warn" | "error" | "off" => &level,
        _ => bail!("invalid log level: {value}"),
    };
    Ok(EnvFilter::new(directive))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{Cli, Command};
    use clap::Parser;
    use temporalio_sdk::WorkerOptions;

    fn options(flags: &[&str]) -> Options {
        let cli = Cli::try_parse_from(
            ["omes", "worker", "--task-queue", "test"]
                .into_iter()
                .chain(flags.iter().copied()),
        )
        .unwrap();
        let Command::Worker(options) = cli.command else {
            panic!("expected worker")
        };
        *options
    }

    #[test]
    fn queue_ranges_match_omes() {
        assert_eq!(options(&[]).queues().unwrap(), ["test"]);
        assert_eq!(
            options(&[
                "--task-queue-suffix-index-start=2",
                "--task-queue-suffix-index-end=3"
            ])
            .queues()
            .unwrap(),
            ["test-2", "test-3"]
        );
        assert!(
            options(&["--task-queue-suffix-index-start=2"])
                .queues()
                .is_err()
        );
    }

    #[test]
    fn cli_accepts_passthrough_tuning_and_boolean_values() {
        let opts = options(&[
            "--max-concurrent-activities=7",
            "--workflow-poller-autoscale-max=3",
            "--err-on-unimplemented=false",
        ]);
        let mut config = WorkerOptions::new("test").build();
        opts.tuning.apply(&mut config, "").unwrap();
        assert!(matches!(
            config.workflow_task_poller_behavior,
            Some(temporalio_sdk::runtime::PollerBehavior::Autoscaling(_))
        ));
    }
}
