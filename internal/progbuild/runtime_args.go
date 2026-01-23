package progbuild

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/temporalio/omes/cmd/clioptions"
)

// BuildRuntimeArgs returns the initial runtime args for a program based on language.
// For Python: returns module name (with dashes replaced by underscores)
// For TypeScript: returns path to compiled entry point
// For Go: returns empty slice (binary takes subcommand directly)
func BuildRuntimeArgs(language clioptions.Language, projectDir string) ([]string, error) {
	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve project directory: %w", err)
	}
	projectName := filepath.Base(absProjectDir)

	switch language {
	case clioptions.LangPython:
		moduleName := strings.ReplaceAll(projectName, "-", "_")
		return []string{moduleName}, nil
	case clioptions.LangTypeScript:
		return []string{fmt.Sprintf("tslib/tests/%s/main.js", projectName)}, nil
	case clioptions.LangGo:
		return []string{}, nil
	default:
		return nil, fmt.Errorf("unsupported language for runtime args: %s", language)
	}
}
