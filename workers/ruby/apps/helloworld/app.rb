# frozen_string_literal: true

require 'harness'
require 'temporalio/worker'
require 'temporalio/workflow'

module Apps
  module HelloWorld
    class HelloWorldWorkflow < Temporalio::Workflow::Definition
      workflow_name 'HelloWorldWorkflow'

      def execute(name)
        "Hello #{name}"
      end
    end

    module_function

    def app
      Harness::App.new(
        worker: method(:build_worker),
        client_factory: Harness.method(:default_client_factory),
        project: Harness::ProjectHandlers.new(execute: method(:execute_project_iteration))
      )
    end

    def build_worker(client, context)
      Temporalio::Worker.new(
        client: client,
        task_queue: context.task_queue,
        workflows: [HelloWorldWorkflow],
        logger: context.logger,
        **context.worker_kwargs
      )
    end

    def execute_project_iteration(client, context)
      result = client.execute_workflow(
        HelloWorldWorkflow,
        'World',
        id: "#{context.run.execution_id}-#{context.iteration}",
        task_queue: context.task_queue
      )
      puts result
    end
  end
end
