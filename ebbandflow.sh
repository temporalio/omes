go run ./cmd run-scenario-with-worker \
  --scenario ebb_and_flow \
  --language go \
  --duration=60s \
  --option phase-time=15s \
  --option min-backlog=0 \
  --option max-backlog=10 \
  --option backlog-log-interval=1s \
  --option fairness-report-interval=10s \
  --option sleep-activity-json=@sleep.json
