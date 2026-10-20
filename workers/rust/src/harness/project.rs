//! gRPC ProjectService server; apps implement per-run setup and iterations.
use super::client::ConnectOptions;
use crate::apps::{App, ExecuteContext, InitContext};
use anyhow::Result;
use clap::Args;
use std::sync::Arc;
use temporalio_client::Client;
use tokio::sync::RwLock;
use tokio_util::sync::CancellationToken;
use tonic::{Request, Response, Status, transport::Server};

#[allow(clippy::all, dead_code)]
mod api {
    include!(concat!(env!("OUT_DIR"), "/temporal.omes.projects.v1.rs"));
}
use api::project_service_server::{ProjectService, ProjectServiceServer};

#[derive(Args)]
pub(crate) struct Options {
    #[arg(long, default_value_t = 8080)]
    port: u16,
}

struct Run {
    client: Client,
    context: InitContext,
}

struct Service {
    app: App,
    // Init publishes a complete run atomically; concurrent Execute calls clone
    // it and never hold a lock across workflow execution.
    run: RwLock<Option<Arc<Run>>>,
    stop: CancellationToken,
}

fn validate_init(request: &api::InitRequest) -> Result<ConnectOptions, Status> {
    for (name, value) in [
        ("task_queue", request.task_queue.as_str()),
        ("execution_id", request.execution_id.as_str()),
        ("run_id", request.run_id.as_str()),
    ] {
        if value.is_empty() {
            return Err(Status::invalid_argument(format!("{name} required")));
        }
    }
    let connect = request
        .connect_options
        .as_ref()
        .ok_or_else(|| Status::invalid_argument("connect_options required"))?;
    if connect.server_address.is_empty() || connect.namespace.is_empty() {
        return Err(Status::invalid_argument(
            "server_address and namespace required",
        ));
    }
    fn optional(value: &str) -> Option<String> {
        (!value.is_empty()).then(|| value.to_owned())
    }
    Ok(ConnectOptions {
        server_address: connect.server_address.clone(),
        namespace: connect.namespace.clone(),
        auth_header: optional(&connect.auth_header),
        tls: connect.enable_tls,
        tls_cert_path: optional(&connect.tls_cert_path),
        tls_key_path: optional(&connect.tls_key_path),
        tls_server_name: optional(&connect.tls_server_name),
        disable_tls_host_verification: connect.disable_host_verification,
        tls_combined_path: None,
    })
}

#[tonic::async_trait]
impl ProjectService for Service {
    async fn init(
        &self,
        request: Request<api::InitRequest>,
    ) -> Result<Response<api::InitResponse>, Status> {
        let request = request.into_inner();
        let connect = validate_init(&request)?;
        let tls = connect
            .tls_options()
            .map_err(|e| Status::invalid_argument(e.to_string()))?;
        let initialize = async {
            let client = connect
                .client_with_tls(tls, None)
                .await
                .map_err(|e| Status::internal(format!("failed to create client: {e:#}")))?;
            let context = InitContext {
                execution_id: request.execution_id,
                run_id: request.run_id,
                task_queue: request.task_queue,
                config_json: request.config_json,
            };
            self.app
                .init(&client, &context)
                .await
                .map_err(|e| Status::internal(format!("init handler failed: {e:#}")))?;
            *self.run.write().await = Some(Arc::new(Run { client, context }));
            Ok(Response::new(api::InitResponse {}))
        };
        tokio::select! {
            result = initialize => result,
            _ = self.stop.cancelled() => Err(Status::unavailable("project server shutting down")),
        }
    }

    async fn execute(
        &self,
        request: Request<api::ExecuteRequest>,
    ) -> Result<Response<api::ExecuteResponse>, Status> {
        let request = request.into_inner();
        if request.task_queue.is_empty() {
            return Err(Status::invalid_argument("task_queue required"));
        }
        let run = self
            .run
            .read()
            .await
            .clone()
            .ok_or_else(|| Status::failed_precondition("Init must be called before Execute"))?;
        let execute = self.app.execute(
            &run.client,
            &run.context,
            ExecuteContext {
                iteration: request.iteration,
                task_queue: request.task_queue,
                payload: request.payload,
            },
        );
        tokio::select! {
            result = execute => result
                .map(|()| Response::new(api::ExecuteResponse {}))
                .map_err(|e| Status::internal(format!("execute handler failed: {e:#}"))),
            _ = self.stop.cancelled() => Err(Status::unavailable("project server shutting down")),
        }
    }
}

pub(crate) async fn run(app: App, options: Options) -> Result<()> {
    app.require_project()?;
    tracing_subscriber::fmt()
        .with_ansi(false)
        .try_init()
        .map_err(|e| anyhow::anyhow!("{e}"))?;
    let stop = CancellationToken::new();
    let service = Service {
        app,
        run: RwLock::new(None),
        stop: stop.clone(),
    };
    let server = Server::builder()
        .add_service(ProjectServiceServer::new(service))
        .serve_with_shutdown(
            ([0, 0, 0, 0], options.port).into(),
            stop.clone().cancelled_owned(),
        );
    tokio::pin!(server);
    tracing::info!(port = options.port, "Project server listening");
    tokio::select! {
        result = &mut server => result?,
        signal = super::shutdown_signal() => {
            stop.cancel();
            // Cancellation releases in-flight Init/Execute futures before
            // waiting for gRPC graceful shutdown and dropping run clients.
            server.await?;
            signal?;
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn request() -> api::InitRequest {
        api::InitRequest {
            execution_id: "execution".into(),
            run_id: "run".into(),
            task_queue: "queue".into(),
            connect_options: Some(api::ConnectOptions {
                namespace: "ns".into(),
                server_address: "localhost:7233".into(),
                auth_header: "Bearer exact".into(),
                enable_tls: true,
                tls_server_name: "temporal.example".into(),
                ..Default::default()
            }),
            config_json: br#"{"opaque":true}"#.to_vec(),
        }
    }

    #[test]
    fn init_preserves_connection_options_and_certificate_verification_opt_out() {
        let mut request = request();
        let options = validate_init(&request).unwrap();
        assert_eq!(options.auth_header.as_deref(), Some("Bearer exact"));
        assert_eq!(options.namespace, "ns");
        assert_eq!(
            options.tls_options().unwrap().unwrap().domain.as_deref(),
            Some("temporal.example")
        );
        request
            .connect_options
            .as_mut()
            .unwrap()
            .disable_host_verification = true;
        let tls = validate_init(&request)
            .unwrap()
            .tls_options()
            .unwrap()
            .unwrap();
        assert_eq!(tls.domain.as_deref(), Some("temporal.example"));
        assert!(tls.server_cert_verifier.is_some());
        request.run_id.clear();
        assert_eq!(
            validate_init(&request).err().unwrap().code(),
            tonic::Code::InvalidArgument
        );
    }

    #[tokio::test]
    async fn execute_requires_init_and_queue() {
        let service = Service {
            app: App::Helloworld,
            run: RwLock::new(None),
            stop: CancellationToken::new(),
        };
        let error = service
            .execute(Request::new(api::ExecuteRequest::default()))
            .await
            .unwrap_err();
        assert_eq!(error.code(), tonic::Code::InvalidArgument);
        let error = service
            .execute(Request::new(api::ExecuteRequest {
                task_queue: "queue".into(),
                ..Default::default()
            }))
            .await
            .unwrap_err();
        assert_eq!(error.code(), tonic::Code::FailedPrecondition);
    }
}
