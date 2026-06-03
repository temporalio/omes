package io.temporal.omes.apps.worker;

import io.temporal.client.WorkflowClient;
import io.temporal.common.converter.ByteArrayPayloadConverter;
import io.temporal.common.converter.DefaultDataConverter;
import io.temporal.common.converter.JacksonJsonPayloadConverter;
import io.temporal.common.converter.NullPayloadConverter;
import io.temporal.common.converter.PayloadConverter;
import io.temporal.common.converter.ProtobufJsonPayloadConverter;
import io.temporal.common.converter.ProtobufPayloadConverter;
import io.temporal.omes.harness.Harness;
import io.temporal.omes.harness.HarnessClients;
import io.temporal.omes.harness.WorkerHarness;
import io.temporal.omes.workerlib.kitchensink.ActivitiesImpl;
import io.temporal.omes.workerlib.kitchensink.KitchenSinkWorkflowImpl;
import io.temporal.omes.workerlib.kitchensink.PassthroughDataConverter;
import io.temporal.worker.Worker;

public final class WorkerApp {
  public static final Harness.App APP =
      new Harness.App(WorkerApp::configureWorker, WorkerApp::createClient);

  private WorkerApp() {}

  private static WorkflowClient createClient(HarnessClients.ClientConfig config) throws Exception {
    PayloadConverter[] converters = {
      new NullPayloadConverter(),
      new ByteArrayPayloadConverter(),
      new PassthroughDataConverter(),
      new ProtobufJsonPayloadConverter(),
      new ProtobufPayloadConverter(),
      new JacksonJsonPayloadConverter()
    };
    return HarnessClients.newWorkflowClient(config, new DefaultDataConverter(converters));
  }

  private static void configureWorker(
      WorkflowClient client, Worker worker, WorkerHarness.WorkerContext context) {
    worker.registerWorkflowImplementationTypes(KitchenSinkWorkflowImpl.class);
    worker.registerActivitiesImplementations(
        new ActivitiesImpl(client, context.errOnUnimplemented));
  }
}
