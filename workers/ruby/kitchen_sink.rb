require 'temporalio/workflow'
require 'temporalio/workflow/definition'
require_relative 'protos/kitchen_sink/kitchen_sink_pb'

KS = Temporal::Omes::KitchenSink

class KitchenSinkWorkflow < Temporalio::Workflow::Definition
  workflow_name 'kitchenSink'
  workflow_arg_hint KS::WorkflowInput
  workflow_result_hint Temporalio::Api::Common::V1::Payload

  def initialize
    super
    @action_set_queue = []
    @workflow_state = KS::WorkflowState.new
  end

  def execute(input = nil)
    Temporalio::Workflow.logger.debug('Started kitchen sink workflow')

    initial_return_value = nil
    if input&.initial_actions&.any?
      input.initial_actions.each do |action_set|
        return_value = handle_action_set(action_set)
        unless return_value.nil?
          initial_return_value = return_value
          break
        end
      end
    end

    if input&.expected_signal_count&.positive?
      raise Temporalio::Error::ApplicationError.new(
        'signal deduplication not implemented',
        non_retryable: true
      )
    end

    return initial_return_value unless initial_return_value.nil?

    loop do
      Temporalio::Workflow.wait_condition { !@action_set_queue.empty? }
      action_set = @action_set_queue.shift
      return_value = handle_action_set(action_set)
      return return_value unless return_value.nil?
    end
  end

  workflow_signal arg_hints: [KS::DoSignal::DoSignalActions]
  def do_actions_signal(signal_actions)
    if signal_actions.variant == :do_actions_in_main
      @action_set_queue << signal_actions.do_actions_in_main
    else
      handle_action_set(signal_actions.do_actions)
    end
  end

  workflow_update arg_hints: [KS::DoActionsUpdate]
  def do_actions_update(actions_update)
    retval = handle_action_set(actions_update.do_actions)
    return retval unless retval.nil?

    @workflow_state
  end

  workflow_update_validator :do_actions_update
  def do_actions_update_validator(actions_update)
    if actions_update.respond_to?(:reject_me) && actions_update.reject_me
      raise Temporalio::Error::ApplicationError, 'Rejected'
    end
  end

  workflow_query result_hint: KS::WorkflowState
  def report_state(_input = nil)
    @workflow_state
  end

  private

  def handle_action_set(action_set)
    return nil if action_set.nil?

    if action_set.concurrent
      return_value = nil
      futures = action_set.actions.map do |action|
        Temporalio::Workflow.async do
          result = handle_action(action)
          return_value = result unless result.nil?
        end
      end
      Temporalio::Workflow.wait_condition { futures.all?(&:done?) || !return_value.nil? }
      return_value
    else
      action_set.actions.each do |action|
        return_value = handle_action(action)
        return return_value unless return_value.nil?
      end
      nil
    end
  end

  def handle_action(action)
    case action.variant
    when :return_result
      return action.return_result.return_this
    when :return_error
      raise Temporalio::Error::ApplicationError, action.return_error.failure.message
    when :continue_as_new
      args = action.continue_as_new.arguments.map { |a| Temporalio::Converters::RawValue.new(a) }
      Temporalio::Workflow.continue_as_new(*args)
    when :timer
      handle_awaitable_choice(
        -> { Temporalio::Workflow.sleep(action.timer.milliseconds / 1000.0) },
        action.timer.awaitable_choice
      )
    when :exec_activity
      handle_awaitable_choice(
        -> { launch_activity(action.exec_activity) },
        action.exec_activity.awaitable_choice
      )
    when :exec_child_workflow
      child_action = action.exec_child_workflow
      child_type = child_action.workflow_type.empty? ? 'kitchenSink' : child_action.workflow_type
      args = child_action.input.map { |a| Temporalio::Converters::RawValue.new(a) }

      proto_sa = Temporalio::Api::Common::V1::SearchAttributes.new(
        indexed_fields: child_action.search_attributes.to_h
      )

      handle_awaitable_choice(
        lambda {
          Temporalio::Workflow.execute_child_workflow(
            child_type,
            *args,
            id: child_action.workflow_id,
            task_queue: child_action.task_queue.empty? ? nil : child_action.task_queue,
            search_attributes: Temporalio::SearchAttributes._from_proto(proto_sa)
          )
        },
        child_action.awaitable_choice,
        after_started_fn: lambda(&:result),
        after_completed_fn: lambda(&:result)
      )
    when :set_patch_marker
      marker = action.set_patch_marker
      if marker.deprecated
        Temporalio::Workflow.deprecate_patch(marker.patch_id)
        was_patched = true
      else
        was_patched = Temporalio::Workflow.patched(marker.patch_id)
      end
      return handle_action(marker.inner_action) if was_patched
    when :set_workflow_state
      @workflow_state = action.set_workflow_state
    when :await_workflow_state
      target_key = action.await_workflow_state.key
      target_value = action.await_workflow_state.value
      Temporalio::Workflow.wait_condition do
        @workflow_state.kvs.to_h.fetch(target_key, nil) == target_value
      end
    when :upsert_memo
      nil
    when :upsert_search_attributes
      updates = action.upsert_search_attributes.search_attributes.map do |k, v|
        if k.include?('Keyword')
          Temporalio::SearchAttributes::Update.new(
            Temporalio::SearchAttributes::Key.new(k, :keyword),
            v.data[0].to_s
          )
        else
          Temporalio::SearchAttributes::Update.new(
            Temporalio::SearchAttributes::Key.new(k, :int),
            v.data[0].unpack1('q<')
          )
        end
      end
      Temporalio::Workflow.upsert_search_attributes(*updates)
    when :nested_action_set
      return handle_action_set(action.nested_action_set)
    when :nexus_operation
      raise Temporalio::Error::ApplicationError.new(
        'ExecuteNexusOperation is not supported',
        non_retryable: true
      )
    else
      raise Temporalio::Error::ApplicationError, "unrecognized action: #{action}"
    end

    nil
  end

  def launch_activity(exec_activity)
    act_type = 'noop'
    args = []

    case exec_activity.activity_type
    when :delay
      act_type = 'delay'
      args << exec_activity.delay
    when :noop
      act_type = 'noop'
    when :payload
      act_type = 'payload'
      input_data = Array.new(exec_activity.payload.bytes_to_receive) { |i| i % 256 }.pack('C*')
      args << input_data
      args << exec_activity.payload.bytes_to_return
    when :client
      act_type = 'client'
      args << exec_activity.client
    when :retryable_error
      act_type = 'retryable_error'
      args << exec_activity.retryable_error
    when :timeout
      act_type = 'timeout'
      args << exec_activity.timeout
    when :heartbeat
      act_type = 'heartbeat'
      args << exec_activity.heartbeat
    end

    options = {
      schedule_to_close_timeout: duration_or_nil(exec_activity, :schedule_to_close_timeout),
      start_to_close_timeout: duration_or_nil(exec_activity, :start_to_close_timeout),
      schedule_to_start_timeout: duration_or_nil(exec_activity, :schedule_to_start_timeout)
    }.compact

    if exec_activity.retry_policy
      options[:retry_policy] = Temporalio::RetryPolicy.new(
        max_attempts: exec_activity.retry_policy.maximum_attempts,
        initial_interval: exec_activity.retry_policy.initial_interval&.then do |d|
          d.seconds + (d.nanos / 1_000_000_000.0)
        end
      )
    end

    if exec_activity.locality == :is_local
      Temporalio::Workflow.execute_local_activity(act_type, *args, **options)
    else
      if exec_activity.respond_to?(:has_priority?) && exec_activity.has_priority?
        raise NotImplementedError, 'priority is not supported yet'
      end

      ht = duration_or_nil(exec_activity, :heartbeat_timeout)
      options[:heartbeat_timeout] = ht if ht
      options[:task_queue] = exec_activity.task_queue unless exec_activity.task_queue.empty?
      if exec_activity.locality == :remote
        options[:cancellation_type] = convert_cancel_type(exec_activity.remote.cancellation_type)
      end

      Temporalio::Workflow.execute_activity(act_type, *args, **options)
    end
  end

  def handle_awaitable_choice(
    start_fn,
    choice,
    after_started_fn: method(:brief_wait),
    after_completed_fn: method(:wait_task_complete)
  )
    if choice.nil? || choice.condition.nil?
      start_fn.call
      return
    end

    case choice.condition
    when :abandon
      nil
    when :cancel_before_started
      task = Temporalio::Workflow.async { start_fn.call }
      task.cancel
      begin
        task.result
      rescue Temporalio::Error::CancelledError
        nil
      end
    when :cancel_after_started
      task = Temporalio::Workflow.async { start_fn.call }
      after_started_fn.call(task)
      task.cancel
      begin
        task.result
      rescue Temporalio::Error::CancelledError
        nil
      end
    when :cancel_after_completed
      task = Temporalio::Workflow.async { start_fn.call }
      after_completed_fn.call(task)
      task.cancel
    when :wait_finish
      start_fn.call
    end
  end

  def brief_wait(_task)
    Temporalio::Workflow.sleep(0.001)
  end

  def wait_task_complete(task)
    task.result
  end

  def convert_cancel_type(ctype)
    case ctype
    when :TRY_CANCEL then Temporalio::Workflow::ActivityCancellationType::TRY_CANCEL
    when :WAIT_CANCELLATION_COMPLETED then Temporalio::Workflow::ActivityCancellationType::WAIT_CANCELLATION_COMPLETED
    when :ABANDON then Temporalio::Workflow::ActivityCancellationType::ABANDON
    else raise NotImplementedError, "Unknown cancellation type #{ctype}"
    end
  end

  def duration_or_nil(activity_action, field)
    val = activity_action.send(field)
    return nil if val.nil? || (val.seconds.zero? && val.nanos.zero?)

    val.seconds + (val.nanos / 1_000_000_000.0)
  end
end
