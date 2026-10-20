//! Keep legacy compatibility-set versioning distinct from deployment versioning.
use anyhow::{Result, bail, ensure};
use clap::Args;
use temporalio_common::worker::{
    VersioningBehavior, WorkerDeploymentOptions, WorkerDeploymentVersion,
};

#[derive(Args, Default)]
#[group(id = "worker-versioning")]
pub(super) struct Options {
    /// Deprecated Build ID compatibility-set versioning; not supported by the Rust SDK.
    #[arg(long, default_value = "")]
    build_id: String,
    /// Worker Deployment name; requires --deployment-build-id.
    #[arg(long, default_value = "")]
    deployment_name: String,
    /// Fixed Build ID within the Worker Deployment.
    #[arg(long, default_value = "")]
    deployment_build_id: String,
    /// pinned or auto-upgrade; defaults to auto-upgrade with a deployment.
    #[arg(long, default_value = "")]
    default_versioning_behavior: String,
}

impl Options {
    pub(super) fn deployment_options(
        &self,
        profile: &str,
    ) -> Result<Option<WorkerDeploymentOptions>> {
        // Worker profiles replace the CLI options, including versioning and its
        // validation, so these constraints can't be expressed in clap.
        if !profile.is_empty() {
            return Ok(None);
        }
        if !self.deployment_name.is_empty() {
            ensure!(
                self.build_id.is_empty(),
                "--build-id and --deployment-name are mutually exclusive; use --deployment-build-id with --deployment-name"
            );
            ensure!(
                !self.deployment_build_id.is_empty(),
                "--deployment-build-id is required when --deployment-name is set"
            );
            let behavior = match self.default_versioning_behavior.as_str() {
                "" | "auto-upgrade" => VersioningBehavior::AutoUpgrade,
                "pinned" => VersioningBehavior::Pinned,
                value => bail!(
                    "invalid --default-versioning-behavior {value:?} (expected pinned or auto-upgrade)"
                ),
            };
            return Ok(Some(
                WorkerDeploymentOptions::new(
                    WorkerDeploymentVersion::builder()
                        .deployment_name(&self.deployment_name)
                        .build_id(&self.deployment_build_id)
                        .build(),
                )
                .use_worker_versioning(true)
                .default_versioning_behavior(behavior)
                .build(),
            ));
        }
        ensure!(
            self.deployment_build_id.is_empty(),
            "--deployment-build-id requires --deployment-name"
        );
        ensure!(
            self.build_id.is_empty(),
            "--build-id requests legacy compatibility-set versioning, which the Rust SDK does not support; use --deployment-name and --deployment-build-id for Worker Deployment Versioning"
        );
        // Without a deployment, --default-versioning-behavior has no effect, and
        // the worker keeps the SDK's default Build ID without opting into versioning.
        Ok(None)
    }
}
