require 'logger'
require 'json'
require 'optparse'
require 'temporalio/client'
require 'temporalio/runtime'
require 'temporalio/worker'
require_relative 'kitchen_sink'
require_relative 'activities'

NAME_TO_LEVEL = {
  'PANIC' => Logger::FATAL,
  'FATAL' => Logger::FATAL,
  'ERROR' => Logger::ERROR,
  'WARN' => Logger::WARN,
  'INFO' => Logger::INFO,
  'DEBUG' => Logger::DEBUG,
  'NOTSET' => Logger::DEBUG
}.freeze

options = {
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

OptionParser.new do |opts|
  opts.banner = 'Usage: runner.rb [options]'

  opts.on('-q', '--task-queue QUEUE', 'Task queue to use') { |v| options[:task_queue] = v }
  opts.on('--task-queue-suffix-index-start N', Integer) { |v| options[:task_queue_suffix_index_start] = v }
  opts.on('--task-queue-suffix-index-end N', Integer) { |v| options[:task_queue_suffix_index_end] = v }
  opts.on('--max-concurrent-activity-pollers N', Integer) { |v| options[:max_concurrent_activity_pollers] = v }
  opts.on('--max-concurrent-workflow-pollers N', Integer) { |v| options[:max_concurrent_workflow_pollers] = v }
  opts.on('--activity-poller-autoscale-max N', Integer) { |v| options[:activity_poller_autoscale_max] = v }
  opts.on('--workflow-poller-autoscale-max N', Integer) { |v| options[:workflow_poller_autoscale_max] = v }
  opts.on('--max-concurrent-activities N', Integer) { |v| options[:max_concurrent_activities] = v }
  opts.on('--max-concurrent-workflow-tasks N', Integer) { |v| options[:max_concurrent_workflow_tasks] = v }
  opts.on('--activities-per-second N', Float) { |v| options[:activities_per_second] = v }
  opts.on('--err-on-unimplemented BOOL') { |v| options[:err_on_unimplemented] = %w[true 1 yes].include?(v.downcase) }
  opts.on('--log-level LEVEL') { |v| options[:log_level] = v }
  opts.on('--log-encoding ENC') { |v| options[:log_encoding] = v }
  opts.on('-n', '--namespace NS') { |v| options[:namespace] = v }
  opts.on('-a', '--server-address ADDR') { |v| options[:server_address] = v }
  opts.on('--tls BOOL') { |v| options[:tls] = %w[true 1 yes].include?(v.downcase) }
  opts.on('--tls-cert-path PATH') { |v| options[:tls_cert_path] = v }
  opts.on('--tls-key-path PATH') { |v| options[:tls_key_path] = v }
  opts.on('--prom-listen-address ADDR') { |v| options[:prom_listen_address] = v }
  opts.on('--prom-handler-path PATH') { |_v| nil }
  opts.on('--auth-header HEADER') { |v| options[:auth_header] = v }
  opts.on('--build-id ID') { |_v| nil }
end.parse!

if options[:task_queue_suffix_index_start] > options[:task_queue_suffix_index_end]
  abort 'Task queue suffix start after end'
end

tls_options = nil
if !options[:tls_cert_path].empty?
  abort 'Client cert specified, but not client key!' if options[:tls_key_path].empty?
  tls_options = Temporalio::Client::Connection::TLSOptions.new(
    client_cert: File.binread(options[:tls_cert_path]),
    client_private_key: File.binread(options[:tls_key_path])
  )
elsif !options[:tls_key_path].empty?
  abort 'Client key specified, but not client cert!'
elsif options[:tls]
  tls_options = Temporalio::Client::Connection::TLSOptions.new
end

api_key = nil
api_key = options[:auth_header].delete_prefix('Bearer ') unless options[:auth_header].empty?

logger = Logger.new($stderr)
logger.level = NAME_TO_LEVEL.fetch(options[:log_level].upcase, Logger::INFO)
if options[:log_encoding] == 'json'
  logger.formatter = proc do |severity, datetime, _progname, msg|
    "#{JSON.generate(message: msg, level: severity, timestamp: datetime.iso8601)}\n"
  end
end

prometheus = nil
if options[:prom_listen_address]
  prometheus = Temporalio::Runtime::PrometheusMetricsOptions.new(
    bind_address: options[:prom_listen_address]
  )
end

runtime = Temporalio::Runtime.new(
  telemetry: Temporalio::Runtime::TelemetryOptions.new(
    metrics: prometheus,
    logging: Temporalio::Runtime::LoggingOptions.new(
      log_filter: Temporalio::Runtime::LoggingFilterOptions.new(
        core_level: ENV.fetch('TEMPORAL_CORE_LOG_LEVEL', 'INFO'),
        other_level: 'WARN'
      )
    )
  )
)

client = Temporalio::Client.connect(
  options[:server_address],
  options[:namespace],
  tls: tls_options,
  api_key: api_key,
  runtime: runtime,
  logger: logger
)

task_queues = if options[:task_queue_suffix_index_end].zero?
                logger.info("Ruby worker running for task queue #{options[:task_queue]}")
                [options[:task_queue]]
              else
                tqs = (options[:task_queue_suffix_index_start]..options[:task_queue_suffix_index_end]).map do |i|
                  "#{options[:task_queue]}-#{i}"
                end
                logger.info("Ruby worker running for #{tqs.length} task queue(s)")
                tqs
              end

worker_opts = {}

if options[:activity_poller_autoscale_max]
  worker_opts[:activity_task_poller_behavior] = Temporalio::Worker::PollerBehavior::Autoscaling.new(
    maximum: options[:activity_poller_autoscale_max]
  )
elsif options[:max_concurrent_activity_pollers]
  worker_opts[:max_concurrent_activity_task_polls] = options[:max_concurrent_activity_pollers]
end

if options[:workflow_poller_autoscale_max]
  worker_opts[:workflow_task_poller_behavior] = Temporalio::Worker::PollerBehavior::Autoscaling.new(
    maximum: options[:workflow_poller_autoscale_max]
  )
elsif options[:max_concurrent_workflow_pollers]
  worker_opts[:max_concurrent_workflow_task_polls] = options[:max_concurrent_workflow_pollers]
end

worker_opts[:max_concurrent_activities] = options[:max_concurrent_activities] if options[:max_concurrent_activities]
if options[:max_concurrent_workflow_tasks]
  worker_opts[:max_concurrent_workflow_tasks] =
    options[:max_concurrent_workflow_tasks]
end
worker_opts[:max_activities_per_second] = options[:activities_per_second] if options[:activities_per_second]

client_activity = ClientActivity.new(client, err_on_unimplemented: options[:err_on_unimplemented])

activities = [
  NoopActivity.new,
  DelayActivity.new,
  PayloadActivity.new,
  RetryableErrorActivity.new,
  TimeoutActivity.new,
  HeartbeatActivity.new,
  client_activity
]

workers = task_queues.map do |tq|
  Temporalio::Worker.new(
    client: client,
    task_queue: tq,
    workflows: [KitchenSinkWorkflow],
    activities: activities,
    logger: logger,
    **worker_opts
  )
end

if workers.length == 1
  workers[0].run(shutdown_signals: ['SIGINT'])
else
  threads = workers.map do |w|
    Thread.new { w.run(shutdown_signals: ['SIGINT']) }
  end
  threads.each(&:join)
end
