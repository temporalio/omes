# frozen_string_literal: true

require 'temporalio/client'
require 'temporalio/runtime'

module Harness
  ClientConfig = Struct.new(
    :target_host,
    :namespace,
    :api_key,
    :tls,
    :runtime,
    keyword_init: true
  )

  module ClientHelpers
    module_function

    def default_client_factory(config)
      Temporalio::Client.connect(
        config.target_host,
        config.namespace,
        api_key: config.api_key,
        tls: config.tls,
        runtime: config.runtime
      )
    end

    def build_client_config(
      server_address:,
      namespace:,
      auth_header:,
      tls:,
      tls_cert_path:,
      tls_key_path:,
      tls_server_name: nil,
      disable_host_verification: false,
      prom_listen_address: nil
    )
      ClientConfig.new(
        target_host: server_address,
        namespace: namespace,
        api_key: build_api_key(auth_header),
        tls: build_tls_config(
          tls: tls,
          tls_cert_path: tls_cert_path,
          tls_key_path: tls_key_path,
          tls_server_name: tls_server_name,
          disable_host_verification: disable_host_verification
        ),
        runtime: build_runtime(prom_listen_address)
      )
    end

    def build_api_key(auth_header)
      return nil if auth_header.to_s.empty?

      auth_header.delete_prefix('Bearer ')
    end

    def build_tls_config(
      tls:,
      tls_cert_path:,
      tls_key_path:,
      tls_server_name: nil,
      disable_host_verification: false
    )
      warn('disable_host_verification is not supported by the Ruby SDK; ignoring') if disable_host_verification

      unless tls_cert_path.to_s.empty?
        raise ArgumentError, 'Client cert specified, but not client key!' if tls_key_path.to_s.empty?

        return Temporalio::Client::Connection::TLSOptions.new(
          client_cert: File.binread(tls_cert_path),
          client_private_key: File.binread(tls_key_path),
          domain: tls_server_name
        )
      end

      raise ArgumentError, 'Client key specified, but not client cert!' unless tls_key_path.to_s.empty?

      return Temporalio::Client::Connection::TLSOptions.new(domain: tls_server_name) if tls

      nil
    end

    def build_runtime(prom_listen_address)
      prometheus = if prom_listen_address
                     Temporalio::Runtime::PrometheusMetricsOptions.new(
                       bind_address: prom_listen_address,
                       durations_as_seconds: true
                     )
                   end

      Temporalio::Runtime.new(
        telemetry: Temporalio::Runtime::TelemetryOptions.new(
          metrics: prometheus,
          logging: Temporalio::Runtime::LoggingOptions.new(
            log_filter: Temporalio::Runtime::LoggingFilterOptions.new(
              core_level: ENV.fetch('TEMPORAL_CORE_LOG_LEVEL', 'INFO'),
              other_level: 'WARN'
            )
          )
        )
      )
    end
  end

  def self.default_client_factory(config)
    ClientHelpers.default_client_factory(config)
  end
end
