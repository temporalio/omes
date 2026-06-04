package workers

import (
	"slices"
	"strings"
)

func withEnv(environ []string, name string, value string) []string {
	prefix := name + "="
	nextEnv := slices.DeleteFunc(slices.Clone(environ), func(item string) bool {
		return strings.HasPrefix(item, prefix)
	})
	if value != "" {
		nextEnv = append(nextEnv, prefix+value)
	}
	return nextEnv
}
