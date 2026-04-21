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
    if argv.empty? || argv.first == 'worker'
      worker_argv = argv.first == 'worker' ? argv.drop(1) : argv
      WorkerCLI.run_cli(app.worker, app.client_factory, worker_argv)
    elsif argv.first == 'project-server'
      if app.project.nil?
        raise SystemExit, 'Wanted project-server but no project handlers registered for this app'
      end

      ProjectCLI.run_cli(app.project, app.client_factory, argv.drop(1))
    else
      raise SystemExit, "Unknown command: #{argv.first(1)}. Expected 'worker' or 'project-server'"
    end
  end
end
