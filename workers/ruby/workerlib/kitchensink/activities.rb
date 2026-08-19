require 'temporalio/activity'
require 'temporalio/activity/definition'
require_relative 'client_action_executor'

class NoopActivity < Temporalio::Activity::Definition
  activity_name 'noop'

  def execute
    nil
  end
end

class DelayActivity < Temporalio::Activity::Definition
  activity_name 'delay'

  def execute(delay_for)
    sleep(delay_for.seconds + (delay_for.nanos / 1_000_000_000.0))
  end
end

class PayloadActivity < Temporalio::Activity::Definition
  activity_name 'payload'

  def execute(_input_data, bytes_to_return)
    Random.bytes(bytes_to_return)
  end
end

class RetryableErrorActivity < Temporalio::Activity::Definition
  activity_name 'retryable_error'

  def execute(config)
    info = Temporalio::Activity::Context.current.info
    return unless info.attempt <= config.fail_attempts

    raise Temporalio::Error::ApplicationError.new('retryable error', type: 'RetryableError')
  end
end

class TimeoutActivity < Temporalio::Activity::Definition
  activity_name 'timeout'

  def execute(config)
    info = Temporalio::Activity::Context.current.info
    duration = config.success_duration
    duration = config.failure_duration if info.attempt <= config.fail_attempts
    sleep(duration.seconds + (duration.nanos / 1_000_000_000.0))
  end
end

class HeartbeatActivity < Temporalio::Activity::Definition
  activity_name 'heartbeat'

  def execute(config)
    info = Temporalio::Activity::Context.current.info
    should_send_heartbeats = info.attempt > config.fail_attempts
    unless should_send_heartbeats
      DelayActivity.new.execute(config.failure_duration)
      return
    end

    duration = duration_seconds(config.success_duration)
    interval = duration_seconds(config.heartbeat_interval)
    if interval <= 0
      sleep(duration)
      Temporalio::Activity::Context.current.heartbeat
      return
    end

    run_with_heartbeats(duration, interval)
  end

  private

  def run_with_heartbeats(duration, interval)
    Temporalio::Activity::Context.current.heartbeat
    remaining = duration
    while remaining.positive?
      sleep_for = [interval, remaining].min
      sleep(sleep_for)
      remaining -= sleep_for
      Temporalio::Activity::Context.current.heartbeat if remaining.positive?
    end
  end

  def duration_seconds(duration)
    return 0 if duration.nil?

    duration.seconds + (duration.nanos / 1_000_000_000.0)
  end
end

class ClientActivity < Temporalio::Activity::Definition
  activity_name 'client'

  def initialize(client, err_on_unimplemented: false)
    super()
    @client = client
    @err_on_unimplemented = err_on_unimplemented
  end

  def execute(client_activity_proto)
    info = Temporalio::Activity::Context.current.info
    executor = ClientActionExecutor.new(
      @client,
      info.workflow_id,
      info.task_queue,
      err_on_unimplemented: @err_on_unimplemented
    )
    executor.execute_client_sequence(client_activity_proto.client_sequence)
  end
end
