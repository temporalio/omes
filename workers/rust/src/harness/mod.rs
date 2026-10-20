//! Shared client setup, worker lifecycle, and project transport.
mod client;
mod metrics;
pub(crate) mod project;
mod tls;
mod tuning;
mod versioning;
pub(crate) mod worker;

use anyhow::Result;

async fn shutdown_signal() -> Result<()> {
    #[cfg(unix)]
    {
        let mut terminate =
            tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate())?;
        tokio::select! {
            result = tokio::signal::ctrl_c() => { result?; }
            _ = terminate.recv() => {}
        }
    }
    #[cfg(not(unix))]
    tokio::signal::ctrl_c().await?;
    Ok(())
}
