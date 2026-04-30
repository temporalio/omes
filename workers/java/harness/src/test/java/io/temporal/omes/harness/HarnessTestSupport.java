package io.temporal.omes.harness;

import com.uber.m3.tally.NoopScope;
import com.uber.m3.tally.Scope;
import io.temporal.api.workflowservice.v1.GetSystemInfoResponse;
import io.temporal.client.WorkflowClient;
import io.temporal.client.WorkflowClientOptions;
import io.temporal.serviceclient.WorkflowServiceStubs;
import io.temporal.serviceclient.WorkflowServiceStubsOptions;
import java.lang.reflect.InvocationHandler;
import java.lang.reflect.Method;
import java.lang.reflect.Proxy;

final class HarnessTestSupport {
  private HarnessTestSupport() {}

  static WorkflowClient fakeWorkflowClient() {
    Scope metricsScope = new NoopScope();
    WorkflowServiceStubsOptions serviceStubsOptions =
        WorkflowServiceStubsOptions.newBuilder().setMetricsScope(metricsScope).build();
    WorkflowServiceStubs serviceStubs =
        (WorkflowServiceStubs)
            Proxy.newProxyInstance(
                WorkflowServiceStubs.class.getClassLoader(),
                new Class<?>[] {WorkflowServiceStubs.class},
                new StaticResponseHandler()
                    .with("getOptions", serviceStubsOptions)
                    .with(
                        "getServerCapabilities",
                        (java.util.function.Supplier<GetSystemInfoResponse.Capabilities>)
                            GetSystemInfoResponse.Capabilities::getDefaultInstance)
                    .with("toString", "fakeWorkflowServiceStubs"));

    return WorkflowClient.newInstance(
        serviceStubs, WorkflowClientOptions.newBuilder().setNamespace("default").build());
  }

  private static final class StaticResponseHandler implements InvocationHandler {
    private final java.util.Map<String, Object> responses = new java.util.HashMap<>();

    private StaticResponseHandler with(String methodName, Object response) {
      responses.put(methodName, response);
      return this;
    }

    @Override
    public Object invoke(Object proxy, Method method, Object[] args) {
      if (responses.containsKey(method.getName())) {
        return responses.get(method.getName());
      }
      if (method.getDeclaringClass() == Object.class) {
        switch (method.getName()) {
          case "hashCode":
            return System.identityHashCode(proxy);
          case "equals":
            return proxy == args[0];
          default:
            break;
        }
      }
      throw new AssertionError("Unexpected fake service call: " + method.getName());
    }
  }
}
