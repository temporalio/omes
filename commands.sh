# Shell commands for SAA/SAW load generation scenarios.

# --- Local testing (embedded dev server) ---

go run ./cmd run-scenario-with-worker --scenario workflow_with_single_activity --language go --iterations 5 --embedded-server --option payload-size=1024

go run ./cmd run-scenario-with-worker --scenario standalone_activity --language go --iterations 5 --embedded-server --option payload-size=1024

# --- Cloud cell: s-saa-cogs ---

# Cell support page: https://staging.thundergun.io/support/cells/s-saa-cogs
#   (s-saa* cells are staging/test cells on thundergun, not cloud.temporal.io)
# K8s access: ct k9s --readonly --context s-saa-cogs

# List all k8s namespaces on the cell
ct kubectl --context s-saa-cogs get namespaces -o json

# Verify cell is up
ct kubectl --context s-saa-cogs get pods -n temporal

# List Temporal namespaces on this cell
# Web: https://staging.thundergun.io/support/cells/s-saa-cogs
ct admintools --context s-saa-cogs -- temporal operator namespace list -o json

# Create namespace pinned to the cell
ct ocld test namespace create \
  --namespace saa-cogs-4.temporal-dev \
  --region us-west-2 \
  --cloud-provider aws \
  --retention 1 \
  --placement-override-cell-id s-saa-cogs \
  --auth-method api_key

# DNS should resolve
nslookup saa-cogs-4.temporal-dev.tmprl-test.cloud

# Namespace should appear on the cell
ct admintools --context s-saa-cogs -- temporal operator namespace list

# Namespace should be active with API key auth enabled
# Output contains grpcAddress
ct ocld test cloud-apis namespaces get -n saa-cogs-4.temporal-dev

export TEMPORAL_API_KEY=xxx
export TEMPORAL_ADDRESS=us-west-2.aws.api.tmprl-test.cloud:7233
export TEMPORAL_NAMESPACE=saa-cogs-4.temporal-dev
export TEMPORAL_TLS=true
export TEMPORAL_TLS_DISABLE_HOST_VERIFICATION=true

ct admintools --context s-saa-cogs -- temporal operator search-attribute create \
  --namespace saa-cogs-4.temporal-dev --name OmesExecutionID --type Keyword

# Run worker (in one terminal)
go run ./cmd run-worker \
  --run-id run-1 \
  --scenario workflow_with_single_activity \
  --language go \
  --server-address us-west-2.aws.api.tmprl-test.cloud:7233 \
  --namespace saa-cogs-4.temporal-dev \
  --tls \
  --disable-tls-host-verification \
  --auth-header "Bearer $TEMPORAL_API_KEY"

# Run SAW scenario
go run ./cmd run-scenario --scenario workflow_with_single_activity \
  --run-id run-1 \
  --iterations 100 --max-concurrent 10 \
  --do-not-register-search-attributes \
  --server-address us-west-2.aws.api.tmprl-test.cloud:7233 \
  --namespace saa-cogs-4.temporal-dev \
  --tls \
  --disable-tls-host-verification \
  --auth-header "Bearer $TEMPORAL_API_KEY"

# Run SAA scenario
go run ./cmd run-scenario --scenario standalone_activity \
  --run-id run-1 \
  --iterations 100 --max-concurrent 10 \
  --do-not-register-search-attributes \
  --server-address us-west-2.aws.api.tmprl-test.cloud:7233 \
  --namespace saa-cogs-4.temporal-dev \
  --tls \
  --disable-tls-host-verification \
  --auth-header "Bearer $TEMPORAL_API_KEY"

# Grafana dashboards
# Overview: https://grafana.tmprl-internal.cloud/d/e613c827-243e-4759-a5ca-3e334201c124/temporal-cloud-overview
# By namespace: https://grafana.tmprl-internal.cloud/d/iyRCOBD4z/temporal-cloud-external-metrics-by-namespace
# Frontend: https://grafana.tmprl-internal.cloud/d/SxRYJXZMz/frontend
# Matching: https://grafana.tmprl-internal.cloud/d/wuh-8uZGk/matching
# History: https://grafana.tmprl-internal.cloud/d/jh_LXEin2/history
