package io.temporal.omes;

import com.google.protobuf.ByteString;
import io.temporal.api.common.v1.Payload;
import io.temporal.api.common.v1.PayloadOrBuilder;
import io.temporal.common.converter.DataConverterException;
import io.temporal.common.converter.EncodingKeys;
import io.temporal.common.converter.GlobalDataConverter;
import io.temporal.common.converter.PayloadConverter;
import io.temporal.workflow.Workflow;
import java.lang.reflect.Type;
import java.nio.charset.StandardCharsets;
import java.util.Optional;
import org.slf4j.Logger;

public final class PassthroughDataConverter implements PayloadConverter {
  public static final Logger log = Workflow.getLogger(PassthroughDataConverter.class);

  static final String METADATA_ENCODING_NAME = "_passthrough";
  static final ByteString METADATA_ENCODING =
      ByteString.copyFrom(METADATA_ENCODING_NAME, StandardCharsets.UTF_8);

  @Override
  public String getEncodingType() {
    return METADATA_ENCODING_NAME;
  }

  @Override
  public Optional<Payload> toData(Object value) throws DataConverterException {
    if (!(value instanceof PayloadOrBuilder)) {
      return Optional.empty();
    }
    if (!((PayloadOrBuilder) value)
        .getMetadataMap()
        .containsKey(EncodingKeys.METADATA_ENCODING_KEY)) {
      return Optional.empty();
    }
    try {
      Payload.Builder builder =
          Payload.newBuilder()
              .putMetadata(EncodingKeys.METADATA_ENCODING_KEY, METADATA_ENCODING)
              .setData(((Payload) value).toByteString());
      return Optional.of(builder.build());
    } catch (Exception e) {
      throw new DataConverterException(e);
    }
  }

  @Override
  @SuppressWarnings("unchecked")
  public <T> T fromData(Payload content, Class<T> valueClass, Type valueType)
      throws DataConverterException {
    try {
      Payload interPayload = Payload.parseFrom(content.getData().asReadOnlyByteBuffer());
      return GlobalDataConverter.get().fromPayload(interPayload, valueClass, valueType);
    } catch (Exception e) {
      throw new DataConverterException(e);
    }
  }
}
