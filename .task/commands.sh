#!/bin/bash
# SAA COGS experiment: useful commands
# Cell: s-saa-cogs

CELL="s-saa-cogs"
# Namespace TBD — confirm with Stephen. Likely one of:
#   ${CELL}-marathon.e2e
#   or a custom namespace
NS="${CELL}-marathon.e2e"
HOST="${NS}.tmprl-test.cloud:7233"

# ── Local testing ────────────────────────────────────────────────────────────
# Start local dev server (with standalone activity support)
temporal server start-dev --headless --log-level warn

# SAW: single-activity workflow (baseline)
go run ./cmd run-scenario-with-worker \
  --scenario saa_cogs_saw --language go \
  --iterations 10 --max-concurrent 5 --run-id saw-local-1

# SAA: standalone activity (no workflow)
go run ./cmd run-scenario-with-worker \
  --scenario saa_cogs_saa --language go \
  --iterations 10 --max-concurrent 5 --run-id saa-local-1

# Rate-limited run (e.g. 10 executions/s for 5 minutes)
go run ./cmd run-scenario-with-worker \
  --scenario saa_cogs_saw --language go \
  --duration 5m --max-iterations-per-second 10 --max-concurrent 100 --run-id saw-rate-1

# ── Cloud cell operations ────────────────────────────────────────────────────
# Check cell pods
ct kubectl --context $CELL get pods -n temporal

# Check namespace
omni admintools --context $CELL -- temporal operator namespace describe $NS

# ── Running against cloud cell ───────────────────────────────────────────────
# Requires TLS certs or API key. Two options:

# Option A: mTLS
# TLS_CERT=path/to/cert.pem
# TLS_KEY=path/to/key.pem
# go run ./cmd run-scenario \
#   --scenario saa_cogs_saw \
#   --server-address $HOST --namespace $NS \
#   --tls --tls-cert-path $TLS_CERT --tls-key-path $TLS_KEY \
#   --do-not-register-search-attributes \
#   --iterations 1 --run-id saw-cloud-1

# Option B: API key
# go run ./cmd run-scenario \
#   --scenario saa_cogs_saw \
#   --server-address $HOST --namespace $NS \
#   --tls --auth-header "Bearer $API_KEY" \
#   --do-not-register-search-attributes \
#   --iterations 1 --run-id saw-cloud-1

# Worker (separate terminal, same auth flags)
# go run ./cmd run-worker \
#   --scenario saa_cogs_saw --language go \
#   --server-address $HOST --namespace $NS \
#   --tls --tls-cert-path $TLS_CERT --tls-key-path $TLS_KEY \
#   --run-id saw-cloud-1

# ── Grafana ──────────────────────────────────────────────────────────────────
# Dashboard: https://grafana.tmprl-internal.cloud/d/saacogs/saa-cogs
# Set cluster variable to: s-saa-cogs
