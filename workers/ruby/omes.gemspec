Gem::Specification.new do |s|
  s.name = 'omes'
  s.version = '0.1.0'
  s.summary = 'Temporal omes Ruby worker'
  s.authors = ['Temporal Technologies Inc']
  s.email = ['sdk@temporal.io']
  s.license = 'MIT'
  s.files = Dir['**/*.rb']
  s.require_paths = ['.']
  s.add_dependency 'grpc', '~> 1.80'
  s.add_dependency 'google-protobuf', '~> 4.0'
  s.add_dependency 'temporalio', '~> 1.3'
  s.metadata['rubygems_mfa_required'] = 'true'
end
