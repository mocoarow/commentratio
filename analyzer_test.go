package commentratio_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/mocoarow/commentratio"
)

func disabledSettings(base commentratio.Settings) commentratio.Settings {
	base.FuncDoc.Enabled = false
	base.DeclDoc.Enabled = false
	base.FuncBody.Enabled = false
	base.File.Enabled = false

	return base
}

func testLimitSettings() commentratio.Settings {
	rule := commentratio.Rule{Enabled: true, FreeLines: 2, MaxRatio: 0.5, MaxLines: 4}

	return commentratio.Settings{
		FuncDoc:  commentratio.DocRule{Rule: rule, RequireFrom: 5},
		DeclDoc:  commentratio.DocRule{Rule: rule, RequireFrom: 5},
		FuncBody: rule,
		File:     rule,
	}
}

func enableFuncDoc(s *commentratio.Settings)  { s.FuncDoc.Enabled = true }
func enableDeclDoc(s *commentratio.Settings)  { s.DeclDoc.Enabled = true }
func enableFuncBody(s *commentratio.Settings) { s.FuncBody.Enabled = true }
func enableFile(s *commentratio.Settings)     { s.File.Enabled = true }

func only(base commentratio.Settings, enable func(s *commentratio.Settings)) commentratio.Settings {
	s := disabledSettings(base)
	enable(&s)

	return s
}

func newAnalyzer(t *testing.T, s commentratio.Settings) *analysis.Analyzer {
	t.Helper()

	a, err := commentratio.NewAnalyzer(s)
	require.NoError(t, err)

	return a
}

func runAnalyzer(t *testing.T, s commentratio.Settings, pkg string) {
	t.Helper()

	analysistest.Run(t, analysistest.TestData(), newAnalyzer(t, s), pkg)
}

func Test_NewAnalyzer_shouldReturnErrInvalidSettings_whenSettingsAreInvalid(t *testing.T) {
	t.Parallel()

	// given
	s := commentratio.DefaultSettings()
	s.FuncDoc.FreeLines = -1

	// when
	_, err := commentratio.NewAnalyzer(s)

	// then
	require.ErrorIs(t, err, commentratio.ErrInvalidSettings)
}

func Test_Analyzer_Run_shouldReturnErrInvalidSettings_whenFlagValueIsInvalid(t *testing.T) {
	t.Parallel()

	// given
	a := newAnalyzer(t, commentratio.DefaultSettings())
	require.NoError(t, a.Flags.Set("func-doc.free-lines", "-1"))

	// when
	_, err := a.Run(&analysis.Pass{})

	// then
	require.ErrorIs(t, err, commentratio.ErrInvalidSettings)
}

func Test_NewAnalyzer_shouldNotReportMissingFuncDoc_whenFuncIsTestFunc(t *testing.T) {
	t.Parallel()

	// given
	s := only(testLimitSettings(), enableFuncDoc)

	// when, then
	runAnalyzer(t, s, "testfuncs")
}

func Test_NewAnalyzer_shouldReportMissingFuncDoc_whenFuncOnlyLooksLikeTestFunc(t *testing.T) {
	t.Parallel()

	// given
	s := only(testLimitSettings(), enableFuncDoc)

	// when, then
	runAnalyzer(t, s, "testfuncslookalike")
}

func Test_NewAnalyzer_shouldReportMissingFuncDoc_whenTestFuncIsInNonTestFile(t *testing.T) {
	t.Parallel()

	// given
	s := only(testLimitSettings(), enableFuncDoc)

	// when, then
	runAnalyzer(t, s, "testfuncsnontest")
}

func Test_NewAnalyzer_shouldNotReportFuncDoc_whenDocIsWithinLimit(t *testing.T) {
	t.Parallel()

	// given
	s := only(testLimitSettings(), enableFuncDoc)

	// when, then
	runAnalyzer(t, s, "funcdocok")
}

func Test_NewAnalyzer_shouldReportFuncDoc_whenDocExceedsLimit(t *testing.T) {
	t.Parallel()

	// given
	s := only(testLimitSettings(), enableFuncDoc)

	// when, then
	runAnalyzer(t, s, "funcdocng")
}

func Test_NewAnalyzer_shouldReportMissingFuncDoc_whenExportedFuncReachesRequireFrom(t *testing.T) {
	t.Parallel()

	// given
	s := only(testLimitSettings(), enableFuncDoc)

	// when, then
	runAnalyzer(t, s, "funcdocmissing")
}

func Test_NewAnalyzer_shouldNotReportMissingFuncDoc_whenNotRequired(t *testing.T) {
	t.Parallel()

	// given
	s := only(testLimitSettings(), enableFuncDoc)

	// when, then
	runAnalyzer(t, s, "funcdocnotmissing")
}

func Test_NewAnalyzer_shouldNotReportDeclDoc_whenDocIsWithinLimit(t *testing.T) {
	t.Parallel()

	// given
	s := only(testLimitSettings(), enableDeclDoc)

	// when, then
	runAnalyzer(t, s, "decldocok")
}

func Test_NewAnalyzer_shouldReportDeclDoc_whenDocExceedsLimit(t *testing.T) {
	t.Parallel()

	// given
	s := only(testLimitSettings(), enableDeclDoc)

	// when, then
	runAnalyzer(t, s, "decldocng")
}

