# frozen_string_literal: true

class ClientActionExecutor
  def initialize(client, workflow_id, task_queue, err_on_unimplemented: false)
    @client = client
    @workflow_id = workflow_id
    @workflow_type = 'kitchenSink'
    @task_queue = task_queue
    @err_on_unimplemented = err_on_unimplemented
  end

  def execute_client_sequence(client_seq)
    client_seq.action_sets.each do |action_set|
      execute_client_action_set(action_set)
    end
  end

  private

  def execute_client_action_set(action_set)
    if action_set.concurrent
      if @err_on_unimplemented
        raise Temporalio::Error::ApplicationError.new(
          'concurrent client actions are not supported',
          non_retryable: true
        )
      end
      return
    end

    action_set.actions.each do |action|
      execute_client_action(action)
    end
  end

  def execute_client_action(action)
    case action.variant
    when :do_signal
      execute_signal_action(action.do_signal)
    when :do_update
      execute_update_action(action.do_update)
    when :do_query
      execute_query_action(action.do_query)
    when :nested_actions
      execute_client_action_set(action.nested_actions)
    else
      raise 'Client action must have a recognized variant'
    end
  end

  def execute_signal_action(signal)
    case signal.variant
    when :do_signal_actions
      signal_name = 'do_actions_signal'
      signal_args = [signal.do_signal_actions]
    when :custom
      signal_name = signal.custom.name
      signal_args = signal.custom.args.to_a
    else
      raise 'DoSignal must have a recognizable variant'
    end

    if signal.with_start
      @client.start_workflow(
        @workflow_type,
        nil,
        id: @workflow_id,
        task_queue: @task_queue,
        id_conflict_policy: :use_existing,
        start_signal: signal_name,
        start_signal_args: signal_args
      )
    else
      handle = @client.workflow_handle(@workflow_id)
      handle.signal(signal_name, *signal_args)
    end
  end

  def execute_update_action(update)
    case update.variant
    when :do_actions
      update_name = 'do_actions_update'
      update_args = [update.do_actions]
    when :custom
      update_name = update.custom.name
      update_args = update.custom.args.to_a
    else
      raise 'DoUpdate must have a recognizable variant'
    end

    begin
      if update.with_start
        start_op = Temporalio::Client::WithStartWorkflowOperation.new(
          workflow: @workflow_type,
          args: [nil],
          id: @workflow_id,
          task_queue: @task_queue,
          id_conflict_policy: :use_existing
        )
        @client.execute_update_with_start_workflow(
          update_name,
          *update_args,
          start_workflow_operation: start_op
        )
      else
        handle = @client.workflow_handle(@workflow_id)
        handle.execute_update(update_name, *update_args)
      end
    rescue StandardError
      raise unless update.failure_expected
    end
  end

  def execute_query_action(query)
    case query.variant
    when :report_state
      handle = @client.workflow_handle(@workflow_id)
      handle.query('report_state', nil)
    when :custom
      handle = @client.workflow_handle(@workflow_id)
      query_args = query.custom.args.to_a
      handle.query(query.custom.name, *query_args)
    else
      raise 'DoQuery must have a recognizable variant'
    end
  rescue StandardError
    raise unless query.failure_expected
  end
end
