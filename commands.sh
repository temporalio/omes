# Shell commands for SAA/SAW load generation scenarios.

# --- Local testing (embedded dev server) ---

go run ./cmd run-scenario-with-worker --scenario workflow_with_single_activity --language go --iterations 5 --embedded-server --option payload-size=1024

go run ./cmd run-scenario-with-worker --scenario standalone_activity --language go --iterations 5 --embedded-server --option payload-size=1024

# --- Cloud cell: s-saa-cogs ---

# Cell support page: https://staging.thundergun.io/support/cells/s-saa-cogs
#   (s-saa* cells are staging/test cells on thundergun, not cloud.temporal.io)
# K8s access: ct k9s --readonly --context s-saa-cogs

# List all k8s namespaces on the cell
ct kubectl --context s-saa-cogs get namespaces

# Verify cell is up
ct kubectl --context s-saa-cogs get pods -n temporal

# List Temporal namespaces on this cell
# Web: https://staging.thundergun.io/support/cells/s-saa-cogs
ct admintools --context s-saa-cogs -- temporal operator namespace list -o json

# Create a namespace
ct admintools --context s-saa-cogs -- temporal operator namespace create saa-cogs

# Grafana dashboards
# Overview: https://grafana.tmprl-internal.cloud/d/e613c827-243e-4759-a5ca-3e334201c124/temporal-cloud-overview
# By namespace: https://grafana.tmprl-internal.cloud/d/iyRCOBD4z/temporal-cloud-external-metrics-by-namespace
# Frontend: https://grafana.tmprl-internal.cloud/d/SxRYJXZMz/frontend
# Matching: https://grafana.tmprl-internal.cloud/d/wuh-8uZGk/matching
# History: https://grafana.tmprl-internal.cloud/d/jh_LXEin2/history

# Run worker (in one terminal)
go run ./workers/go --task-queue omes \
  --server-address TODO \
  --namespace saa-cogs \
  --tls-cert-path TODO \
  --tls-key-path TODO

# Run SAW scenario
go run ./cmd run-scenario --scenario workflow_with_single_activity \
  --server-address TODO \
  --namespace saa-cogs \
  --tls-cert-path TODO \
  --tls-key-path TODO \
  --iterations 100 --max-concurrent 10

# Run SAA scenario
go run ./cmd run-scenario --scenario standalone_activity \
  --server-address TODO \
  --namespace saa-cogs \
  --tls-cert-path TODO \
  --tls-key-path TODO \
  --iterations 100 --max-concurrent 10
