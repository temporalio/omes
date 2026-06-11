package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
)

// envVarFor maps a flag name to its env-var name. Example: "run-id" -> "OMES_RUN_ID".
func envVarFor(envPrefix, flagName string) string {
	return envPrefix + strings.ToUpper(strings.ReplaceAll(flagName, "-", "_"))
}

// applyEnvFallbacks populates unset flags from environment variables. For a flag
// named "my-flag" the env var is "<envPrefix>MY_FLAG". Flags supplied on the
// command line are not overridden. Setting via fs.Set marks the flag as Changed,
// so the parent's passthrough logic propagates env-var-derived values to the
// prepared worker subprocess.
//
// Used in Lambda mode: the harness CLI's ENTRYPOINT is fixed in the container
// image, so per-invocation values like --run-id come in as Lambda function env
// vars instead of args.
func applyEnvFallbacks(fs *pflag.FlagSet, envPrefix string) error {
	var errs []string
	fs.VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			return
		}
		envName := envVarFor(envPrefix, f.Name)
		val, ok := os.LookupEnv(envName)
		if !ok {
			return
		}
		if err := fs.Set(f.Name, val); err != nil {
			errs = append(errs, fmt.Sprintf("--%s=%q (from %s): %v", f.Name, val, envName, err))
		}
	})
	if len(errs) > 0 {
		return fmt.Errorf("invalid env-var fallback values: %s", strings.Join(errs, "; "))
	}
	return nil
}

// requireFlagsOrEnv ensures each named flag was set, either on the command line
// or via env-var fallback (which applyEnvFallbacks must have already resolved).
func requireFlagsOrEnv(fs *pflag.FlagSet, envPrefix string, names ...string) error {
	var missing []string
	for _, name := range names {
		f := fs.Lookup(name)
		if f == nil {
			return fmt.Errorf("unknown flag --%s", name)
		}
		if !f.Changed {
			missing = append(missing, fmt.Sprintf("--%s (or env %s)", name, envVarFor(envPrefix, name)))
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("required flag(s) not set: %s", strings.Join(missing, ", "))
	}
	return nil
}
