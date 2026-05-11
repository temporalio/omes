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
