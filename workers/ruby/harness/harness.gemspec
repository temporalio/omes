# frozen_string_literal: true

Gem::Specification.new do |s|
  s.name = 'harness'
  s.version = '0.1.0'
  s.summary = 'Ruby harness for Omes projects'
  s.authors = ['Temporal Technologies Inc']
  s.email = ['sdk@temporal.io']
  s.license = 'MIT'
  s.required_ruby_version = '>= 3.3'
  s.files = Dir['lib/**/*.rb'] + Dir['sig/**/*.rbs']
  s.require_paths = ['lib']
  s.add_dependency 'google-protobuf', '~> 4.0'
  s.add_dependency 'grpc', '~> 1.80'
  s.add_dependency 'temporalio', '~> 1.3'
  s.metadata['rubygems_mfa_required'] = 'true'
end
