#!/usr/bin/env fish

# Runs the OMES scenario executor as a Kubernetes Job.
# Deletes any previous run, recreates from template, and follows logs.

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

set -l job_name "omes-executor"
set -l namespace_fqdn "$ns.temporal-dev"
set -l image "$omes_ecr_registry:$omes_image_tag"
set -l tmpfile (mktemp /tmp/omes-executor.XXXXXX.yaml)

if test "$auth_method" = "api_key"
    set -l server_address "$api_gateway"
    yq eval "
      .spec.template.spec.containers[0].image = \"$image\" |
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
        \"--max-concurrent=$max_concurrent\",
        \"--max-iterations-per-second=$max_iterations_per_second\",
        \"--option\", \"payload-size=102400\",
        \"--do-not-register-search-attributes\"
      ] |
      .spec.template.spec.containers[0].env = [
        {\"name\": \"TEMPORAL_API_KEY\", \"valueFrom\": {\"secretKeyRef\": {\"name\": \"omes-api-key\", \"key\": \"api-key\"}}}
      ] |
      del(.spec.template.spec.volumes) |
      del(.spec.template.spec.containers[0].volumeMounts)
    " "$script_dir/executor-job.yaml" > $tmpfile
else
    set -l deployment_name "omes-$ns-temporal-dev-omes-worker"
    set -l server_address "$namespace_fqdn.tmprl-test.cloud:7233"
    set -l secret_name "$deployment_name"
    yq eval "
      .spec.template.spec.containers[0].image = \"$image\" |
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
        \"--max-concurrent=$max_concurrent\",
        \"--max-iterations-per-second=$max_iterations_per_second\",
        \"--option\", \"payload-size=102400\",
        \"--do-not-register-search-attributes\"
      ] |
      .spec.template.spec.volumes[0].secret.secretName = \"$secret_name\"
    " "$script_dir/executor-job.yaml" > $tmpfile
end

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
    echo "  ct kubectl --context $cell logs -f job/$job_name -n omes"
    echo "  ct kubectl --context $cell get job $job_name -n omes"
else
    rm -f $tmpfile
    echo "Cancelled."
    exit 0
end
