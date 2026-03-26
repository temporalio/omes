# Configuration for OMES worker and executor scripts
# Edit the values below for your environment

set -x cell "s-saa-cogs"
set -x ns "saa-cogs-4"
set -x runid "run-cell-1"
set -x scenario "standalone_activity"
set -x omes_image_tag "27bd42d-go-1.41.1"

# ECR registry for OMES images (mirrored from Docker Hub via skopeo)
set -x omes_ecr_registry "450777629615.dkr.ecr.us-west-2.amazonaws.com/omes"

# Auth: "mtls" (uses /certs volume) or "api_key" (uses k8s secret)
set -x auth_method "api_key"

# API gateway endpoint (used with api_key auth)
set -x api_gateway "us-west-2.aws.api.tmprl-test.cloud:7233"

# Worker tuning (passed to run-worker with worker- prefix)
set -x max_concurrent_activity_pollers 40
set -x max_concurrent_activities 500

# Executor load parameters
set -x duration "1h"
set -x max_concurrent 500
set -x max_iterations_per_second 500
set -x scenario_options "--option payload-size=102400"
