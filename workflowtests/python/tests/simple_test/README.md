# Simple Test

Minimal Python workflow load test example for the new starter pattern.

## Run worker only

```bash
omes exec --language python --project-dir workflowtests/python/tests/simple_test -- \
  worker --task-queue omes-smoke-q --server-address localhost:7233 --namespace default
```

## Run load test against existing worker

```bash
omes workflow --language python --project-dir workflowtests/python/tests/simple_test \
  --iterations 10 --task-queue omes-smoke-q --server-address localhost:7233 --namespace default
```

## Run load test and spawn worker automatically

```bash
omes workflow --language python --project-dir workflowtests/python/tests/simple_test \
  --spawn-worker --iterations 10 --task-queue omes-smoke-q --server-address localhost:7233 --namespace default
```

## SDK override examples

```bash
# Released SDK version
omes workflow --language python --project-dir workflowtests/python/tests/simple_test \
  --version 1.14.0 --iterations 10 --task-queue omes-smoke-q

# Local SDK checkout
omes workflow --language python --project-dir workflowtests/python/tests/simple_test \
  --version ../sdk-python --iterations 10 --task-queue omes-smoke-q
```
