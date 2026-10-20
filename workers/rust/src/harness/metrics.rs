//! The SDK's Prometheus exporter, which always serves `/metrics`.
use anyhow::{Context, Result};
use std::sync::Arc;
use temporalio_common::telemetry::{
    PrometheusExporterOptions, metrics::CoreMeter, start_prometheus_metric_exporter,
};
use tokio::{net::lookup_host, task::AbortHandle};

pub(super) struct Exporter {
    pub(super) meter: Arc<dyn CoreMeter>,
    abort: AbortHandle,
}

impl Drop for Exporter {
    fn drop(&mut self) {
        self.abort.abort();
    }
}

pub(super) async fn start(address: &str) -> Result<Exporter> {
    // ":port" means all interfaces.
    let address = if address.starts_with(':') {
        format!("0.0.0.0{address}")
    } else {
        address.to_owned()
    };
    let address = lookup_host(address.as_str())
        .await
        .context("resolve Prometheus listen address (expected host:port)")?
        .next()
        .context("Prometheus listen address resolved to no addresses")?;
    let sdk = start_prometheus_metric_exporter(
        PrometheusExporterOptions::builder()
            .socket_addr(address)
            .use_seconds_for_durations(true)
            .build(),
    )
    .context("bind Prometheus listen address")?;
    Ok(Exporter {
        meter: sdk.meter,
        abort: sdk.abort_handle,
    })
}
