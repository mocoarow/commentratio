package plugin

import (
	"fmt"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"

	"github.com/mocoarow/commentratio"
)

// golangci-lint module plugins can only be registered from init, so this package is an exception to the no-init rule.
func init() {
	register.Plugin(commentratio.Name, New)
}

type linterPlugin struct {
	settings commentratio.Settings
}

// New decodes the plugin settings given in .golangci.yml.
func New(settings any) (register.LinterPlugin, error) { //nolint:ireturn // register.NewPlugin requires this signature
	s, err := commentratio.DecodeSettings(settings)
	if err != nil {
		return nil, fmt.Errorf("decode settings: %w", err)
	}

	return linterPlugin{settings: s}, nil
}

func (p linterPlugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	a, err := commentratio.NewAnalyzer(p.settings)
	if err != nil {
		return nil, fmt.Errorf("build analyzer: %w", err)
	}

	return []*analysis.Analyzer{a}, nil
}

// GetLoadMode returns the syntax load mode, since commentratio needs no type information.
func (p linterPlugin) GetLoadMode() string {
	return register.LoadModeSyntax
}
