# frozen_string_literal: true

require 'temporalio/worker'

module Harness
  module Profiles
    WORKER_PROFILE_ENV_VAR = 'OMES_WORKER_PROFILE'
    RESOURCE_BASED_DEFAULT_PROFILE = 'resource-based-default'

    @registry = {}

    module_function

    def register(name, profile)
      @registry[name] = profile
    end

    def lookup(name)
      @registry.fetch(name).dup
    rescue KeyError
      raise ArgumentError, "Unknown worker profile #{name.inspect}"
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
  end
end
