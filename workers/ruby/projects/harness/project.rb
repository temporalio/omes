# frozen_string_literal: true

require 'logger'
require 'optparse'
require 'grpc'

$LOAD_PATH.unshift(File.expand_path('..', __dir__)) unless $LOAD_PATH.include?(File.expand_path('..', __dir__))

require_relative 'client'
require_relative 'api/api_pb'
require_relative 'api/api_services_pb'

module Harness
  ProjectRunMetadata = Struct.new(
    :run_id,
    :execution_id,
    keyword_init: true
  )

  ProjectInitContext = Struct.new(
    :logger,
    :run,
    :task_queue,
    :config_json,
    keyword_init: true
  )

  ProjectExecuteContext = Struct.new(
    :logger,
    :run,
    :task_queue,
    :iteration,
    :payload,
    keyword_init: true
  )

  ProjectHandlers = Struct.new(
    :execute,
    :init,
    keyword_init: true
  )

  class ProjectServiceServer < Temporal::Omes::Projects::V1::ProjectService::Service
    def initialize(handlers, client_factory)
      @handlers = handlers
      @client_factory = client_factory
      @client = nil
      @run = nil
      @logger = Logger.new($stderr)
    end

    def init(request, _call)
      raise grpc_status(GRPC::Core::StatusCodes::INVALID_ARGUMENT, 'task_queue required') if request.task_queue.to_s.empty?
      raise grpc_status(GRPC::Core::StatusCodes::INVALID_ARGUMENT, 'execution_id required') if request.execution_id.to_s.empty?
      raise grpc_status(GRPC::Core::StatusCodes::INVALID_ARGUMENT, 'run_id required') if request.run_id.to_s.empty?

      connect_options = request.connect_options || Temporal::Omes::Projects::V1::ConnectOptions.new
      raise grpc_status(GRPC::Core::StatusCodes::INVALID_ARGUMENT, 'server_address required') if connect_options.server_address.to_s.empty?
      raise grpc_status(GRPC::Core::StatusCodes::INVALID_ARGUMENT, 'namespace required') if connect_options.namespace.to_s.empty?

      begin
        config = ClientHelpers.build_client_config(
          server_address: connect_options.server_address,
          namespace: connect_options.namespace,
          auth_header: connect_options.auth_header,
          tls: connect_options.enable_tls,
          tls_cert_path: connect_options.tls_cert_path,
          tls_key_path: connect_options.tls_key_path,
          tls_server_name: connect_options.tls_server_name.to_s.empty? ? nil : connect_options.tls_server_name,
          disable_host_verification: connect_options.disable_host_verification
        )
      rescue ArgumentError => error
        raise grpc_status(GRPC::Core::StatusCodes::INVALID_ARGUMENT, error.message)
      end

      begin
        client = @client_factory.call(config)
      rescue StandardError => error
        raise grpc_status(GRPC::Core::StatusCodes::INTERNAL, "failed to create client: #{error}")
      end

      run = ProjectRunMetadata.new(
        run_id: request.run_id,
        execution_id: request.execution_id
      )

      if @handlers.init
        begin
          @handlers.init.call(
            client,
            ProjectInitContext.new(
              logger: @logger,
              run: run,
              task_queue: request.task_queue,
              config_json: request.config_json
            )
          )
        rescue StandardError => error
          raise grpc_status(GRPC::Core::StatusCodes::INTERNAL, "init handler failed: #{error}")
        end
      end

      @client = client
      @run = run
      Temporal::Omes::Projects::V1::InitResponse.new
    rescue GRPC::BadStatus
      raise
    end

    def execute(request, _call)
      raise grpc_status(GRPC::Core::StatusCodes::INVALID_ARGUMENT, 'task_queue required') if request.task_queue.to_s.empty?
      raise grpc_status(GRPC::Core::StatusCodes::FAILED_PRECONDITION, 'Init must be called before Execute') if @client.nil? || @run.nil?

      @handlers.execute.call(
        @client,
        ProjectExecuteContext.new(
          logger: @logger,
          run: @run,
          task_queue: request.task_queue,
          iteration: request.iteration,
          payload: request.payload
        )
      )
      Temporal::Omes::Projects::V1::ExecuteResponse.new
    rescue GRPC::BadStatus
      raise
    rescue StandardError => error
      raise grpc_status(GRPC::Core::StatusCodes::INTERNAL, "execute handler failed: #{error}")
    end

    private

    def grpc_status(code, details)
      GRPC::BadStatus.new_status_exception(code, details)
    end

  end

  module ProjectCLI
    module_function

    def run_cli(handlers, client_factory, argv)
      options = { port: 8080 }
      parser = OptionParser.new do |opts|
        opts.banner = 'Usage: runner.rb project-server [options]'
        opts.on('--port PORT', Integer, 'gRPC listen port') { |value| options[:port] = value }
      end
      parser.parse!(Array(argv).dup)
      serve(handlers, client_factory, options[:port])
    end

    def serve(handlers, client_factory, port)
      logger = Logger.new($stderr)
      server = GRPC::RpcServer.new
      server.handle(ProjectServiceServer.new(handlers, client_factory))
      server.add_http2_port("0.0.0.0:#{port}", :this_port_is_insecure)
      logger.info("Project server listening on port #{port}")
      server.run_till_terminated_or_interrupted(['SIGINT'])
    end
  end
end
