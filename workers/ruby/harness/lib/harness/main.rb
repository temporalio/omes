# frozen_string_literal: true

require_relative 'project'
require_relative 'worker'

module Harness
  class App
    attr_reader :worker, :client_factory, :project

    def initialize(worker:, client_factory:, project: nil)
      @worker = worker
      @client_factory = client_factory
      @project = project
    end
  end

  def self.run(app, argv = ARGV)
    argv = Array(argv).dup
    if argv.first == 'worker'
      WorkerCLI.run_cli(app.worker, app.client_factory, argv.drop(1))
    elsif argv.first == 'project-server'
      raise SystemExit, 'Wanted project-server but no project handlers registered for this app' if app.project.nil?

      ProjectCLI.run_cli(app.project, app.client_factory, argv.drop(1))
    else
      raise SystemExit, "Unknown command: #{argv.first(1)}. Expected 'worker' or 'project-server'"
    end
  end
end
