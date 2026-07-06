# frozen_string_literal: true

require 'bundler/setup'
require_relative 'apps/registry'

Apps::Registry.run if __FILE__ == $PROGRAM_NAME
