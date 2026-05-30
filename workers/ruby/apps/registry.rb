# frozen_string_literal: true

require 'harness'
require 'optparse'
require_relative 'worker/app'

module Apps
  module Registry
    DEFAULT_APP_NAME = 'worker'
    APPS = {
      'worker' => Worker.method(:app)
    }.freeze

    module_function

    def run(argv = ARGV)
      argv = Array(argv).dup
      app_name = DEFAULT_APP_NAME
      OptionParser.new do |opts|
        opts.on('--app APP') { |value| app_name = value }
      end.order!(argv)

      app = APPS[app_name]
      raise SystemExit, "unknown Ruby worker app #{app_name.inspect}" if app_name.empty? || app.nil?

      Harness.run(app.call, argv)
    end
  end
end
