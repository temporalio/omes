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

    register(
      THROUGHPUT_STRESS_BASELINE_PROFILE,
      {
        tuner: Temporalio::Worker::Tuner.create_fixed(
          workflow_slots: 8,
          activity_slots: 32,
          local_activity_slots: 32
        ),
        max_cached_workflows: 50,
        max_concurrent_workflow_task_polls: 2,
        max_concurrent_activity_task_polls: 4
      }
    )
  end
end
