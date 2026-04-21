# frozen_string_literal: true

require 'json'
require 'logger'
require 'time'

module Harness
  module Helpers
    NAME_TO_LEVEL = {
      'PANIC' => Logger::FATAL,
      'FATAL' => Logger::FATAL,
      'ERROR' => Logger::ERROR,
      'WARN' => Logger::WARN,
      'INFO' => Logger::INFO,
      'DEBUG' => Logger::DEBUG,
      'NOTSET' => Logger::DEBUG
    }.freeze

    module_function

    def configure_logger(log_level, log_encoding)
      logger = Logger.new($stderr)
      logger.level = NAME_TO_LEVEL.fetch(log_level.upcase, Logger::INFO)
      if log_encoding == 'json'
        logger.formatter = proc do |severity, datetime, _progname, message|
          "#{JSON.generate(message: message, level: severity, timestamp: datetime.iso8601)}\n"
        end
      end
      logger
    end
  end
end
