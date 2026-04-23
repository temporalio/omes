# frozen_string_literal: true

require 'bundler/setup'
require 'logger'
require 'minitest/autorun'
require 'harness'

class HarnessWorkerTest < Minitest::Test
  def test_run_passes_shared_client_and_context_to_each_worker_factory
    client = Object.new
    worker_factory_calls = []
    created_workers = [Object.new, Object.new]
    captured_workers = nil

    worker_factory = lambda do |given_client, context|
      worker_factory_calls << [given_client, context]
      created_workers.fetch(worker_factory_calls.length - 1)
    end
    client_factory = ->(_config) { client }

    with_stubbed_run_all(lambda do |*workers, **_kwargs|
      captured_workers = workers
    end) do
      Harness::WorkerCLI.run_cli(
        worker_factory,
        client_factory,
        [
          '--task-queue', 'omes',
          '--task-queue-suffix-index-start', '1',
          '--task-queue-suffix-index-end', '2'
        ]
      )
    end

    assert_equal created_workers, captured_workers
    assert_equal 2, worker_factory_calls.length

    assert_same client, worker_factory_calls[0][0]
    assert_same client, worker_factory_calls[1][0]
    assert_same worker_factory_calls[0][1].worker_kwargs, worker_factory_calls[1][1].worker_kwargs
    assert_equal 'omes-1', worker_factory_calls[0][1].task_queue
    assert_equal 'omes-2', worker_factory_calls[1][1].task_queue
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
