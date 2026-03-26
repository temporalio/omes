# Deploying OMES to a Cloud Test Cell (API Key Auth)

Step-by-step record of deploying omes workers to cell `s-saa-cogs`, targeting
namespace `saa-cogs-4.temporal-dev` with API key authentication. March 2026.

## Background

The cell `s-saa-cogs` was created by the Cloud Capacity team via `omni scaffold
environment create` (without `--omes-enabled`). The namespace
`saa-cogs-4.temporal-dev` was created separately with `--auth-method api_key`.
We needed to deploy omes workers running custom scenarios
(`workflow_with_single_activity`, `standalone_activity`) from the `saa-cogs`
branch for SAA COGS measurement experiments.

## Problems encountered and solutions

### 1. Scaffold omes setup failed: missing search attributes

```
ct scaffold environment omes setup \
  --cell-id s-saa-cogs \
  --namespace saa-cogs-4.temporal-dev \
  --run-id run-1
```

Failed with: `namespace saa-cogs-4.temporal-dev missing the following search
attributes CustomKeywordField,CustomStringField,KS_Int,KS_Keyword`.

The search attributes existed on the cell (created via `ct admintools`) but the
scaffold workflow validates via the **cloud control plane API**, not direct cell
access.

**Fix**: Create search attributes via the cloud API:

```bash
ct ocld test cloud-apis namespaces update \
  -n saa-cogs-4.temporal-dev \
  --sa "CustomKeywordField=Keyword" \
  --sa "CustomStringField=Text" \
  --sa "KS_Int=Int" \
  --sa "KS_Keyword=Keyword"
```

### 2. Scaffold deployment failed: JWT authentication error

Scaffold creates a deployment with mTLS cert-based auth (`--tls-cert-path`,
`--tls-key-path`), but the namespace uses API key auth. Pods crash with:

```
FATAL  failed to dial: Jwt is missing
```

**Fix**: Patch the deployment to use API key auth instead of mTLS (see scripts
below). This requires:
- A k8s secret containing the API key
- Deployment args using `--auth-header` and `--disable-tls-host-verification`
  instead of cert paths
- The server address changed to the API gateway endpoint
  (`us-west-2.aws.api.tmprl-test.cloud:7233`) rather than the direct cell
  address

### 3. Custom image not in ECR

The omes CI pushes images to **Docker Hub** (`temporaliotest/omes`), not ECR.
Test cells enforce a Kyverno `restrict-image-registries` policy that only allows
images from approved ECR registries.

**Fix**: Mirror the image from Docker Hub to ECR using `skopeo` (see below).

### 4. Custom scenarios not in stock image

The stock omes image (`go-1.35.0`) doesn't contain the `workflow_with_single_activity`
or `standalone_activity` scenarios added on the `saa-cogs` branch.

**Fix**: Trigger the GitHub Actions CI on the branch to build a new image, then
mirror it to ECR (see below).

## Procedure

### Step 1: Create the omes deployment via scaffold

