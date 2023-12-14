import root, { temporal } from './protos/root';
import {
  ProtobufBinaryPayloadConverter,
  ProtobufJsonPayloadConverter
} from '@temporalio/common/lib/converter/protobuf-payload-converters';
import {
  BinaryPayloadConverter,
  CompositePayloadConverter,
  JsonPayloadConverter,
  PayloadConverterWithEncoding,
  UndefinedPayloadConverter,
  ValueError
} from '@temporalio/common';
import Payload = temporal.api.common.v1.Payload;
import { encode, decode } from '@temporalio/common/lib/encoding';

export class PassThroughPayload implements PayloadConverterWithEncoding {

  public toPayload(value: unknown): Payload | undefined {
    if (!value || !value.hasOwnProperty("metadata") || !value.hasOwnProperty("data")) {
      return undefined;
    }
    let asPayload;
    try {
      asPayload = Payload.fromObject(value as any);
    } catch (e) {
      throw new ValueError('PassThroughPayload can only convert Payloads');
    }
    const asBytes = Payload.encode(asPayload).finish();
    return Payload.create({
      metadata: {
        encoding: encode(this.encodingType)
      }, data: asBytes
    });
  }

  public fromPayload<T>(content: Payload): T {
    if (decode(content.metadata?.encoding) === '__passthrough') {
      const innerPayload = Payload.decode(new Uint8Array(content.data));
      return payloadConverter.fromPayload<T>(innerPayload);
    }
    throw new ValueError('PassThroughPayload can only decode passthrough Payloads, got ' + JSON.stringify(content));
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
  new JsonPayloadConverter()
);

