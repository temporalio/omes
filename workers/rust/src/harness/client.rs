//! Shared connection options for worker CLI and ProjectService initialization.
use anyhow::{Context, Result, bail};
use clap::Args;
use std::time::Duration;
use temporalio_client::{
    Client, ClientOptions, ClientTlsOptions, Connection, ConnectionOptions, TlsOptions, Url,
};

#[derive(Args, Default)]
pub(super) struct ConnectOptions {
    #[arg(long, default_value = "localhost:7233")]
    pub(super) server_address: String,
    #[arg(long, default_value = "default")]
    pub(super) namespace: String,
    #[arg(long, default_value_t = false, action = clap::ArgAction::Set)]
    pub(super) tls: bool,
    #[arg(long)]
    pub(super) tls_cert_path: Option<String>,
    #[arg(long)]
    pub(super) tls_key_path: Option<String>,
    #[arg(long, conflicts_with_all = ["tls_cert_path", "tls_key_path"])]
    pub(super) tls_combined_path: Option<String>,
    #[arg(long)]
    pub(super) tls_server_name: Option<String>,
    /// Disable server certificate trust and hostname verification (unsafe).
    #[arg(long, default_value_t = false, action = clap::ArgAction::Set)]
    pub(super) disable_tls_host_verification: bool,
    #[arg(long, env = "TEMPORAL_OMES_AUTH_HEADER", hide_env_values = true)]
    pub(super) auth_header: Option<String>,
}

impl ConnectOptions {
    pub(super) fn tls_options(&self) -> Result<Option<TlsOptions>> {
        let identity = if let Some(path) = &self.tls_combined_path {
            let pem = std::fs::read(path).context("read combined TLS PEM")?;
            Some(
                ClientTlsOptions::builder()
                    .client_cert(pem.clone())
                    .client_private_key(pem)
                    .build(),
            )
        } else {
            match (&self.tls_cert_path, &self.tls_key_path) {
                (Some(cert), Some(key)) => Some(
                    ClientTlsOptions::builder()
                        .client_cert(std::fs::read(cert).context("read TLS certificate")?)
                        .client_private_key(std::fs::read(key).context("read TLS key")?)
                        .build(),
                ),
                (None, None) => None,
                _ => bail!("TLS certificate and key must be supplied together"),
            }
        };
        // An auth header is an API key, which requires TLS.
        Ok(
            if self.tls || identity.is_some() || self.auth_header.is_some() {
                Some(
                    TlsOptions::builder()
                        .maybe_domain(self.tls_server_name.clone())
                        .maybe_client_tls_options(identity)
                        .maybe_server_cert_verifier(
                            self.disable_tls_host_verification
                                .then(super::tls::without_server_certificate_verification),
                        )
                        .build(),
                )
            } else {
                None
            },
        )
    }

    pub(super) async fn client_with_tls(
        &self,
        tls: Option<TlsOptions>,
        meter: Option<temporalio_common::telemetry::metrics::TemporalMeter>,
    ) -> Result<Client> {
        if self.disable_tls_host_verification && tls.is_some() {
            // Worker tracing is initialized after TLS option validation.
            tracing::warn!(
                "--disable-tls-host-verification disables server certificate trust and hostname verification; unsafe outside controlled tests"
            );
        }
        let scheme = if tls.is_some() { "https" } else { "http" };
        let target: Url = format!("{scheme}://{}", self.server_address)
            .parse()
            .context("server-address must be host:port")?;
        let api_key = self
            .auth_header
            .as_ref()
            .map(|value| value.strip_prefix("Bearer ").unwrap_or(value).to_owned());
        let connection = Connection::connect(
            ConnectionOptions::new(target)
                .maybe_tls_options(tls)
                .maybe_api_key(api_key)
                .maybe_metrics_meter(meter)
                .connect_timeout(Duration::from_secs(10))
                .build(),
        )
        .await?;
        Ok(Client::new(
            connection,
            ClientOptions::new(&self.namespace).build(),
        )?)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{Cli, Command};
    use clap::Parser;

    fn options(flags: &[&str]) -> ConnectOptions {
        let cli = Cli::try_parse_from(
            ["omes", "worker", "--task-queue", "test"]
                .into_iter()
                .chain(flags.iter().copied()),
        )
        .unwrap();
        let Command::Worker(options) = cli.command else {
            panic!("expected worker")
        };
        options.connect
    }

    #[test]
    fn tls_settings_match_go() {
        assert!(options(&["--tls-key-path=key"]).tls_options().is_err());
        assert!(options(&["--tls-cert-path=cert"]).tls_options().is_err());
        assert!(
            options(&["--tls-server-name=example.com"])
                .tls_options()
                .unwrap()
                .is_none()
        );
        assert!(
            options(&["--disable-tls-host-verification=true"])
                .tls_options()
                .unwrap()
                .is_none()
        );
        assert!(
            options(&["--auth-header=Bearer key"])
                .tls_options()
                .unwrap()
                .is_some()
        );
        assert!(
            Cli::try_parse_from([
                "omes",
                "worker",
                "--task-queue=test",
                "--tls-combined-path=combined",
                "--tls-key-path=key"
            ])
            .is_err()
        );
        let tls = options(&["--tls=true", "--tls-server-name=example.com"])
            .tls_options()
            .unwrap()
            .unwrap();
        assert_eq!(tls.domain.as_deref(), Some("example.com"));
        assert!(tls.server_cert_verifier.is_none());
        let tls = options(&[
            "--tls=true",
            "--tls-server-name=example.com",
            "--disable-tls-host-verification=true",
        ])
        .tls_options()
        .unwrap()
        .unwrap();
        assert_eq!(tls.domain.as_deref(), Some("example.com"));
        assert!(tls.server_cert_verifier.is_some());
    }
}
