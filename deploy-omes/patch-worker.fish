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

for var in cell ns runid scenario
    if test -z "$$var"
        echo "Error: \$$var is not set in config.fish"
        exit 1
    end
end

set -l replicas 2
if test (count $argv) -gt 0
    set replicas $argv[1]
end

set -l deployment_name "omes-$ns-e2e-omes-worker"
set -l namespace_fqdn "$ns.e2e"
set -l server_address "$namespace_fqdn.tmprl-test.cloud:7233"

set -l tmpfile (mktemp /tmp/omes-worker-patch.XXXXXX.json)
yq -o=json -n "
  .spec.replicas = $replicas |
  .spec.template.spec.containers[0].name = \"omes\" |
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
