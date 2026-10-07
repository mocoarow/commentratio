package commentratio_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mocoarow/commentratio"
)

func Test_DefaultSettings_shouldReturnDefaults_whenCalled(t *testing.T) {
	t.Parallel()

	// given
	want := commentratio.Settings{
		FuncDoc: commentratio.DocRule{
			Rule:        commentratio.Rule{Enabled: true, FreeLines: 3, MaxRatio: 0.2, MaxLines: 15},
			RequireFrom: 10,
		},
		DeclDoc: commentratio.DocRule{
			Rule:        commentratio.Rule{Enabled: true, FreeLines: 3, MaxRatio: 0.2, MaxLines: 15},
			RequireFrom: 10,
		},
		FuncBody: commentratio.Rule{Enabled: true, FreeLines: 2, MaxRatio: 0.2, MaxLines: 10},
		File:     commentratio.Rule{Enabled: true, FreeLines: 5, MaxRatio: 0.3, MaxLines: 200},
	}

	// when
	got := commentratio.DefaultSettings()

	// then
	assert.Equal(t, want, got)
}

func Test_Settings_Validate_shouldReturnNil_whenSettingsAreValid(t *testing.T) {
	t.Parallel()

	zero := commentratio.Settings{}
	freeEqualsMax := commentratio.DefaultSettings()
	freeEqualsMax.FuncDoc.FreeLines = freeEqualsMax.FuncDoc.MaxLines
	ratioZero := commentratio.DefaultSettings()
	ratioZero.File.MaxRatio = 0

	tests := []struct {
		name     string
		settings commentratio.Settings
	}{
		{name: "defaults", settings: commentratio.DefaultSettings()},
		{name: "all zero", settings: zero},
		{name: "free-lines equals max-lines", settings: freeEqualsMax},
		{name: "max-ratio is zero", settings: ratioZero},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// given
			s := tt.settings

			// when
			err := s.Validate()

			// then
			require.NoError(t, err)
		})
	}
}

type ruleTarget struct {
	name string
	rule func(s *commentratio.Settings) *commentratio.Rule
}

func ruleTargets() []ruleTarget {
	return []ruleTarget{
		{name: "func-doc", rule: func(s *commentratio.Settings) *commentratio.Rule { return &s.FuncDoc.Rule }},
		{name: "decl-doc", rule: func(s *commentratio.Settings) *commentratio.Rule { return &s.DeclDoc.Rule }},
		{name: "func-body", rule: func(s *commentratio.Settings) *commentratio.Rule { return &s.FuncBody }},
		{name: "file", rule: func(s *commentratio.Settings) *commentratio.Rule { return &s.File }},
	}
}

type invalidCase struct {
	name     string
	settings commentratio.Settings
	// wantMessage is a substring of the error message that identifies the invalid value.
	wantMessage string
}

