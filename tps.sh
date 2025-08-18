#!/bin/bash
go run ./cmd run-scenario-with-worker \
  --scenario throughput_stress \
  --language go \
  --duration 5s