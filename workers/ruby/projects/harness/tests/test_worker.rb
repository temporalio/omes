# frozen_string_literal: true

require 'logger'
require 'minitest/autorun'
require_relative '../../harness'

class HarnessWorkerTest < Minitest::Test
  FakeRunnableWorker = Struct.new(
    :on_run,
    :run_calls,
    :shutdown_calls,
    :received_shutdown_signals
  ) do
    def run(shutdown_signals: [])
      self.run_calls += 1
      self.received_shutdown_signals = shutdown_signals
      on_run.call
    end

    def shutdown
      self.shutdown_calls += 1
    end
  end

  def test_run_passes_shared_client_and_context_to_each_worker_factory
    client = Object.new
    built_config = nil
    worker_factory_calls = []
    created_workers = [Object.new, Object.new]
    captured_workers = nil
    captured_kwargs = nil

    worker_factory = lambda do |given_client, context|
      worker_factory_calls << [given_client, context]
      created_workers.fetch(worker_factory_calls.length - 1)
    end
    client_factory = lambda do |config|
      built_config = config
      client
    end

    with_stubbed_run_all(lambda do |*workers, **kwargs|
      captured_workers = workers
      captured_kwargs = kwargs
    end) do
      Harness::WorkerCLI.run(
        worker_factory,
        client_factory,
        [
          '--task-queue', 'omes',
          '--task-queue-suffix-index-start', '1',
          '--task-queue-suffix-index-end', '2'
        ]
      )
    end

    assert_equal 'localhost:7233', built_config.target_host
    assert_equal 'default', built_config.namespace
    assert_nil built_config.api_key
    assert_nil built_config.tls
    assert_instance_of Temporalio::Runtime, built_config.runtime
    assert_equal created_workers, captured_workers
    assert_equal({ shutdown_signals: ['SIGINT'] }, captured_kwargs)
    assert_equal 2, worker_factory_calls.length
    assert_same client, worker_factory_calls[0][0]
    assert_same client, worker_factory_calls[1][0]
    assert_equal 'omes-1', worker_factory_calls[0][1].task_queue
    assert_equal 'omes-2', worker_factory_calls[1][1].task_queue
    assert_instance_of Logger, worker_factory_calls[0][1].logger
  end

  def test_run_workers_shuts_down_all_workers_when_one_fails
    boom = RuntimeError.new('boom')
    failing_worker = FakeRunnableWorker.new(
      on_run: -> { raise boom },
      run_calls: 0,
      shutdown_calls: 0,
      received_shutdown_signals: nil
    )
    successful_worker = FakeRunnableWorker.new(
      on_run: -> {},
      run_calls: 0,
      shutdown_calls: 0,
      received_shutdown_signals: nil
    )

    error = with_stubbed_run_all(lambda do |*workers, **kwargs|
      first_error = nil
      workers.each do |worker|
        worker.run(shutdown_signals: kwargs[:shutdown_signals])
      rescue StandardError => e
        first_error ||= e
      end
      workers.each(&:shutdown)
      raise first_error if first_error
    end) do
      assert_raises(RuntimeError) do
        Harness::WorkerCLI.run_workers([failing_worker, successful_worker])
      end
    end

    assert_same boom, error
    assert_equal ['SIGINT'], failing_worker.received_shutdown_signals
    assert_equal ['SIGINT'], successful_worker.received_shutdown_signals
    assert_equal 1, failing_worker.run_calls
    assert_equal 1, successful_worker.run_calls
    assert_equal 1, failing_worker.shutdown_calls
    assert_equal 1, successful_worker.shutdown_calls
  end

  private

  def with_stubbed_run_all(stub_implementation)
    singleton = Temporalio::Worker.singleton_class
    singleton.send(:alias_method, :__original_run_all_for_test, :run_all)
    singleton.send(:define_method, :run_all, &stub_implementation)
    yield
  ensure
    if singleton.method_defined?(:__original_run_all_for_test)
      singleton.send(:remove_method, :run_all)
      singleton.send(:alias_method, :run_all, :__original_run_all_for_test)
      singleton.send(:remove_method, :__original_run_all_for_test)
    end
  end
end
