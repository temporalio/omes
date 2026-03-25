#!/usr/bin/env fish

# Deletes the OMES executor job
# Usage: delete-executor.fish

set script_dir (dirname (status --current-filename))

if test -f "$script_dir/config.fish"
    source "$script_dir/config.fish"
else
    echo "Error: config.fish not found in $script_dir"
    exit 1
end

if test -z "$cell"
    echo "Error: \$cell is not set in config.fish"
    exit 1
end

omni kubectl --context $cell delete job omes-executor -n omes
