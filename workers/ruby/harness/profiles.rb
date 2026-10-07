# frozen_string_literal: true

require 'temporalio/worker'

module Harness
  module Profiles
    WORKER_PROFILE_ENV_VAR = 'OMES_WORKER_PROFILE'
    RESOURCE_BASED_DEFAULT_PROFILE = 'resource-based-default'
    THROUGHPUT_STRESS_BASELINE_PROFILE = 'throughput-stress-baseline'

    @registry = {}

    class << self
      def register(name, profile)
        @registry[name] = profile
      end

      def lookup(name)
        @registry.fetch(name).dup
      rescue KeyError
        raise ArgumentError, "Unknown worker profile #{name.inspect}"
      end
    end

    register(
      RESOURCE_BASED_DEFAULT_PROFILE,
      {
        tuner: Temporalio::Worker::Tuner.create_resource_based(
          target_memory_usage: 0.8,
          target_cpu_usage: 0.8
        )
      }
    )

    def self.throughput_stress_profile(scale)
      {
        tuner: Temporalio::Worker::Tuner.create_fixed(
          workflow_slots: 8 * scale,
          activity_slots: 32 * scale,
          local_activity_slots: 32 * scale
        ),
        max_cached_workflows: 50 * scale,
        max_concurrent_workflow_task_polls: 2 * scale,
        max_concurrent_activity_task_polls: 4 * scale
      }
    end

    register(THROUGHPUT_STRESS_BASELINE_PROFILE, throughput_stress_profile(1))
    # The baseline profile with every limit scaled, e.g. throughput-stress-baseline-x4.
    [2, 4, 8, 16, 32, 64].each do |scale|
      register("#{THROUGHPUT_STRESS_BASELINE_PROFILE}-x#{scale}", throughput_stress_profile(scale))
    end
  end
end
