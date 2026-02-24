# Workflowtests Docker Setup

This directory contains Dockerfiles and Compose assets for running workflow tests with Temporal and Prometheus.

## Files

1. `docker-compose.workflowtests.yml`: Temporal dev server, UI, Prometheus, and language runner/worker services.
2. `workflowtest-*.Dockerfile`: reusable language base images.
3. `workflowtest-project.Dockerfile`: thin project overlay image.
4. `prometheus.compose.yml`: Prometheus scrape targets for a Docker worker container (`omes-worker`).
5. `prometheus.host.yml`: Prometheus scrape targets for a worker running on the host machine.

## Recommended Modes

Use one of these two setups.

1. **All Docker**: run infra in Docker and run runner/worker in Docker.
2. **Docker infra + local omes**: run infra in Docker but run `omes` from your machine.

Important worker-mode rule:

- Choose one worker source per run: runner-managed (`--spawn-worker`) or a dedicated worker (container/local process).
- Avoid using both against the same task queue in one run. Both workers will poll tasks, so load and metrics are split across processes and harder to attribute.
- Temporal SDK metrics (`temporal_*`) are available in either mode. Dedicated worker mode is primarily for cleaner `process_*` attribution across repeated runs.

## Shared Prerequisite (Docker image build)

Build the language image(s) you plan to run:

```bash
docker build -f workflowtests/dockerfiles/workflowtest-go.Dockerfile \
  -t omes-workflowtest-go-base:local .

docker build -f workflowtests/dockerfiles/workflowtest-python.Dockerfile \
  -t omes-workflowtest-python-base:local .

docker build -f workflowtests/dockerfiles/workflowtest-typescript.Dockerfile \
  -t omes-workflowtest-typescript-base:local .
```

Optional shell setup:

```bash
export COMPOSE_FILE=workflowtests/dockerfiles/docker-compose.workflowtests.yml
```

If you do not set `COMPOSE_FILE`, add `-f workflowtests/dockerfiles/docker-compose.workflowtests.yml` to each compose command.

## Mode 1: All Docker

Use this when runner and worker both run in containers.

### 1) Start infra

Use Compose scrape config (`prometheus.compose.yml`):

```bash
docker compose up -d temporal-dev temporal-ui prometheus
```

### 2) Pick worker strategy

#### Option A (recommended for process-metrics analysis): dedicated worker container

SDK metrics are available in both worker strategies. This option is mainly for stable, repeatable `process_*` attribution.

Go example:

```bash
docker compose run -d --name omes-worker --service-ports worker-go \
  exec \
  --language go \
  --project-dir /app/workflowtests/go/tests/simpletest \
  -- \
  worker \
  --task-queue omes-compose-q \
  --server-address temporal-dev:7233 \
  --namespace default \
  --prom-listen-address 0.0.0.0:19090 \
  --worker-process-metrics-address 0.0.0.0:9091
```

Run tests from runner container:

```bash
docker compose run --rm runner-go \
  workflow \
  --language go \
  --project-dir /app/workflowtests/go/tests/simpletest \
  --iterations 10 \
  --task-queue omes-compose-q \
  --server-address temporal-dev:7233 \
  --namespace default
```

#### Option B: runner-managed worker with `--spawn-worker`

Use for quick correctness runs and SDK metrics collection. For process-metrics comparisons across runs, dedicated worker mode is usually cleaner.

```bash
docker compose run --rm runner-go \
  workflow \
  --language go \
  --project-dir /app/workflowtests/go/tests/simpletest \
  --iterations 10 \
  --spawn-worker \
  --task-queue omes-compose-q \
  --server-address temporal-dev:7233 \
  --namespace default
```

## Mode 2: Docker infra + Local omes

Use this when Temporal/Prometheus run in Docker but you run `omes` locally.

### 1) Start infra with host scrape config

When worker runs on host, use `prometheus.host.yml` (not `prometheus.compose.yml`):

```bash
PROM_CONFIG_FILE=prometheus.host.yml docker compose up -d --force-recreate temporal-dev temporal-ui prometheus
```

Use `--force-recreate` when changing `PROM_CONFIG_FILE` so Prometheus remounts the selected config.

### 2) Run tests locally

You can run locally with either:

1. `omes workflow ... --spawn-worker`
2. a separate local worker process + `omes workflow ...` without `--spawn-worker`

If you need stable `process_*` attribution, prefer the dedicated local worker process path.

For host scraping, expose worker metrics on host ports used by `prometheus.host.yml`:

1. worker SDK metrics: `--prom-listen-address 0.0.0.0:19090`
2. optional process metrics: `--worker-process-metrics-address 0.0.0.0:9091`

## Endpoints and Quick Checks

1. Temporal UI: `http://localhost:8080`
2. Prometheus: `http://localhost:9090`
3. Worker SDK metrics: `http://localhost:19090/metrics`
4. Optional worker process metrics: `http://localhost:9091/metrics`

```bash
curl -s http://localhost:9090/api/v1/targets | rg 'omes_worker_app'
curl -s http://localhost:9090/api/v1/targets | rg 'omes_worker_process'
```

## Cleanup

```bash
docker rm -f omes-worker
docker compose down -v
```