This creates the k8s deployment, namespace on the omes k8s namespace, and TLS
secrets (even though we won't use mTLS).

```bash
ct scaffold environment omes setup \
  --cell-id s-saa-cogs \
  --namespace saa-cogs-4.temporal-dev \
  --run-id run-1
```

Track the workflow at:
```
https://cloud.temporal.io/namespaces/scaffold.infra/workflows
```

### Step 2: Create the API key secret

```bash
ct kubectl --context s-saa-cogs create secret generic omes-api-key \
  -n omes --from-literal=api-key='<your-temporal-api-key>'
```

### Step 3: Build and push a custom omes image

Trigger the GitHub Actions workflow on the branch:

```bash
gh workflow run all-docker-images.yml \
  --ref saa-cogs \
  -f go-version=v1.41.1 \
  -f do-push=true
```

This pushes to Docker Hub as `temporaliotest/omes:<omes-commit-sha>-go-1.41.1`.
Check the tag:

```bash
curl -s "https://hub.docker.com/v2/repositories/temporaliotest/omes/tags/?page_size=5&ordering=-last_updated" \
  | python3 -c "import json,sys; d=json.load(sys.stdin); [print(r['name']) for r in d.get('results',[])]"
```

### Step 4: Mirror the image from Docker Hub to ECR

```bash
ecr=450777629615.dkr.ecr.us-west-2.amazonaws.com
tag=27bd42d-go-1.41.1  # your tag from step 3

# Authenticate to ECR (requires infra-01 access)
ct access --account infra-01 -d 8h

aws ecr get-login-password --region us-west-2 --profile infra-01/AWSAdministratorAccess \
  | skopeo login $ecr --username AWS --password-stdin

# Copy from Docker Hub to ECR
skopeo copy --all --insecure-policy \
  docker://docker.io/temporaliotest/omes:$tag \
  docker://$ecr/omes:$tag
```

Note: `docker login` credentials are not used by skopeo; you must use
`skopeo login` separately.

### Step 5: Configure and patch the deployment

Edit `deploy-omes/config.fish`:

```fish
# Configuration for OMES worker and executor scripts
# Edit the values below for your environment

set -x cell "s-saa-cogs"
set -x ns "saa-cogs-4"
set -x runid "run-cell-1"
set -x scenario "workflow_with_single_activity"
set -x omes_image_tag "27bd42d-go-1.41.1"

# ECR registry for OMES images (mirrored from Docker Hub via skopeo)
set -x omes_ecr_registry "450777629615.dkr.ecr.us-west-2.amazonaws.com/omes"

# Auth: "mtls" (uses /certs volume) or "api_key" (uses k8s secret)
set -x auth_method "api_key"

# API gateway endpoint (used with api_key auth)
set -x api_gateway "us-west-2.aws.api.tmprl-test.cloud:7233"
```

Then patch the deployment:

```bash
cd deploy-omes
fish patch-worker.fish
```

### Step 6: Verify

```bash
ct kubectl --context s-saa-cogs get pods -n omes
ct kubectl --context s-saa-cogs logs -n omes -l app.kubernetes.io/instance=omes --tail=5
```

Expected output: pods Running, logs showing `Started Worker` on the correct
task queue and namespace.

## Script reference

### patch-worker.fish

Patches the omes deployment to run Go workers. Supports both API key and mTLS
auth. Sets the container image, command args, and (for API key) injects the
`TEMPORAL_API_KEY` env var from the `omes-api-key` k8s secret. Kubernetes
expands `$(TEMPORAL_API_KEY)` in container args at pod creation time.

```fish
#!/usr/bin/env fish

# Patches the existing OMES deployment to run only Go workers (no scenario executor)
# Usage: patch-worker.fish [replicas]
# Default replicas: 2
# Requires: config.fish with variables defined

set script_dir (dirname (status --current-filename))

if test -f "$script_dir/config.fish"
    source "$script_dir/config.fish"
else
    echo "Error: config.fish not found in $script_dir"
    exit 1
end

for var in cell ns runid scenario auth_method omes_image_tag omes_ecr_registry
    if test -z "$$var"
        echo "Error: \$$var is not set in config.fish"
        exit 1
    end
end

set -l replicas 2
if test (count $argv) -gt 0
    set replicas $argv[1]
end

set -l deployment_name "omes-$ns-temporal-dev-omes-worker"
set -l namespace_fqdn "$ns.temporal-dev"
set -l image "$omes_ecr_registry:$omes_image_tag"

set -l tmpfile (mktemp /tmp/omes-worker-patch.XXXXXX.json)

if test "$auth_method" = "api_key"
    set -l server_address "$api_gateway"
    yq -o=json -n "
      .spec.replicas = $replicas |
      .spec.template.spec.containers[0].name = \"omes\" |
      .spec.template.spec.containers[0].image = \"$image\" |
      .spec.template.spec.containers[0].command = [\"/app/temporal-omes\"] |
      .spec.template.spec.containers[0].args = [
        \"run-worker\",
        \"--language=go\",
        \"--dir-name=prepared\",
        \"--scenario=$scenario\",
        \"--run-id=cicd-go-$runid\",
        \"--namespace=$namespace_fqdn\",
        \"--server-address=$server_address\",
        \"--tls\",
        \"--disable-tls-host-verification\",
        \"--auth-header=Bearer \$(TEMPORAL_API_KEY)\"
      ] |
      .spec.template.spec.containers[0].env = [
        {\"name\": \"TEMPORAL_API_KEY\", \"valueFrom\": {\"secretKeyRef\": {\"name\": \"omes-api-key\", \"key\": \"api-key\"}}}
      ]" > $tmpfile
else
    set -l server_address "$namespace_fqdn.tmprl-test.cloud:7233"
    yq -o=json -n "
      .spec.replicas = $replicas |
      .spec.template.spec.containers[0].name = \"omes\" |
      .spec.template.spec.containers[0].image = \"$image\" |
      .spec.template.spec.containers[0].command = [\"/app/temporal-omes\"] |
      .spec.template.spec.containers[0].args = [
        \"run-worker\",
        \"--language=go\",
        \"--dir-name=prepared\",
        \"--scenario=$scenario\",
        \"--run-id=cicd-go-$runid\",
        \"--namespace=$namespace_fqdn\",
        \"--server-address=$server_address\",
        \"--disable-tls-host-verification\",
        \"--tls\",
        \"--tls-cert-path=/certs/tls.crt\",
        \"--tls-key-path=/certs/tls.key\"
      ]" > $tmpfile
end

echo "Patching deployment '$deployment_name' to run $replicas worker(s):"
echo ""
yq -P $tmpfile
echo ""
echo "Do you want to proceed? (y/n)"
read -l confirm

if test "$confirm" = "y" -o "$confirm" = "Y"
    echo ""
    echo "Patching deployment..."
    omni kubectl --context $cell patch deployment $deployment_name \
        -n omes \
        --type=strategic \
        --patch-file=$tmpfile
    rm -f $tmpfile
else
    rm -f $tmpfile
    echo "Cancelled."
    exit 0
end
```

### run-executor.fish

Creates a k8s Job that runs the scenario executor (starts
workflows/activities). The workers (deployed via `patch-worker.fish`) handle the
actual execution.

```fish
#!/usr/bin/env fish

# Runs the OMES scenario executor as a Kubernetes Job.
# Deletes any previous run, recreates from template, and follows logs.
# Usage: run-executor.fish [duration]
# Default duration: 600s

set script_dir (dirname (status --current-filename))

if test -f "$script_dir/config.fish"
    source "$script_dir/config.fish"
else
    echo "Error: config.fish not found in $script_dir"
    exit 1
end

for var in cell ns runid scenario omes_image_tag omes_ecr_registry auth_method
    if test -z "$$var"
        echo "Error: \$$var is not set in config.fish"
        exit 1
    end
end

set -l duration "600s"
if test (count $argv) -gt 0
    set duration $argv[1]
end

set -l job_name "omes-executor"
set -l namespace_fqdn "$ns.temporal-dev"
set -l image "$omes_ecr_registry:$omes_image_tag"

if test "$auth_method" = "api_key"
    set -l server_address "$api_gateway"
    set -l yq_expr ".spec.template.spec.containers[0].image = \"$image\" |
         .spec.template.spec.containers[0].args = [
           \"run-scenario\",
           \"--scenario=$scenario\",
           \"--run-id=cicd-go-$runid\",
           \"--namespace=$namespace_fqdn\",
           \"--server-address=$server_address\",
           \"--tls\",
           \"--disable-tls-host-verification\",
           \"--auth-header=Bearer \$(TEMPORAL_API_KEY)\",
           \"--duration=$duration\",
           \"--do-not-register-search-attributes\"
         ] |
         .spec.template.spec.containers[0].env = [
           {\"name\": \"TEMPORAL_API_KEY\", \"valueFrom\": {\"secretKeyRef\": {\"name\": \"omes-api-key\", \"key\": \"api-key\"}}}
         ] |
         del(.spec.template.spec.volumes) |
         del(.spec.template.spec.containers[0].volumeMounts)"
else
    set -l deployment_name "omes-$ns-temporal-dev-omes-worker"
    set -l server_address "$namespace_fqdn.tmprl-test.cloud:7233"
    set -l secret_name "$deployment_name"
    set -l yq_expr ".spec.template.spec.containers[0].image = \"$image\" |
         .spec.template.spec.containers[0].args = [
           \"run-scenario\",
           \"--scenario=$scenario\",
           \"--run-id=cicd-go-$runid\",
           \"--namespace=$namespace_fqdn\",
           \"--server-address=$server_address\",
           \"--disable-tls-host-verification\",
           \"--tls\",
           \"--tls-cert-path=/certs/tls.crt\",
           \"--tls-key-path=/certs/tls.key\",
           \"--duration=$duration\",
           \"--do-not-register-search-attributes\"
         ] |
         .spec.template.spec.volumes[0].secret.secretName = \"$secret_name\""
end

# Write rendered yaml to a temp file to preserve formatting
set -l tmpfile (mktemp /tmp/omes-executor.XXXXXX.yaml)
yq eval "$yq_expr" "$script_dir/executor-job.yaml" > $tmpfile

cat $tmpfile
echo ""
echo "Run executor? (y/n)"
read -l confirm

if test "$confirm" = "y" -o "$confirm" = "Y"
    omni kubectl --context $cell delete job $job_name -n omes --wait=true 2>/dev/null
    omni kubectl --context $cell apply -f $tmpfile -n omes
    rm -f $tmpfile
    echo ""
    echo "Job started. Useful commands:"
    echo "  omni kubectl --context $cell logs -f job/$job_name -n omes"
    echo "  omni kubectl --context $cell get job $job_name -n omes"
else
    rm -f $tmpfile
    echo "Cancelled."
    exit 0
end
```

### executor-job.yaml

Template for the executor k8s Job. Fields are overwritten by `run-executor.fish`.

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: omes-executor
  namespace: omes
  labels:
    app.kubernetes.io/name: omes-executor
    app.kubernetes.io/instance: omes
spec:
  backoffLimit: 0
  ttlSecondsAfterFinished: 3600
  template:
    metadata:
      labels:
        app.kubernetes.io/name: omes-executor
        app.kubernetes.io/instance: omes
    spec:
      restartPolicy: Never
      containers:
      - name: omes
        image: placeholder
        imagePullPolicy: IfNotPresent
        command:
        - /app/temporal-omes
        args: []
        ports:
        - containerPort: 9090
          name: metrics
          protocol: TCP
        volumeMounts:
        - mountPath: /certs
          name: certs
      volumes:
      - name: certs
        secret:
          secretName: placeholder
```

## Key details

- **Image registry**: Test cells enforce a Kyverno `restrict-image-registries`
  policy. Only `450777629615.dkr.ecr.us-west-2.amazonaws.com` and
  `612212029444.dkr.ecr.us-west-2.amazonaws.com` are allowed. Docker Hub images
  must be mirrored to ECR via `skopeo`.

- **ECR access**: Requires `ct access --account infra-01` then
  `skopeo login` (not `docker login`) with the ECR credentials.

- **API key in k8s**: Stored as a k8s secret `omes-api-key` in the `omes`
  namespace. Injected into pods via `secretKeyRef`. Kubernetes expands
  `$(TEMPORAL_API_KEY)` in container args.

- **Server address**: API key auth uses the API gateway
  (`us-west-2.aws.api.tmprl-test.cloud:7233`), not the direct cell address
  (`<ns>.tmprl-test.cloud:7233`) which is used with mTLS.

- **Deployment naming**: Scaffold creates the deployment as
  `omes-<ns>-temporal-dev-omes-worker` (e.g.
  `omes-saa-cogs-4-temporal-dev-omes-worker`).

- **Task queue**: The worker polls `omes-cicd-go-<runid>` (prefixed by
  `omes` hardcoded, then `cicd-go-` from the `--run-id` flag).

- **Scaffold workflow tracking**: Scaffold workflows run in the `scaffold.infra`
  namespace, visible at
  `https://cloud.temporal.io/namespaces/scaffold.infra/workflows`.
