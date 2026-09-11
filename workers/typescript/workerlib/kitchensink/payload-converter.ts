import root from './protos/root';
import {
  ProtobufBinaryPayloadConverter,
  ProtobufJsonPayloadConverter,
} from '@temporalio/common/lib/converter/protobuf-payload-converters';
import {
  BinaryPayloadConverter,
  CompositePayloadConverter,
  JsonPayloadConverter,
  UndefinedPayloadConverter,
} from '@temporalio/common';

export const payloadConverter = new CompositePayloadConverter(
  new UndefinedPayloadConverter(),
  new BinaryPayloadConverter(),
  new ProtobufJsonPayloadConverter(root),
  new ProtobufBinaryPayloadConverter(root),
  new JsonPayloadConverter(),
);
