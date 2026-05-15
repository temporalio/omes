package clioptions

import (
	"os"
	"strings"

	"github.com/spf13/pflag"
)

// EnvVarPrefix is the prefix applied to all environment variable names.
const EnvVarPrefix = "TEMPORAL_OMES_"

// FlagNameToEnvVar converts a flag name to its corresponding environment variable name.
// For example, "server-address" becomes "TEMPORAL_OMES_SERVER_ADDRESS".
func FlagNameToEnvVar(flagName string) string {
	return EnvVarPrefix + strings.ToUpper(strings.ReplaceAll(flagName, "-", "_"))
}

// BindEnvVars sets flag values from environment variables for any flags not explicitly
// set on the command line. CLI flags always take precedence over environment variables.
// Environment variable names are derived from flag names via FlagNameToEnvVar.
func BindEnvVars(fs *pflag.FlagSet) {
	fs.VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			return
		}
		if val, ok := os.LookupEnv(FlagNameToEnvVar(f.Name)); ok {
			_ = f.Value.Set(val)
		}
	})
}

// WithEnv removes any existing environment variable matching name and appends
// name=value when value is not empty.
func WithEnv(environ []string, name string, value string) []string {
	prefix := name + "="
	nextEnv := make([]string, 0, len(environ)+1)
	for _, item := range environ {
		if strings.HasPrefix(item, prefix) {
			continue
		}
		nextEnv = append(nextEnv, item)
	}
	if value != "" {
		nextEnv = append(nextEnv, prefix+value)
	}
	return nextEnv
}
