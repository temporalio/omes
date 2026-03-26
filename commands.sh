# Shell commands for SAA/SAW load generation scenarios.

# --- Local testing (embedded dev server) ---

go run ./cmd run-scenario-with-worker --scenario workflow_with_single_activity --language go --iterations 5 --embedded-server --option payload-size=1024
go run ./cmd run-scenario-with-worker --scenario standalone_activity --language go --iterations 5 --embedded-server --option payload-size=1024

# --- Cloud cell: s-saa-cogs ---

# Cell support page: https://staging.thundergun.io/support/cells/s-saa-cogs
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
  --namespace $NS.temporal-dev \
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

# Dyanamic config
# https://staging.thundergun.io/support/namespaces/saa-cogs-4.temporal-devo
{
  "activity.enableStandalone": true,
  "history.enableChasm": true
}

# scale canary to 0
ct kubectl --context s-saa-cogs patch deployment/temporal-go-canary -n temporal -p '{"spec":{"replicas":0}}'

# SAW
go run ./cmd run-scenario-with-worker \
  --scenario workflow_with_single_activity \
  --language go \
  --run-id run-2 \
  --duration 1h --max-concurrent 500 --max-iterations-per-second 100 \
  --option payload-size=1024 \
  --option fail-for-attempts=9 \
  --worker-max-concurrent-workflow-pollers 40 \
  --worker-max-concurrent-workflow-tasks 500 \
  --worker-max-concurrent-activity-pollers 40 \
  --worker-max-concurrent-activities 500 \
  --do-not-register-search-attributes \
  --server-address us-west-2.aws.api.tmprl-test.cloud:7233 \
  --namespace $NS.temporal-dev \
  --tls \
  --disable-tls-host-verification \
  --auth-header "Bearer $TEMPORAL_API_KEY"

# SAA
go run ./cmd run-scenario-with-worker \
  --scenario standalone_activity \
  --language go \
  --run-id run-2 \
  --duration 1h --max-concurrent 500 --max-iterations-per-second 1000 \
  --option payload-size=102400 \
  --worker-max-concurrent-activity-pollers 40 \
  --worker-max-concurrent-activities 500 \
  --do-not-register-search-attributes \
  --server-address us-west-2.aws.api.tmprl-test.cloud:7233 \
  --namespace $NS.temporal-dev \
  --tls \
  --disable-tls-host-verification \
  --auth-header "Bearer $TEMPORAL_API_KEY"

ct ocld test dynamic-config namespace get -n saa-cogs-4.temporal-dev

# 88ms RTT
for i in $(seq 10); do curl -s -o /dev/null -w '%{time_connect}\n' https://us-west-2.aws.api.tmprl-test.cloud:7233; done

# parameterized

CELL=s-saa-cogs
NS=saa-cogs-4

ct admintools --context $CELL -- temporal operator namespace list
ct ocld test namespace create \
  --namespace $CELL.temporal-dev \
  --region us-west-2 \
  --cloud-provider aws \
  --retention 1 \
  --placement-override-cell-id $CELL \
  --auth-method api_key

ct admintools --context $CELL -- temporal operator namespace list
nslookup $CELL.temporal-dev.tmprl-test.cloud

export TEMPORAL_API_KEY=xxx
export TEMPORAL_ADDRESS=us-west-2.aws.api.tmprl-test.cloud:7233
export TEMPORAL_NAMESPACE=$CELL.temporal-dev
export TEMPORAL_TLS=true
export TEMPORAL_TLS_DISABLE_HOST_VERIFICATION=true

ct kubectl --context $CELL patch deployment/temporal-go-canary -n temporal -p '{"spec":{"replicas":0}}'
