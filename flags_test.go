package commentratio_test

import (
	"bytes"
	"flag"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mocoarow/commentratio"
)

func jsonKeys(typ reflect.Type) []string {
	keys := make([]string, 0, typ.NumField())
	for field := range typ.Fields() {
		if field.Anonymous {
			keys = append(keys, jsonKeys(field.Type)...)

			continue
		}

		keys = append(keys, strings.Split(field.Tag.Get("json"), ",")[0])
	}

	return keys
}

func settingsKeys() []string {
	var keys []string

	for field := range reflect.TypeFor[commentratio.Settings]().Fields() {
		prefix := strings.Split(field.Tag.Get("json"), ",")[0]
		if field.Type.Kind() != reflect.Struct {
			keys = append(keys, prefix)

			continue
		}

		for _, key := range jsonKeys(field.Type) {
			keys = append(keys, prefix+"."+key)
		}
	}

	return keys
}

func Test_NewAnalyzer_shouldRegisterFlag_whenSettingsHasJSONKey(t *testing.T) {
	t.Parallel()

	for _, key := range settingsKeys() {
		t.Run(key, func(t *testing.T) {
			t.Parallel()

			// given
			a := newAnalyzer(t, commentratio.DefaultSettings())

			// when
			got := a.Flags.Lookup(key)

			// then
			assert.NotNil(t, got)
		})
	}
}

func Test_NewAnalyzer_shouldNotRegisterRequireFromFlag_whenRuleIsNotDocRule(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"func-body.require-from", "file.require-from"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// given
			a := newAnalyzer(t, commentratio.DefaultSettings())

			// when
			got := a.Flags.Lookup(name)

			// then
			assert.Nil(t, got)
		})
	}
}

func Test_NewAnalyzer_shouldAppendIgnorePattern_whenFlagIsRepeated(t *testing.T) {
	t.Parallel()

	// given
	s := commentratio.DefaultSettings()
	s.IgnorePatterns = []string{"^given$"}
	a := newAnalyzer(t, s)

	// when
	err := a.Flags.Parse([]string{"-ignore-patterns", "^when$", "-ignore-patterns", "^then$"})

	// then
	require.NoError(t, err)
	assert.Equal(t, "^given$,^when$,^then$", a.Flags.Lookup("ignore-patterns").Value.String())
}

func Test_NewAnalyzer_shouldPrintFlags_whenHelpIsRequested(t *testing.T) {
	t.Parallel()

	// given
	a := newAnalyzer(t, commentratio.DefaultSettings())

	var out bytes.Buffer

	a.Flags.SetOutput(&out)

	// when
	err := a.Flags.Parse([]string{"-h"})

	// then
	require.ErrorIs(t, err, flag.ErrHelp)
	assert.Contains(t, out.String(), "func-doc.max-lines")
}
