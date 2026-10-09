package plugin_test

import (
	"testing"

	"github.com/golangci/plugin-module-register/register"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis"

	"github.com/mocoarow/commentratio"
	"github.com/mocoarow/commentratio/plugin"
)

func analyzerNames(analyzers []*analysis.Analyzer) []string {
	names := make([]string, 0, len(analyzers))
	for _, a := range analyzers {
		names = append(names, a.Name)
	}

	return names
}

func Test_init_shouldRegisterPlugin_whenPackageIsImported(t *testing.T) {
	t.Parallel()

	// given
	name := commentratio.Name

	// when
	_, err := register.GetPlugin(name)

	// then
	require.NoError(t, err)
}

func Test_New_shouldReturnPlugin_whenSettingsAreValid(t *testing.T) {
	t.Parallel()

	// given
	raw := map[string]any{"func-doc": map[string]any{"max-lines": 7}}

	// when
	p, err := plugin.New(raw)

	// then
	require.NoError(t, err)
	assert.NotNil(t, p)
}

func Test_New_shouldReturnErrInvalidSettings_whenSettingsAreInvalid(t *testing.T) {
	t.Parallel()

	// given
	raw := map[string]any{"func-doc": map[string]any{"free-lines": -1}}

	// when
	_, err := plugin.New(raw)

	// then
	require.ErrorIs(t, err, commentratio.ErrInvalidSettings)
}

func Test_linterPlugin_BuildAnalyzers_shouldReturnCommentratioAnalyzer_whenCalled(t *testing.T) {
	t.Parallel()

	// given
	p, err := plugin.New(nil)
	require.NoError(t, err)

	// when
	analyzers, err := p.BuildAnalyzers()

	// then
	require.NoError(t, err)
	assert.Equal(t, []string{commentratio.Name}, analyzerNames(analyzers))
}

func Test_linterPlugin_BuildAnalyzers_shouldApplySettings_whenSettingsAreGiven(t *testing.T) {
	t.Parallel()

	// given
	p, err := plugin.New(map[string]any{"func-doc": map[string]any{"max-lines": 7}})
	require.NoError(t, err)

	// when
	analyzers, err := p.BuildAnalyzers()

	// then
	require.NoError(t, err)
	require.Len(t, analyzers, 1)

	maxLines := analyzers[0].Flags.Lookup("func-doc.max-lines")
	require.NotNil(t, maxLines)
	assert.Equal(t, "7", maxLines.Value.String())
}

func Test_linterPlugin_GetLoadMode_shouldReturnSyntax_whenCalled(t *testing.T) {
	t.Parallel()

	// given
	p, err := plugin.New(nil)
	require.NoError(t, err)

	// when
	got := p.GetLoadMode()

	// then
	assert.Equal(t, register.LoadModeSyntax, got)
}
