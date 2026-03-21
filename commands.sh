# Shell commands for SAA/SAW load generation scenarios.

# --- Local testing (embedded dev server) ---

go run ./cmd run-scenario-with-worker --scenario workflow_with_single_activity --language go --iterations 5 --embedded-server --option payload-size=1024

go run ./cmd run-scenario-with-worker --scenario standalone_activity --language go --iterations 5 --embedded-server --option payload-size=1024

# --- Cloud cell: s-saa-cogs ---

# List all k8s namespaces on the cell
ct kubectl --context s-saa-cogs get namespaces

# Verify cell is up
ct kubectl --context s-saa-cogs get pods -n temporal

# Check namespace
ct admintools --context s-saa-cogs -- temporal operator namespace describe s-saa-cogs-marathon.e2e

# Run worker (in one terminal)
go run ./workers/go --task-queue omes \
  --server-address TODO \
  --namespace s-saa-cogs-marathon.e2e \
  --tls-cert-path TODO \
  --tls-key-path TODO

# Run SAW scenario
go run ./cmd run-scenario --scenario workflow_with_single_activity \
  --server-address TODO \
  --namespace s-saa-cogs-marathon.e2e \
  --tls-cert-path TODO \
  --tls-key-path TODO \
  --iterations 100 --max-concurrent 10

# Run SAA scenario
go run ./cmd run-scenario --scenario standalone_activity \
  --server-address TODO \
  --namespace s-saa-cogs-marathon.e2e \
  --tls-cert-path TODO \
  --tls-key-path TODO \
  --iterations 100 --max-concurrent 10
