#!/usr/bin/env bash
# SAA COGS experiment — useful commands
# See .task/plan.md for context.

## --- Local testing (against embedded dev server) ---

# SAW: 5 iterations
go run ./cmd run-scenario-with-worker --scenario saa_cogs_saw --language go --iterations 5

# SAA: 5 iterations
go run ./cmd run-scenario-with-worker --scenario saa_cogs_saa --language go --iterations 5

# SAW: sustained 60s at 10 starts/s
go run ./cmd run-scenario-with-worker --scenario saa_cogs_saw --language go \
    --duration 60s --max-iterations-per-second 10 --max-concurrent 100

# SAA: sustained 60s at 10 starts/s
go run ./cmd run-scenario-with-worker --scenario saa_cogs_saa --language go \
    --duration 60s --max-iterations-per-second 10 --max-concurrent 100

## --- Cloud cell: s-saa-cogs ---

CELL=s-saa-cogs
NS=${CELL}-marathon.e2e
HOST=${NS}.tmprl-test.cloud:7233

# Verify cell is alive
ct kubectl --context $CELL get pods -n temporal

# Check namespace
omni admintools --context $CELL -- temporal operator namespace describe $NS

# Run worker against cloud cell (in one terminal)
go run ./cmd run-worker --language go --run-id saa-cogs-test \
    --server-address $HOST --namespace $NS --tls \
    --tls-cert-path /tmp/saa-cogs-cert.pem --tls-key-path /tmp/saa-cogs-key.pem

# Run SAW scenario against cloud cell (in another terminal)
go run ./cmd run-scenario --scenario saa_cogs_saw --run-id saa-cogs-test \
    --server-address $HOST --namespace $NS --tls \
    --tls-cert-path /tmp/saa-cogs-cert.pem --tls-key-path /tmp/saa-cogs-key.pem \
    --iterations 5 --do-not-register-search-attributes

# Run SAA scenario against cloud cell
go run ./cmd run-scenario --scenario saa_cogs_saa --run-id saa-cogs-test \
    --server-address $HOST --namespace $NS --tls \
    --tls-cert-path /tmp/saa-cogs-cert.pem --tls-key-path /tmp/saa-cogs-key.pem \
    --iterations 5 --do-not-register-search-attributes

# Grafana dashboard
# https://grafana.tmprl-internal.cloud/d/saacogs/saa-cogs?var-cluster=s-saa-cogs
