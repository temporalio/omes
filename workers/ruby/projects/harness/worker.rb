# frozen_string_literal: true

require 'optparse'
require 'temporalio/worker'
require_relative 'client'
require_relative 'helpers'

module Harness
  WorkerContext = Data.define(
    :logger,
    :task_queue,
    :err_on_unimplemented,
    :worker_kwargs
  )

  module WorkerCLI
    module_function

    def run_cli(worker_factory, client_factory, argv)
      run(worker_factory, client_factory, argv)
    end

    def run(worker_factory, client_factory, argv)
      options = default_options
      build_parser(options).parse!(Array(argv).dup)

      if options[:task_queue_suffix_index_start] > options[:task_queue_suffix_index_end]
        raise ArgumentError,
              'Task queue suffix start after end'
      end

      logger = Helpers.configure_logger(options[:log_level], options[:log_encoding])
      config = ClientHelpers.build_client_config(
        server_address: options[:server_address],
        namespace: options[:namespace],
        auth_header: options[:auth_header],
        tls: options[:tls],
        tls_cert_path: options[:tls_cert_path],
        tls_key_path: options[:tls_key_path],
        prom_listen_address: options[:prom_listen_address]
      )
      client = client_factory.call(config)

      task_queues = build_task_queues(
        logger,
        options[:task_queue],
        options[:task_queue_suffix_index_start],
        options[:task_queue_suffix_index_end]
      )
      worker_kwargs = build_worker_kwargs(options)
      workers = task_queues.map do |task_queue|
        worker_factory.call(
          client,
          WorkerContext.new(
            logger: logger,
            task_queue: task_queue,
            err_on_unimplemented: options[:err_on_unimplemented],
            worker_kwargs: worker_kwargs
          )
        )
      end
      run_workers(workers)
    end

    def run_workers(workers)
      # The Ruby SDK already owns/supports coordinated multi-worker shutdown.
      Temporalio::Worker.run_all(*workers, shutdown_signals: ['SIGINT'])
    end

    def build_parser(options)
      OptionParser.new do |opts|
        opts.banner = 'Usage: runner.rb [options]'

        opts.on('-q', '--task-queue QUEUE', 'Task queue to use') { |value| options[:task_queue] = value }
        opts.on('--task-queue-suffix-index-start N', Integer) do |value|
          options[:task_queue_suffix_index_start] = value
        end
        opts.on('--task-queue-suffix-index-end N', Integer) { |value| options[:task_queue_suffix_index_end] = value }
        opts.on('--max-concurrent-activity-pollers N', Integer) do |value|
          options[:max_concurrent_activity_pollers] = value
        end
        opts.on('--max-concurrent-workflow-pollers N', Integer) do |value|
          options[:max_concurrent_workflow_pollers] = value
        end
        opts.on('--activity-poller-autoscale-max N', Integer) do |value|
          options[:activity_poller_autoscale_max] = value
        end
        opts.on('--workflow-poller-autoscale-max N', Integer) do |value|
          options[:workflow_poller_autoscale_max] = value
        end
        opts.on('--max-concurrent-activities N', Integer) { |value| options[:max_concurrent_activities] = value }
        opts.on('--max-concurrent-workflow-tasks N', Integer) do |value|
          options[:max_concurrent_workflow_tasks] = value
        end
        opts.on('--activities-per-second N', Float) { |value| options[:activities_per_second] = value }
        opts.on('--err-on-unimplemented BOOL') { |value| options[:err_on_unimplemented] = parse_bool(value) }
        opts.on('--log-level LEVEL') { |value| options[:log_level] = value }
        opts.on('--log-encoding ENCODING') { |value| options[:log_encoding] = value }
        opts.on('-n', '--namespace NAMESPACE') { |value| options[:namespace] = value }
        opts.on('-a', '--server-address ADDRESS') { |value| options[:server_address] = value }
        opts.on('--tls BOOL') { |value| options[:tls] = parse_bool(value) }
        opts.on('--tls-cert-path PATH') { |value| options[:tls_cert_path] = value }
        opts.on('--tls-key-path PATH') { |value| options[:tls_key_path] = value }
        opts.on('--prom-listen-address ADDRESS') { |value| options[:prom_listen_address] = value }
        opts.on('--prom-handler-path PATH') { |_value| nil }
        opts.on('--auth-header HEADER') { |value| options[:auth_header] = value }
        opts.on('--build-id ID') { |_value| nil }
      end
    end

    def build_task_queues(logger, task_queue, suffix_start, suffix_end)
      if suffix_end.zero?
        logger.info("Ruby worker will run on task queue #{task_queue}")
        return [task_queue]
      end

      task_queues = (suffix_start..suffix_end).map { |index| "#{task_queue}-#{index}" }
      logger.info("Ruby worker will run on #{task_queues.length} task queue(s)")
      task_queues
    end

    def build_worker_kwargs(options)
      worker_kwargs = {}
      if options[:activity_poller_autoscale_max]
        worker_kwargs[:activity_task_poller_behavior] = Temporalio::Worker::PollerBehavior::Autoscaling.new(
          maximum: options[:activity_poller_autoscale_max]
        )
      elsif options[:max_concurrent_activity_pollers]
        worker_kwargs[:max_concurrent_activity_task_polls] = options[:max_concurrent_activity_pollers]
      end

      if options[:workflow_poller_autoscale_max]
        worker_kwargs[:workflow_task_poller_behavior] = Temporalio::Worker::PollerBehavior::Autoscaling.new(
          maximum: options[:workflow_poller_autoscale_max]
        )
      elsif options[:max_concurrent_workflow_pollers]
        worker_kwargs[:max_concurrent_workflow_task_polls] = options[:max_concurrent_workflow_pollers]
      end

      if options[:max_concurrent_activities]
        worker_kwargs[:max_concurrent_activities] =
          options[:max_concurrent_activities]
      end
      if options[:max_concurrent_workflow_tasks]
        worker_kwargs[:max_concurrent_workflow_tasks] =
          options[:max_concurrent_workflow_tasks]
      end
      worker_kwargs[:max_activities_per_second] = options[:activities_per_second] if options[:activities_per_second]
      worker_kwargs
    end

    def parse_bool(value)
      return value if [true, false].include?(value)

      %w[true 1 yes].include?(value.to_s.downcase)
    end

    def default_options
      {
        task_queue: 'omes',
        task_queue_suffix_index_start: 0,
        task_queue_suffix_index_end: 0,
        max_concurrent_activity_pollers: nil,
        max_concurrent_workflow_pollers: nil,
        activity_poller_autoscale_max: nil,
        workflow_poller_autoscale_max: nil,
        max_concurrent_activities: nil,
        max_concurrent_workflow_tasks: nil,
        activities_per_second: nil,
        err_on_unimplemented: false,
        log_level: 'info',
        log_encoding: 'console',
        namespace: 'default',
        server_address: 'localhost:7233',
        tls: false,
        tls_cert_path: '',
        tls_key_path: '',
        prom_listen_address: nil,
        auth_header: ''
      }
    end
  end
end
