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

# Make dir to store metrics
mkdir -p /app/metrics

set +e
# Pass through all arguments to temporal-omes
/app/temporal-omes "$@"
EXIT_CODE=$?
set -e

if [ -n "$S3_BUCKET_URI" ] && [ $EXIT_CODE -eq 0 ]; then
  DATE=$(date +%Y-%m-%d)
  S3_KEY="${S3_BUCKET_URI}/is_experiment=${IS_EXPERIMENT}/language=${LANGUAGE}/date=${DATE}/run-id=${RUN_ID}/"

  echo "Uploading metrics to S3: ${S3_KEY}"
  aws s3 cp /app/metrics/ "${S3_KEY}" --recursive
else
  if [ $EXIT_CODE -ne 0 ]; then
    echo "Scenario failed. Skipping S3 upload."
  fi
fi

exit $EXIT_CODE