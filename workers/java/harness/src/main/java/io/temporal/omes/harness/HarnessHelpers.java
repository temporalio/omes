package io.temporal.omes.harness;

import ch.qos.logback.classic.Level;
import ch.qos.logback.classic.LoggerContext;
import ch.qos.logback.classic.encoder.PatternLayoutEncoder;
import ch.qos.logback.classic.spi.ILoggingEvent;
import ch.qos.logback.core.ConsoleAppender;
import java.util.Locale;
import net.logstash.logback.encoder.LogstashEncoder;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

final class HarnessHelpers {
  private HarnessHelpers() {}

  static Logger configure(String logLevel, String logEncoding) {
    Logger logger = LoggerFactory.getLogger(org.slf4j.Logger.ROOT_LOGGER_NAME);
    if (!(logger instanceof ch.qos.logback.classic.Logger)) {
      return logger;
    }

    ch.qos.logback.classic.Logger rootLogger = (ch.qos.logback.classic.Logger) logger;
    rootLogger.setLevel(resolveLogLevel(logLevel));

    if ("json".equals(logEncoding)) {
      rootLogger.detachAndStopAllAppenders();
      rootLogger.addAppender(jsonAppender(rootLogger.getLoggerContext()));
    } else if (!rootLogger.iteratorForAppenders().hasNext()) {
      rootLogger.addAppender(consoleAppender(rootLogger.getLoggerContext()));
    }

    return rootLogger;
  }

  private static Level resolveLogLevel(String logLevel) {
    switch (logLevel.toUpperCase(Locale.ROOT)) {
      case "PANIC":
      case "FATAL":
      case "ERROR":
        return Level.ERROR;
      case "WARN":
        return Level.WARN;
      case "INFO":
        return Level.INFO;
      case "DEBUG":
        return Level.DEBUG;
      case "NOTSET":
        return Level.ALL;
      default:
        throw new IllegalArgumentException(
            "Invalid log level: "
                + logLevel
                + ". Expected one of: debug, info, warn, error, panic, fatal, notset");
    }
  }

  private static ConsoleAppender<ILoggingEvent> jsonAppender(LoggerContext context) {
    LogstashEncoder encoder = new LogstashEncoder();
    encoder.setContext(context);
    encoder.start();
    return appender(context, encoder);
  }

  private static ConsoleAppender<ILoggingEvent> consoleAppender(LoggerContext context) {
    PatternLayoutEncoder encoder = new PatternLayoutEncoder();
    encoder.setContext(context);
    encoder.setPattern("%d{HH:mm:ss.SSS} %-5level %logger{36} - %msg%n");
    encoder.start();
    return appender(context, encoder);
  }

  private static ConsoleAppender<ILoggingEvent> appender(
      LoggerContext context, ch.qos.logback.core.encoder.Encoder<ILoggingEvent> encoder) {
    ConsoleAppender<ILoggingEvent> appender = new ConsoleAppender<>();
    appender.setContext(context);
    appender.setEncoder(encoder);
    appender.start();
    return appender;
  }
}
