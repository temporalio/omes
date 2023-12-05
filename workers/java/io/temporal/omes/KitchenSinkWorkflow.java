package io.temporal.omes;

import io.temporal.workflow.*;

@WorkflowInterface
public interface KitchenSinkWorkflow {

  @WorkflowMethod(name = "kitchenSink")
  Object execute(KitchenSink.WorkflowInput input);

  @SignalMethod(name = "do_actions_signal")
  void doActionsSignal(KitchenSink.DoSignal.DoSignalActions signalInput);

  @UpdateMethod(name = "do_actions_update")
  Object doActionsUpdate(KitchenSink.DoActionsUpdate updateInput);

  @UpdateValidatorMethod(updateName = "do_actions_update")
  void doActionsUpdateValidator(KitchenSink.DoActionsUpdate updateInput);

  @QueryMethod(name = "report_state")
  KitchenSink.WorkflowState reportState(Object queryInput);
}
