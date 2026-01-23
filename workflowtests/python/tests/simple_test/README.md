# Simple Test

A minimal workflow load test example.

## Running Locally (Recommended)

Use `omes workflow` for the simplest local development experience:

```bash
# One command runs everything
omes workflow \
  --language python \
  --client-command "python client.py" \
  --worker-command "python worker_main.py" \
  --iterations 10

# With a custom SDK version
omes workflow \
  --language python \
  --sdk-version /path/to/sdk \
  --client-command "python client.py" \
  --worker-command "python worker_main.py" \
  --iterations 100 \
  --max-concurrent 10
```

## Running Against Remote Containers

```bash
# Connect to pre-running ECS containers
omes workflow \
  --client-url http://client.ecs.internal:8080 \
  --worker-url http://worker.ecs.internal:8081 \
  --iterations 1000 \
  --max-concurrent 50
```

## Architecture (Local Mode)

```
┌─────────────────────────────────────────────────────────────────┐
│                      omes workflow                               │
│                                                                  │
│   ┌─────────────────┐    ┌─────────────────┐                    │
│   │   Load Runner   │    │ Worker Lifecycle │                   │
│   │                 │    │   (HTTP server)  │                   │
│   └────────┬────────┘    └────────┬────────┘                    │
│            │                      │                              │
│            │ HTTP                 │ spawns immediately           │
│            ▼                      ▼                              │
│   ┌─────────────────┐    ┌─────────────────┐                    │
│   │   client.py     │    │  worker_main.py │                    │
│   │ (OmesClient     │    │ (plain Temporal │                    │
│   │  Starter)       │    │  worker)        │                    │
│   └─────────────────┘    └─────────────────┘                    │
└─────────────────────────────────────────────────────────────────┘
```
