import root from './protos/root';
import {
  DefaultPayloadConverterWithProtobufs
} from '@temporalio/common/lib/converter/protobuf-payload-converters';

export const payloadConverter = new DefaultPayloadConverterWithProtobufs({ protobufRoot: root });