func invalidSettingsCases() []invalidCase {
	// wantFormat takes the target name as %[1]s.
	mutations := []struct {
		name       string
		mutate     func(r *commentratio.Rule)
		wantFormat string
	}{
		{name: "negative free-lines", mutate: func(r *commentratio.Rule) { r.FreeLines = -1 }, wantFormat: "%[1]s.free-lines must be >= 0"},
		{name: "negative max-lines", mutate: func(r *commentratio.Rule) { r.FreeLines, r.MaxLines = 0, -1 }, wantFormat: "%[1]s.max-lines must be >= 0"},
		{name: "negative max-ratio", mutate: func(r *commentratio.Rule) { r.MaxRatio = -0.1 }, wantFormat: "%[1]s.max-ratio must be"},
		{name: "NaN max-ratio", mutate: func(r *commentratio.Rule) { r.MaxRatio = math.NaN() }, wantFormat: "%[1]s.max-ratio must be"},
		{name: "infinite max-ratio", mutate: func(r *commentratio.Rule) { r.MaxRatio = math.Inf(1) }, wantFormat: "%[1]s.max-ratio must be"},
		{name: "free-lines greater than max-lines", mutate: func(r *commentratio.Rule) { r.FreeLines, r.MaxLines = 5, 4 }, wantFormat: "%[1]s.free-lines (5) must be <= %[1]s.max-lines (4)"},
	}

	targets := ruleTargets()
	cases := make([]invalidCase, 0, len(targets)*len(mutations)+2)
	for _, target := range targets {
		for _, m := range mutations {
			s := commentratio.DefaultSettings()
			m.mutate(target.rule(&s))
			cases = append(cases, invalidCase{
				name:        target.name + " " + m.name,
				settings:    s,
				wantMessage: fmt.Sprintf(m.wantFormat, target.name),
			})
		}
	}

	funcDoc := commentratio.DefaultSettings()
	funcDoc.FuncDoc.RequireFrom = -1
	declDoc := commentratio.DefaultSettings()
	declDoc.DeclDoc.RequireFrom = -1

	return append(cases,
		invalidCase{name: "func-doc negative require-from", settings: funcDoc, wantMessage: "func-doc.require-from must be >= 0"},
		invalidCase{name: "decl-doc negative require-from", settings: declDoc, wantMessage: "decl-doc.require-from must be >= 0"},
	)
}

func Test_Settings_Validate_shouldReturnErrInvalidSettings_whenValueIsInvalid(t *testing.T) {
	t.Parallel()

	for _, tt := range invalidSettingsCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// given
			s := tt.settings

			// when
			err := s.Validate()

			// then
			require.ErrorIs(t, err, commentratio.ErrInvalidSettings)
		})
	}
}

func Test_Settings_Validate_shouldNameInvalidKey_whenValueIsInvalid(t *testing.T) {
	t.Parallel()

	for _, tt := range invalidSettingsCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// given
			s := tt.settings

			// when
			err := s.Validate()

			// then
			require.ErrorContains(t, err, tt.wantMessage)
		})
	}
}

func Test_Settings_Validate_shouldReturnErrInvalidSettings_whenDisabledRuleIsInvalid(t *testing.T) {
	t.Parallel()

	// given
	s := commentratio.DefaultSettings()
	s.File.Enabled = false
	s.File.FreeLines = -1

	// when
	err := s.Validate()

	// then
	require.ErrorIs(t, err, commentratio.ErrInvalidSettings)
}

func Test_DecodeSettings_shouldReturnDefaults_whenRawIsNil(t *testing.T) {
	t.Parallel()

	// given
	var raw any

	// when
	got, err := commentratio.DecodeSettings(raw)

	// then
	require.NoError(t, err)
	assert.Equal(t, commentratio.DefaultSettings(), got)
}

func Test_DecodeSettings_shouldKeepDefaults_whenKeyIsOmitted(t *testing.T) {
	t.Parallel()

	// given
	raw := map[string]any{"func-doc": map[string]any{"max-ratio": 0.45}}
	want := commentratio.DefaultSettings()
	want.FuncDoc.MaxRatio = 0.45

	// when
	got, err := commentratio.DecodeSettings(raw)

	// then
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func Test_DecodeSettings_shouldKeepDefaults_whenValueIsNull(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  map[string]any
	}{
		{name: "rule is null", raw: map[string]any{"func-doc": nil}},
		{name: "key is null", raw: map[string]any{"func-doc": map[string]any{"max-lines": nil}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// given
			raw := tt.raw

			// when
			got, err := commentratio.DecodeSettings(raw)

			// then
			require.NoError(t, err)
			assert.Equal(t, commentratio.DefaultSettings(), got)
		})
	}
}

