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

for var in cell ns runid scenario omes_image_tag omes_ecr_registry
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
set -l deployment_name "omes-$ns-e2e-omes-worker"
set -l namespace_fqdn "$ns.e2e"
set -l server_address "$namespace_fqdn.tmprl-test.cloud:7233"
set -l image "$omes_ecr_registry:$omes_image_tag"
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
