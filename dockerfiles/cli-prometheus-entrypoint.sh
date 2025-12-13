#!/bin/bash
set -e

# Generate prom-config.yml if WORKER_METRICS_HOST is set
# This enables scraping metrics from a remote worker (e.g., separate ECS task)
if [ -n "$WORKER_METRICS_HOST" ]; then
  cat > /app/prom-config.yml << PROMCFG
global:
  scrape_interval: 1s
scrape_configs:
  - job_name: 'omes-client'
    static_configs:
      - targets: ['localhost:9091']
  - job_name: 'omes-worker'
    static_configs:
      - targets: ['${WORKER_METRICS_HOST}:${WORKER_METRICS_PORT:-9092}']
PROMCFG
  echo "Generated prom-config.yml with worker target: ${WORKER_METRICS_HOST}:${WORKER_METRICS_PORT:-9092}"
fi

# Pass through all arguments to temporal-omes
exec /app/temporal-omes "$@"