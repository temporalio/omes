# frozen_string_literal: true

require 'bundler/setup'
require 'harness'
require_relative 'kitchen_sink_app'

Harness.run(KitchenSinkApp.app) if __FILE__ == $PROGRAM_NAME