func Test_DecodeSettings_shouldApplyValue_whenKeyCaseDiffers(t *testing.T) {
	t.Parallel()

	// given
	raw := map[string]any{"FUNC-DOC": map[string]any{"Max-Lines": 21}}
	want := commentratio.DefaultSettings()
	want.FuncDoc.MaxLines = 21

	// when
	got, err := commentratio.DecodeSettings(raw)

	// then
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func Test_DecodeSettings_shouldDisableRule_whenEnabledIsFalse(t *testing.T) {
	t.Parallel()

	// given
	raw := map[string]any{"func-body": map[string]any{"enabled": false}}

	// when
	got, err := commentratio.DecodeSettings(raw)

	// then
	require.NoError(t, err)
	assert.False(t, got.FuncBody.Enabled)
}

func Test_DecodeSettings_shouldApplyValue_whenKeyIsSet(t *testing.T) {
	t.Parallel()

	// given
	raw := map[string]any{
		"func-doc":  map[string]any{"free-lines": 7, "max-ratio": 0.45, "max-lines": 21, "require-from": 13},
		"decl-doc":  map[string]any{"free-lines": 6, "max-ratio": 0.35, "max-lines": 22, "require-from": 14},
		"func-body": map[string]any{"free-lines": 4, "max-ratio": 0.15, "max-lines": 9},
		"file":      map[string]any{"free-lines": 8, "max-ratio": 0.55, "max-lines": 123},
	}
	want := commentratio.Settings{
		FuncDoc: commentratio.DocRule{
			Rule:        commentratio.Rule{Enabled: true, FreeLines: 7, MaxRatio: 0.45, MaxLines: 21},
			RequireFrom: 13,
		},
		DeclDoc: commentratio.DocRule{
			Rule:        commentratio.Rule{Enabled: true, FreeLines: 6, MaxRatio: 0.35, MaxLines: 22},
			RequireFrom: 14,
		},
		FuncBody: commentratio.Rule{Enabled: true, FreeLines: 4, MaxRatio: 0.15, MaxLines: 9},
		File:     commentratio.Rule{Enabled: true, FreeLines: 8, MaxRatio: 0.55, MaxLines: 123},
	}

	// when
	got, err := commentratio.DecodeSettings(raw)

	// then
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func Test_DecodeSettings_shouldReturnErrInvalidSettings_whenRequireFromIsSetOnNonDocRule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
	}{
		{name: "func-body", key: "func-body"},
		{name: "file", key: "file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// given
			raw := map[string]any{tt.key: map[string]any{"require-from": 5}}

			// when
			_, err := commentratio.DecodeSettings(raw)

			// then
			require.ErrorIs(t, err, commentratio.ErrInvalidSettings)
		})
	}
}

func Test_DecodeSettings_shouldReturnErrInvalidSettings_whenKeyIsUnknown(t *testing.T) {
	t.Parallel()

	// given
	raw := map[string]any{"func-docs": map[string]any{"max-lines": 5}}

	// when
	_, err := commentratio.DecodeSettings(raw)

	// then
	require.ErrorIs(t, err, commentratio.ErrInvalidSettings)
}

func Test_DecodeSettings_shouldReturnErrInvalidSettings_whenTypeMismatches(t *testing.T) {
	t.Parallel()

	// given
	raw := map[string]any{"func-doc": map[string]any{"max-lines": "many"}}

	// when
	_, err := commentratio.DecodeSettings(raw)

	// then
	require.ErrorIs(t, err, commentratio.ErrInvalidSettings)
}

func Test_DecodeSettings_shouldReturnErrInvalidSettings_whenValueIsOutOfRange(t *testing.T) {
	t.Parallel()

	// given
	raw := map[string]any{"file": map[string]any{"free-lines": -1}}

	// when
	_, err := commentratio.DecodeSettings(raw)

	// then
	require.ErrorIs(t, err, commentratio.ErrInvalidSettings)
}

func Test_DecodeSettings_shouldReturnErrInvalidSettings_whenRawIsNotJSONEncodable(t *testing.T) {
	t.Parallel()

	// given
	raw := map[string]any{"func-doc": func() {}}

	// when
	_, err := commentratio.DecodeSettings(raw)

	// then
	require.ErrorIs(t, err, commentratio.ErrInvalidSettings)
}
