import root, { temporal } from './protos/root';
import {
  ProtobufBinaryPayloadConverter,
  ProtobufJsonPayloadConverter,
} from '@temporalio/common/lib/converter/protobuf-payload-converters';
import {
  BinaryPayloadConverter,
  CompositePayloadConverter,
  JsonPayloadConverter,
  PayloadConverterWithEncoding,
  UndefinedPayloadConverter,
} from '@temporalio/common';
import Payload = temporal.api.common.v1.Payload;

// TODO(thomas): can remove this file entirely (and usage of custom payload converter for worker)
// once RawValue.fromPayload(p) is released.
export class PassThroughPayload implements PayloadConverterWithEncoding {
  public toPayload(value: any): Payload | undefined {
    if (
      !value ||
      value.metadata == null ||
      value.data == null ||
      value?.metadata?.encoding == null
    ) {
      return undefined;
    }
    // If it looks like a Payload, return it as-is
    return value as Payload;
  }

  public fromPayload<T>(_: Payload): T {
    // This should never be called since we don't modify the encoding
    throw new Error('PassThroughPayload.fromPayload should not be called');
  }

  public get encodingType(): string {
    return '__passthrough';
  }
}

export const payloadConverter = new CompositePayloadConverter(
  new UndefinedPayloadConverter(),
  new BinaryPayloadConverter(),
  new PassThroughPayload(),
  new ProtobufJsonPayloadConverter(root),
  new ProtobufBinaryPayloadConverter(root),
  new JsonPayloadConverter(),
);
