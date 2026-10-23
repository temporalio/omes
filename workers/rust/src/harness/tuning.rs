use anyhow::{Result, bail};
use clap::Args;
use temporalio_sdk::runtime::{AutoscalingOptions, PollerBehavior};
use temporalio_sdk::{
    WorkerOptions,
    runtime::worker_tuner::{
        FixedSizeSlotSupplier, ResourceBasedTuner, ResourceBasedTunerOptions, TunerHolder,
        WorkerTuner,
    },
};

#[derive(Args, Default)]
pub(super) struct Options {
    #[arg(long, default_value_t = 0)]
    max_concurrent_activity_pollers: usize,
    #[arg(long, default_value_t = 0)]
    max_concurrent_workflow_pollers: usize,
    #[arg(long, default_value_t = 0)]
    activity_poller_autoscale_max: usize,
    #[arg(long, default_value_t = 0)]
    workflow_poller_autoscale_max: usize,
    #[arg(long, default_value_t = 0)]
    max_concurrent_activities: usize,
    #[arg(long, default_value_t = 0)]
    max_concurrent_workflow_tasks: usize,
    #[arg(long, default_value_t = 0.0)]
    activities_per_second: f64,
}

const DEFAULT_SLOTS: usize = 100;
// Matches the SDK's default fixed poller count.
const DEFAULT_POLLERS: usize = 5;

fn fixed(workflows: usize, activities: usize, local_activities: usize) -> WorkerTuner {
    TunerHolder::builder()
        .workflow_task_slot_supplier(FixedSizeSlotSupplier::new(workflows))
        .activity_task_slot_supplier(FixedSizeSlotSupplier::new(activities))
        .local_activity_task_slot_supplier(FixedSizeSlotSupplier::new(local_activities))
        .nexus_task_slot_supplier(FixedSizeSlotSupplier::new(DEFAULT_SLOTS))
        .build()
        .into()
}

fn pollers(max: usize, autoscale: usize) -> Option<PollerBehavior> {
    if autoscale > 0 {
        Some(PollerBehavior::Autoscaling(
            AutoscalingOptions::builder()
                .minimum(1)
                .maximum(autoscale)
                // Start at 5 pollers so autoscaling behaves comparably across SDKs.
                .initial(autoscale.min(5))
                .build(),
        ))
    } else if max > 0 {
        Some(PollerBehavior::SimpleMaximum(max))
    } else {
        // Set explicitly rather than left to the SDK so the worker is not auto-enrolled into
        // poller autoscaling when the namespace advertises it.
        Some(PollerBehavior::SimpleMaximum(DEFAULT_POLLERS))
    }
}

impl Options {
    pub(super) fn apply(&self, config: &mut WorkerOptions, profile: &str) -> Result<()> {
        if profile == "resource-based-default" {
            config.tuner = ResourceBasedTuner::builder()
                .tuner_options(
                    ResourceBasedTunerOptions::builder()
                        .target_memory_usage(0.8)
                        .target_cpu_usage(0.8)
                        .build(),
                )
                .build()
                .into();
        } else if !profile.is_empty() {
            let scale = if profile == "throughput-stress-baseline" {
                1
            } else if let Some(suffix) = profile.strip_prefix("throughput-stress-baseline-x") {
                suffix.parse::<usize>().unwrap_or(0)
            } else {
                0
            };
            if ![1, 2, 4, 8, 16, 32, 64].contains(&scale) {
                bail!("unknown worker profile {profile:?}");
            }
            config.max_cached_workflows = 50 * scale;
            config.tuner = fixed(8 * scale, 32 * scale, 32 * scale);
            config.workflow_task_poller_behavior = pollers(2 * scale, 0);
            config.activity_task_poller_behavior = pollers(4 * scale, 0);
        } else {
            if self.max_concurrent_workflow_tasks > 0 || self.max_concurrent_activities > 0 {
                let or_default = |n: usize| if n == 0 { DEFAULT_SLOTS } else { n };
                config.tuner = fixed(
                    or_default(self.max_concurrent_workflow_tasks),
                    or_default(self.max_concurrent_activities),
                    DEFAULT_SLOTS,
                );
            }
            config.workflow_task_poller_behavior = pollers(
                self.max_concurrent_workflow_pollers,
                self.workflow_poller_autoscale_max,
            );
            config.activity_task_poller_behavior = pollers(
                self.max_concurrent_activity_pollers,
                self.activity_poller_autoscale_max,
            );
            anyhow::ensure!(
                self.activities_per_second.is_finite() && self.activities_per_second >= 0.0,
                "activities-per-second must be finite and nonnegative"
            );
            if self.activities_per_second > 0.0 {
                config.max_worker_activities_per_second = Some(self.activities_per_second);
            }
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn profiles_replace_cli_tuning() {
        let options = Options {
            activities_per_second: f64::NAN,
            ..Default::default()
        };
        let mut config = WorkerOptions::new("test").build();
        assert!(options.apply(&mut config, "").is_err());
        options
            .apply(&mut config, "throughput-stress-baseline-x4")
            .unwrap();
        assert_eq!(config.max_cached_workflows, 200);
        assert_eq!(config.workflow_task_poller_behavior, pollers(8, 0));
        assert_eq!(config.activity_task_poller_behavior, pollers(16, 0));
        options
            .apply(&mut config, "resource-based-default")
            .unwrap();
        assert!(matches!(config.tuner, WorkerTuner::ResourceBased(_)));
        assert!(options.apply(&mut config, "unknown").is_err());
        assert!(
            options
                .apply(&mut config, "throughput-stress-baseline-x3")
                .is_err()
        );
    }

    #[test]
    fn autoscaling_takes_precedence_over_fixed_pollers() {
        assert!(matches!(pollers(9, 4), Some(PollerBehavior::Autoscaling(o))
            if o.maximum == 4 && o.minimum == 1 && o.initial == 4));
        assert_eq!(pollers(0, 0), Some(PollerBehavior::SimpleMaximum(DEFAULT_POLLERS)));
    }
}
