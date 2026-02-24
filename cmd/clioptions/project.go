package clioptions

import (
	"github.com/spf13/pflag"
)

type ProgramOptions struct {
	// Path to user's test program directory
	ProgramDir string
	// Directory for SDK build output
	BuildDir string
}

func (p *ProgramOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&p.ProgramDir, "project-dir", "", "Path to test project directory (required)")
	fs.StringVar(&p.BuildDir, "build-dir", "", "Directory for SDK build output (optional)")
}
