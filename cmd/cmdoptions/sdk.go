package cmdoptions

import (
	"fmt"

	"github.com/spf13/pflag"
)

type Language string

const (
	LangGo         Language = "go"
	LangPython     Language = "python"
	LangJava       Language = "java"
	LangTypeScript Language = "typescript"
	LangDotNet     Language = "dotnet"
)

func (lf *Language) String() string {
	return string(*lf)
}

func (lf *Language) Set(value string) error {
	switch value {
	case string(LangGo):
		*lf = LangGo
	case string(LangPython), "py":
		*lf = LangPython
	case string(LangJava):
		*lf = LangJava
	case string(LangTypeScript), "ts":
		*lf = LangTypeScript
	case string(LangDotNet), "cs":
		*lf = LangDotNet
	default:
		return fmt.Errorf("invalid language %q, must be one of: [go, python, java, typescript, dotnet] or aliases [py, ts, cs]", value)
	}
	return nil
}

func (lf *Language) Type() string {
	return "language"
}

type SdkOptions struct {
	Language Language
	Version  string
}

func (s *SdkOptions) AddCLIFlags(fs *pflag.FlagSet) {
	fs.Var(&s.Language, "language", "Language to use (go, python, ts, cs, java)")
	fs.StringVar(&s.Version, "version", "", "SDK version to use - treated as path if slash present")
}
