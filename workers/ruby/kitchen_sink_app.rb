# frozen_string_literal: true

require 'temporalio/worker'
require_relative 'activities'
require_relative 'kitchen_sink'
require_relative 'projects/harness'

module KitchenSinkApp
  module_function

  def app
    Harness::App.new(
      worker: method(:build_worker),
      client_factory: Harness.method(:default_client_factory)
    )
  end

  def build_worker(client, context)
    Temporalio::Worker.new(
      client: client,
      task_queue: context.task_queue,
      workflows: [KitchenSinkWorkflow],
      activities: [
        NoopActivity.new,
        DelayActivity.new,
        PayloadActivity.new,
        RetryableErrorActivity.new,
        TimeoutActivity.new,
        HeartbeatActivity.new,
        ClientActivity.new(client, err_on_unimplemented: context.err_on_unimplemented)
      ],
      logger: context.logger,
      **context.worker_kwargs
    )
  end
end
