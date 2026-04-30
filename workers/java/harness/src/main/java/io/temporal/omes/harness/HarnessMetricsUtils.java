package io.temporal.omes.harness;

import static java.nio.charset.StandardCharsets.UTF_8;

import com.sun.net.httpserver.HttpServer;
import io.micrometer.prometheus.PrometheusMeterRegistry;
import java.io.IOException;
import java.io.OutputStream;
import java.net.InetSocketAddress;

final class HarnessMetricsUtils {
  private HarnessMetricsUtils() {}

  static HttpServer startPrometheusScrapeEndpoint(
      PrometheusMeterRegistry registry, String path, String address) {
    try {
      String[] parts = address.split(":");
      if (parts.length > 2) {
        throw new IllegalArgumentException("Invalid address: " + address);
      }
      String host = parts[0];
      int port = parts.length == 2 ? Integer.parseInt(parts[1]) : 0;

      HttpServer server = HttpServer.create(new InetSocketAddress(host, port), 0);
      server.createContext(
          path,
          httpExchange -> {
            String response = registry.scrape();
            httpExchange
                .getResponseHeaders()
                .set("Content-Type", "text/plain; version=0.0.4; charset=utf-8");
            httpExchange.sendResponseHeaders(200, response.getBytes(UTF_8).length);
            try (OutputStream output = httpExchange.getResponseBody()) {
              output.write(response.getBytes(UTF_8));
            }
          });

      server.start();
      return server;
    } catch (IOException e) {
      throw new RuntimeException(e);
    }
  }
}