func Test_NewAnalyzer_shouldReportMissingDeclDoc_whenExportedSpecReachesRequireFrom(t *testing.T) {
	t.Parallel()

	// given
	s := only(testLimitSettings(), enableDeclDoc)

	// when, then
	runAnalyzer(t, s, "decldocmissing")
}

func Test_NewAnalyzer_shouldNotReportMissingDeclDoc_whenGroupOrSpecHasDoc(t *testing.T) {
	t.Parallel()

	// given
	s := only(testLimitSettings(), enableDeclDoc)

	// when, then
	runAnalyzer(t, s, "decldocnotmissing")
}

func Test_NewAnalyzer_shouldNotReportFuncBody_whenCommentsAreWithinLimit(t *testing.T) {
	t.Parallel()

	// given
	s := only(testLimitSettings(), enableFuncBody)

	// when, then
	runAnalyzer(t, s, "funcbodyok")
}

func Test_NewAnalyzer_shouldReportFuncBody_whenCommentsExceedLimit(t *testing.T) {
	t.Parallel()

	// given
	s := only(testLimitSettings(), enableFuncBody)

	// when, then
	runAnalyzer(t, s, "funcbodyng")
}

func Test_NewAnalyzer_shouldNotReportFile_whenOnlyHeaderAndPackageDocAreLong(t *testing.T) {
	t.Parallel()

	// given
	s := only(testLimitSettings(), enableFile)

	// when, then
	runAnalyzer(t, s, "fileok")
}

func Test_NewAnalyzer_shouldReportFile_whenCommentsExceedLimit(t *testing.T) {
	t.Parallel()

	// given
	s := only(testLimitSettings(), enableFile)

	// when, then
	runAnalyzer(t, s, "fileng")
}

func Test_NewAnalyzer_shouldSkipFile_whenFileIsGenerated(t *testing.T) {
	t.Parallel()

	// given
	s := only(testLimitSettings(), enableFuncDoc)

	// when, then
	runAnalyzer(t, s, "generated")
}

func Test_NewAnalyzer_shouldCountPhysicalLines_whenLineDirectiveIsPresent(t *testing.T) {
	t.Parallel()

	// given
	s := only(testLimitSettings(), enableFuncDoc)

	// when, then
	runAnalyzer(t, s, "linedirective")
}

func Test_NewAnalyzer_shouldCheckFile_whenFileIsTestFile(t *testing.T) {
	t.Parallel()

	// given
	s := only(testLimitSettings(), enableFuncDoc)

	// when, then
	runAnalyzer(t, s, "testfile")
}

func Test_NewAnalyzer_shouldNotReport_whenRuleIsDisabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		pkg  string
	}{
		{name: "func-doc", pkg: "disabledfuncdoc"},
		{name: "decl-doc", pkg: "disableddecldoc"},
		{name: "func-body", pkg: "disabledfuncbody"},
		{name: "file", pkg: "disabledfile"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// given
			s := disabledSettings(testLimitSettings())

			// when, then
			runAnalyzer(t, s, tt.pkg)
		})
	}
}

func Test_NewAnalyzer_shouldApplyFlagValue_whenFlagIsSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		flag   string
		value  string
		enable func(s *commentratio.Settings)
		pkg    string
	}{
		{flag: "func-doc.max-lines", value: "2", enable: enableFuncDoc, pkg: "flagapplied"},
		{flag: "func-doc.max-ratio", value: "0.1", enable: enableFuncDoc, pkg: "flagapplied"},
		{flag: "func-doc.require-from", value: "4", enable: enableFuncDoc, pkg: "flagrequirefrom"},
		{flag: "func-doc.enabled", value: "false", enable: enableFuncDoc, pkg: "disabledfuncdoc"},
		{flag: "decl-doc.enabled", value: "false", enable: enableDeclDoc, pkg: "disableddecldoc"},
		{flag: "func-body.enabled", value: "false", enable: enableFuncBody, pkg: "disabledfuncbody"},
		{flag: "file.enabled", value: "false", enable: enableFile, pkg: "disabledfile"},
	}
	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			t.Parallel()

			// given
			a := newAnalyzer(t, only(testLimitSettings(), tt.enable))
			require.NoError(t, a.Flags.Set(tt.flag, tt.value))

			// when, then
			analysistest.Run(t, analysistest.TestData(), a, tt.pkg)
		})
	}
}

func Test_NewAnalyzer_shouldReportFuncBody_whenLimitIsZero(t *testing.T) {
	t.Parallel()

	// given
	s := only(testLimitSettings(), enableFuncBody)
	s.FuncBody = commentratio.Rule{Enabled: true, FreeLines: 0, MaxRatio: 0, MaxLines: 0}

	// when, then
	runAnalyzer(t, s, "funcbodyzero")
}

func Test_NewAnalyzer_shouldReportSpecMessage_whenDefaultLimitsApply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		pkg    string
		enable func(s *commentratio.Settings)
	}{
		{name: "func-doc too long", pkg: "specfuncdoc", enable: enableFuncDoc},
		{name: "decl-doc too long", pkg: "specdecldoc", enable: enableDeclDoc},
		{name: "func-doc missing", pkg: "specmissing", enable: enableFuncDoc},
		{name: "func-body too many", pkg: "specbody", enable: enableFuncBody},
		{name: "file too many", pkg: "specfile", enable: enableFile},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// given
			s := only(commentratio.DefaultSettings(), tt.enable)

			// when, then
			runAnalyzer(t, s, tt.pkg)
		})
	}
}
