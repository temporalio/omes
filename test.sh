./scripts/build_proto.sh
go run ./cmd run-scenario-with-worker --scenario throughput_stress --language go --run-id "run-$(date +%s)" --duration 10s