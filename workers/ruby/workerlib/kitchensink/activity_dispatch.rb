# frozen_string_literal: true

require 'temporalio/retry_policy'

# ActivityDispatch maps an ExecuteActivityAction to the registered activity name
# and its args, and converts its retry policy and timeouts. Shared by the
# workflow-scheduled path (KitchenSinkWorkflow#launch_activity) and the
# standalone-activity client path so the two stay in sync.
module ActivityDispatch
  module_function

  # Returns [activity_name, args] for the given ExecuteActivityAction.
  # Unrecognized variants fall back to the noop activity.
  def name_and_args(exec_activity)
    case exec_activity.activity_type
    when :delay
      ['delay', [exec_activity.delay]]
    when :payload
      input_data = Array.new(exec_activity.payload.bytes_to_receive) { |i| i % 256 }.pack('C*')
      ['payload', [input_data, exec_activity.payload.bytes_to_return]]
    when :client
      ['client', [exec_activity.client]]
    when :retryable_error
      ['retryable_error', [exec_activity.retryable_error]]
    when :timeout
      ['timeout', [exec_activity.timeout]]
    when :heartbeat
      ['heartbeat', [exec_activity.heartbeat]]
    else
      ['noop', []]
    end
  end

  # Converts a proto RetryPolicy into a Temporalio::RetryPolicy, or nil if unset.
  def retry_policy_from_proto(retry_policy)
    return nil if retry_policy.nil?

    Temporalio::RetryPolicy.new(
      max_attempts: retry_policy.maximum_attempts,
      initial_interval: retry_policy.initial_interval&.then do |d|
        d.seconds + (d.nanos / 1_000_000_000.0)
      end
    )
  end

  # Converts the named protobuf Duration field to seconds, or nil if unset/zero.
  def duration_or_nil(activity_action, field)
    val = activity_action.send(field)
    return nil if val.nil? || (val.seconds.zero? && val.nanos.zero?)

    val.seconds + (val.nanos / 1_000_000_000.0)
  end
end
