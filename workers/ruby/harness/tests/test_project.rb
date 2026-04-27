# frozen_string_literal: true

require 'bundler/setup'
require 'minitest/autorun'
require 'securerandom'
require 'temporalio/testing/workflow_environment'
require 'temporalio/worker'
require 'temporalio/workflow'
require 'grpc'
require 'harness'

class HarnessProjectTest < Minitest::Test
  class ProjectHarnessEchoWorkflow < Temporalio::Workflow::Definition
    def execute(payload)
      payload
    end
  end

  def test_init_rejects_invalid_tls_configuration
    server = Harness::ProjectServiceServer.new(
      Harness::ProjectHandlers.new(execute: ->(_client, _context) {}),
      ->(_config) { Object.new }
    )

    error = assert_raises(GRPC::BadStatus) do
      server.init(
        make_init_request(
          connect_options: Temporal::Omes::Projects::V1::ConnectOptions.new(
            namespace: 'default',
            server_address: 'localhost:7233',
            enable_tls: true,
            tls_cert_path: '/tmp/cert.pem',
            tls_key_path: ''
          )
        ),
        nil
      )
    end

    assert_equal GRPC::Core::StatusCodes::INVALID_ARGUMENT, error.code
    assert_equal 'Client cert specified, but not client key!', error.details
  end

  def test_init_passes_run_metadata_to_handler
    client = Object.new
    init_calls = []
    server = Harness::ProjectServiceServer.new(
      Harness::ProjectHandlers.new(
        execute: ->(_handler_client, _context) {},
        init: lambda do |handler_client, context|
          init_calls << [handler_client, context]
        end
      ),
      lambda do |given_config|
        assert_equal 'localhost:7233', given_config.target_host
        assert_equal 'default', given_config.namespace
        assert_equal 'token', given_config.api_key
        assert_nil given_config.tls
        assert_instance_of Temporalio::Runtime, given_config.runtime
        client
      end
    )

    response = server.init(make_init_request, nil)
    assert_instance_of Temporal::Omes::Projects::V1::InitResponse, response

    assert_equal 1, init_calls.length
    handler_client, init_context = init_calls.first
    assert_same client, handler_client
    assert_equal 'run-id', init_context.run.run_id
    assert_equal 'exec-id', init_context.run.execution_id
    assert_equal 'task-queue', init_context.task_queue
    assert_equal '{"hello":"world"}'.b, init_context.config_json
  end

  def test_execute_requires_init
    server = Harness::ProjectServiceServer.new(
      Harness::ProjectHandlers.new(execute: ->(_client, _context) {}),
      ->(_config) { Object.new }
    )

    error = assert_raises(GRPC::BadStatus) do
      server.execute(make_execute_request, nil)
    end

    assert_equal GRPC::Core::StatusCodes::FAILED_PRECONDITION, error.code
    assert_equal 'Init must be called before Execute', error.details
  end

  def test_execute_passes_iteration_payload_and_run_metadata
    client = Object.new
    execute_calls = []
    server = Harness::ProjectServiceServer.new(
      Harness::ProjectHandlers.new(
        execute: lambda do |handler_client, context|
          execute_calls << [handler_client, context]
        end
      ),
      ->(_config) { client }
    )

    server.init(make_init_request, nil)
    response = server.execute(make_execute_request, nil)
    assert_instance_of Temporal::Omes::Projects::V1::ExecuteResponse, response

    assert_equal 1, execute_calls.length
    handler_client, execute_context = execute_calls.first
    assert_same client, handler_client
    assert_equal 7, execute_context.iteration
    assert_equal 'payload'.b, execute_context.payload
    assert_equal 'task-queue', execute_context.task_queue
    assert_equal 'run-id', execute_context.run.run_id
    assert_equal 'exec-id', execute_context.run.execution_id
  end

  def test_client_factory_failure_maps_to_internal_error
    server = Harness::ProjectServiceServer.new(
      Harness::ProjectHandlers.new(execute: ->(_client, _context) {}),
      ->(_config) { raise 'boom' }
    )

    error = assert_raises(GRPC::BadStatus) do
      server.init(make_init_request, nil)
    end

    assert_equal GRPC::Core::StatusCodes::INTERNAL, error.code
    assert_equal 'failed to create client: boom', error.details
  end

  def test_init_handler_failure_does_not_leave_server_initialized
    server = Harness::ProjectServiceServer.new(
      Harness::ProjectHandlers.new(
        execute: ->(_client, _context) {},
        init: ->(_client, _context) { raise 'bad init' }
      ),
      ->(_config) { Object.new }
    )

    error = assert_raises(GRPC::BadStatus) do
      server.init(make_init_request, nil)
    end

    assert_equal GRPC::Core::StatusCodes::INTERNAL, error.code
    assert_equal 'init handler failed: bad init', error.details

    execute_error = assert_raises(GRPC::BadStatus) do
      server.execute(make_execute_request, nil)
    end
    assert_equal GRPC::Core::StatusCodes::FAILED_PRECONDITION, execute_error.code
    assert_equal 'Init must be called before Execute', execute_error.details
  end

  def test_execute_handler_failure_maps_to_internal_error
    server = Harness::ProjectServiceServer.new(
      Harness::ProjectHandlers.new(
        execute: ->(_client, _context) { raise 'bad execute' }
      ),
      ->(_config) { Object.new }
    )

    server.init(make_init_request, nil)

    error = assert_raises(GRPC::BadStatus) do
      server.execute(make_execute_request, nil)
    end

    assert_equal GRPC::Core::StatusCodes::INTERNAL, error.code
    assert_equal 'execute handler failed: bad execute', error.details
  end

  def test_project_server_executes_workflow_against_real_temporal_server
    events = []
    task_queue = "project-harness-e2e-#{SecureRandom.uuid}"

    init_handler = lambda do |handler_client, context|
      events << [:init, handler_client, context]
    end
    execute_handler = lambda do |handler_client, context|
      result = handler_client.execute_workflow(
        ProjectHarnessEchoWorkflow,
        context.payload,
        id: "#{context.run.execution_id}-#{context.iteration}",
        task_queue: context.task_queue
      )
      events << [:execute, handler_client, context, result]
    end

    Temporalio::Testing::WorkflowEnvironment.start_local do |env|
      worker = Temporalio::Worker.new(
        client: env.client,
        task_queue: task_queue,
        workflows: [ProjectHarnessEchoWorkflow]
      )

      worker.run do
        grpc_server = GRPC::RpcServer.new
        grpc_server.handle(
          Harness::ProjectServiceServer.new(
            Harness::ProjectHandlers.new(execute: execute_handler, init: init_handler),
            Harness.method(:default_client_factory)
          )
        )
        port = grpc_server.add_http2_port('127.0.0.1:0', :this_port_is_insecure)
        refute_equal 0, port

        server_thread = Thread.new { grpc_server.run }
        begin
          assert grpc_server.wait_till_running(5)
          stub = Temporal::Omes::Projects::V1::ProjectService::Stub.new(
            "127.0.0.1:#{port}",
            :this_channel_is_insecure
          )

          stub.init(
            make_init_request(
              task_queue: task_queue,
              connect_options: Temporal::Omes::Projects::V1::ConnectOptions.new(
                namespace: 'default',
                server_address: env.client.connection.target_host
              )
            )
          )
          stub.execute(make_execute_request(task_queue: task_queue))
        ensure
          grpc_server.stop if grpc_server.running?
          server_thread.join
        end
      end
    end

    assert_equal 2, events.length
    init_kind, init_client, init_context = events[0]
    execute_kind, execute_client, execute_context, execute_result = events[1]

    assert_equal :init, init_kind
    assert_equal 'run-id', init_context.run.run_id
    assert_equal 'exec-id', init_context.run.execution_id
    assert_equal task_queue, init_context.task_queue
    assert_equal '{"hello":"world"}'.b, init_context.config_json

    assert_equal :execute, execute_kind
    assert_same init_client, execute_client
    assert_equal 'run-id', execute_context.run.run_id
    assert_equal 'exec-id', execute_context.run.execution_id
    assert_equal task_queue, execute_context.task_queue
    assert_equal 7, execute_context.iteration
    assert_equal 'payload'.b, execute_context.payload
    assert_equal 'payload'.b, execute_result
  end

  private

  def make_init_request(
    execution_id: 'exec-id',
    run_id: 'run-id',
    task_queue: 'task-queue',
    connect_options: nil,
    config_json: '{"hello":"world"}'.b
  )
    Temporal::Omes::Projects::V1::InitRequest.new(
      execution_id: execution_id,
      run_id: run_id,
      task_queue: task_queue,
      connect_options: connect_options || Temporal::Omes::Projects::V1::ConnectOptions.new(
        namespace: 'default',
        server_address: 'localhost:7233',
        auth_header: 'Bearer token',
        enable_tls: false
      ),
      config_json: config_json
    )
  end

  def make_execute_request(
    iteration: 7,
    task_queue: 'task-queue',
    payload: 'payload'.b
  )
    Temporal::Omes::Projects::V1::ExecuteRequest.new(
      iteration: iteration,
      task_queue: task_queue,
      payload: payload
    )
  end
end
