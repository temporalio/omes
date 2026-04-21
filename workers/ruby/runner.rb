require_relative 'kitchen_sink_app'
require_relative 'projects/harness'

Harness.run(KitchenSinkApp.app) if __FILE__ == $PROGRAM_NAME
