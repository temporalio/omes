package io.temporal.omes;

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
import io.temporal.worker.Worker;

public final class Main {
  private Main() {}

  public static void main(String... args) throws Exception {
    Harness.run(app(), args);
  }

  private static Harness.App app() {
    return new Harness.App(Main::configureWorker, Main::createClient);
  }

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
    worker.registerActivitiesImplementations(new ActivitiesImpl(client, context.errOnUnimplemented));
  }
}
