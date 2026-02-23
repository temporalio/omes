# Workflowtests Docker + Compose

This directory contains Dockerfiles and compose assets for workflow test runs.

Files:

1. `docker-compose.workflowtests.yml`: Temporal dev server + UI + metrics infra + language runner/worker services
2. `workflowtest-*.Dockerfile`: reusable language base images
3. `workflowtest-project.Dockerfile`: thin project overlay image
4. `prometheus.compose.yml`: Prometheus scrape config for Docker Compose setup
5. `prometheus.host.yml`: Prometheus scrape config for local setup
6. `.env.example`: optional compose env overrides

## Local Setup

Local setup is intentionally minimal:

1. Run Temporal + worker + runner on host.
2. Expose worker metrics on `localhost:19090` (`--prom-listen-address`).
3. Run local Prometheus with this template config:
   - `workflowtests/dockerfiles/prometheus.host.yml`

### Start Prometheus

```bash
prometheus \
  --config.file=workflowtests/dockerfiles/prometheus.host.yml \
  --web.listen-address=:9090
```

### View Metrics

1. Prometheus UI: `http://localhost:9090`
2. Worker endpoint: `http://localhost:19090/metrics`
3. Targets API quick check:

```bash
curl -s http://localhost:9090/api/v1/targets | rg 'omes-worker-host'
```

### Optional JSON Export

```bash
curl -sG http://localhost:9090/api/v1/query_range \
  --data-urlencode 'query=rate(temporal_request_latency_count{job="omes-worker-host"}[1m])' \
  --data-urlencode "start=$(date -u -v-10M +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d '10 minutes ago' +%Y-%m-%dT%H:%M:%SZ)" \
  --data-urlencode "end=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --data-urlencode 'step=15s' \
  > worker-sdk-metrics.json
```

## Docker Compose Setup

Default flow:

1. Start infra once
2. Start a dedicated worker container
3. Run one or many runner passes against that worker

This keeps worker metrics stable and avoids worker rebuild/restart churn during repeated test runs.

### Prerequisite: Build Base Images

Build the language image(s) you plan to run:

```bash
docker build -f workflowtests/dockerfiles/workflowtest-go.Dockerfile \
  -t omes-workflowtest-go-base:local .

docker build -f workflowtests/dockerfiles/workflowtest-python.Dockerfile \
  -t omes-workflowtest-python-base:local .

docker build -f workflowtests/dockerfiles/workflowtest-typescript.Dockerfile \
  -t omes-workflowtest-typescript-base:local .
```

### Optional Shell Setup

Optional once per shell (avoid repeating `-f ...`):

```bash
export COMPOSE_FILE=workflowtests/dockerfiles/docker-compose.workflowtests.yml
```

If you do not set `COMPOSE_FILE`, add
`-f workflowtests/dockerfiles/docker-compose.workflowtests.yml` to each compose command.

### Optional macOS Socket Override

If cAdvisor cannot read Docker metadata on macOS Docker Desktop:

```bash
cp workflowtests/dockerfiles/.env.example workflowtests/dockerfiles/.env
echo "DOCKER_SOCKET=$HOME/.docker/run/docker.sock" >> workflowtests/dockerfiles/.env
docker compose --env-file workflowtests/dockerfiles/.env up -d temporal-dev temporal-ui cadvisor prometheus
```

On Linux, the default `/var/run/docker.sock` usually works without overrides.

### Start Infra

```bash
docker compose up -d temporal-dev temporal-ui cadvisor prometheus
```

### Start Dedicated Worker (Default)

Compose mounts the repo writable at `/app/workflowtests` so `omes` can create
temporary `program-*` SDK build directories under test projects.

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
  --prom-listen-address 0.0.0.0:19090
```

For Python/TypeScript, switch to `worker-python` / `worker-typescript` and update
`--language` + `--project-dir`.

### Run Test Passes

Go example:

```bash
docker compose run --rm runner-go \
  workflow \
  --language go \
  --project-dir /app/workflowtests/go/tests/simpletest \
  --iterations 10 \
  --task-queue omes-compose-q \
  --server-address temporal-dev:7233 \
  --namespace default \
  --client-port 18180
```

Python example:

```bash
docker compose run --rm runner-python \
  workflow \
  --language python \
  --project-dir /app/workflowtests/python/tests/simple_test \
  --iterations 10 \
  --task-queue omes-compose-q \
  --server-address temporal-dev:7233 \
  --namespace default \
  --client-port 18180
```

### Convenience Path: `--spawn-worker`

One-shot correctness run (no dedicated worker container):

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

This is fine for quick checks, but dedicated worker mode is preferred for stable compose worker metrics.

### Metrics Flow

1. Worker SDK/application metrics: Prometheus scrapes `omes-worker:19090` via `prometheus.compose.yml`.
2. Container CPU/memory metrics: cAdvisor exposes `/metrics`; Prometheus scrapes `cadvisor:8080`.
3. cAdvisor gathers container stats using mounted host paths + Docker socket from compose.
4. cAdvisor metrics include all Docker containers it can see from the mounted socket/runtime paths, not just `omes-worker`.

### Endpoints and Checks

1. Temporal UI: `http://localhost:8080`
2. Prometheus: `http://localhost:9090`
3. Worker metrics (while dedicated worker is running): `http://localhost:19090/metrics`
4. cAdvisor metrics: `http://localhost:8081/metrics`

```bash
curl -s http://localhost:19090/metrics | rg '^temporal_' | head
curl -s http://localhost:9090/api/v1/targets | rg 'omes-worker|cadvisor'
```

### Cleanup

```bash
docker rm -f omes-worker
docker compose down -v
```
