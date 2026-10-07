package linecount_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mocoarow/commentratio/internal/linecount"
)

func classify(t *testing.T, src string) linecount.Lines {
	t.Helper()

	lines, err := linecount.Classify("a.go", []byte(src))
	require.NoError(t, err)

	return lines
}

func srcWithLine3(line string) string {
	return "package a\n\n" + line + "\nvar x = 1\n"
}

func Test_Classify_shouldCountCodeLine_whenLineHasToken(t *testing.T) {
	t.Parallel()

	// given
	src := "package a\n\nvar x = 1\n"

	// when
	lines := classify(t, src)

	// then
	assert.Equal(t, 1, lines.Code(3, 3))
}

func Test_Classify_shouldNotCountLine_whenLineIsBlank(t *testing.T) {
	t.Parallel()

	// given
	src := "package a\n\nvar x = 1\n"

	// when
	lines := classify(t, src)

	// then
	assert.Equal(t, 0, lines.Code(2, 2)+lines.Comment(2, 2))
}

func Test_Classify_shouldCountCommentLine_whenCommentHasContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  string
		line int
	}{
		{name: "line comment", src: srcWithLine3("// text"), line: 3},
		{name: "block comment", src: srcWithLine3("/* text */"), line: 3},
		{name: "license header", src: "// Copyright 2026 mocoarow\n\npackage a\n", line: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// given
			src := tt.src

			// when
			lines := classify(t, src)

			// then
			assert.Equal(t, 1, lines.Comment(tt.line, tt.line))
		})
	}
}

func Test_Classify_shouldNotCountCommentLine_whenCommentIsEmptyOrDirective(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		comment string
	}{
		{name: "empty line comment", comment: "//"},
		{name: "line comment with only spaces", comment: "//   "},
		{name: "go:generate", comment: "//go:generate x"},
		{name: "go:build", comment: "//go:build x"},
		{name: "nolint", comment: "//nolint:errcheck"},
		{name: "lint:ignore", comment: "//lint:ignore SA1000 reason"},
		{name: "line directive", comment: "//line a.go:1"},
		{name: "empty block comment", comment: "/* */"},
		{name: "block comment without space", comment: "/**/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// given
			src := srcWithLine3(tt.comment)

			// when
			lines := classify(t, src)

			// then
			assert.Equal(t, 0, lines.Comment(3, 3))
		})
	}
}

func Test_Classify_shouldCountCommentLine_whenTextOnlyResemblesDirective(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		comment string
	}{
		{name: "space before go:", comment: "// go:generate x"},
		{name: "line without space", comment: "//lines of text"},
		{name: "space before nolint", comment: "// nolint is a directive"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// given
			src := srcWithLine3(tt.comment)

			// when
			lines := classify(t, src)

			// then
			assert.Equal(t, 1, lines.Comment(3, 3))
		})
	}
}

func Test_Classify_shouldSkipBlankInnerLines_whenBlockCommentSpansLines(t *testing.T) {
	t.Parallel()

	// given
	src := "package a\n\n/*\n * first\n *\n\n   second\n */\nvar x = 1\n"

	// when
	lines := classify(t, src)

	// then
	assert.Equal(t, 2, lines.Comment(3, 8))
}

type mixedLineCase struct {
	name string
	src  string
	line int
}

func mixedLineCases() []mixedLineCase {
	return []mixedLineCase{
		{name: "line comment after code", src: "package a\n\nfunc f() {\n\tx := 1 // note\n\t_ = x\n}\n", line: 4},
		{name: "block comment before code", src: srcWithLine3("/* a */ var y = 2"), line: 3},
		{name: "code after multi-line block comment", src: "package a\n\n/* a\n b */ var y = 2\n", line: 4},
	}
}

func Test_Classify_shouldCountCodeLine_whenLineAlsoHasComment(t *testing.T) {
	t.Parallel()

	for _, tt := range mixedLineCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// given
			src := tt.src

			// when
			lines := classify(t, src)

			// then
			assert.Equal(t, 1, lines.Code(tt.line, tt.line))
		})
	}
}

func Test_Classify_shouldCountCommentLine_whenLineAlsoHasCode(t *testing.T) {
	t.Parallel()

	for _, tt := range mixedLineCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// given
			src := tt.src

			// when
			lines := classify(t, src)

			// then
			assert.Equal(t, 1, lines.Comment(tt.line, tt.line))
		})
	}
}

func Test_Classify_shouldCountEveryLineAsCode_whenRawStringSpansLines(t *testing.T) {
	t.Parallel()

	// given
	src := "package a\n\nvar s = `a\n\n\nb`\n"

	// when
	lines := classify(t, src)

	// then
	assert.Equal(t, 4, lines.Code(3, 6))
}

func perLine(lines linecount.Lines) [][2]int {
	got := make([][2]int, 0, lines.LineCount())
	for i := 1; i <= lines.LineCount(); i++ {
		got = append(got, [2]int{lines.Code(i, i), lines.Comment(i, i)})
	}

	return got
}

func Test_Classify_shouldClassifySameAsLF_whenSourceUsesCRLF(t *testing.T) {
	t.Parallel()

	// given
	lf := "// header\n\npackage a\n\n//\n/*\n * a\n\n */\nvar s = `x\n\ny` // note\n"
	crlf := strings.ReplaceAll(lf, "\n", "\r\n")

	// when
	got := perLine(classify(t, crlf))

	// then
	assert.Equal(t, perLine(classify(t, lf)), got)
}

func Test_Classify_shouldUsePhysicalLine_whenLineDirectiveIsPresent(t *testing.T) {
	t.Parallel()

	// given
	src := "package a\n\n//line other.go:100\nvar x = 1\n"

	// when
	lines := classify(t, src)

	// then
	assert.Equal(t, 1, lines.Code(4, 4))
}

func Test_Classify_shouldReportFirstError_whenSourceHasManyErrors(t *testing.T) {
	t.Parallel()

	// given
	src := []byte("package a\n#\n#\n")

	// when
	_, err := linecount.Classify("a.go", src)

	// then
	require.ErrorContains(t, err, "a.go:2:1")
}

func Test_Classify_shouldNotReportLaterErrors_whenSourceHasManyErrors(t *testing.T) {
	t.Parallel()

	// given
	src := []byte("package a\n#\n#\n")

	// when
	_, err := linecount.Classify("a.go", src)

	// then
	require.ErrorIs(t, err, linecount.ErrScan)
	assert.NotContains(t, err.Error(), "a.go:3:1")
}

func Test_Classify_shouldReportPhysicalPosition_whenErrorFollowsLineDirective(t *testing.T) {
	t.Parallel()

	// given
	src := []byte("package a\n\n//line other.go:100\n#\n")

	// when
	_, err := linecount.Classify("a.go", src)

	// then
	require.ErrorContains(t, err, "a.go:4:1")
}

func Test_Classify_shouldReturnErrScan_whenSourceHasIllegalCharacter(t *testing.T) {
	t.Parallel()

	// given
	src := []byte("package a\n\x00\n")

	// when
	_, err := linecount.Classify("a.go", src)

	// then
	require.ErrorIs(t, err, linecount.ErrScan)
}

func Test_Lines_LineCount_shouldReturnNumberOfLines_whenSourceEndsWithNewline(t *testing.T) {
	t.Parallel()

	// given
	src := "package a\n\nvar x = 1\n"

	// when
	lines := classify(t, src)

	// then
	assert.Equal(t, 3, lines.LineCount())
}

func Test_Classify_shouldCountLastLine_whenSourceDoesNotEndWithNewline(t *testing.T) {
	t.Parallel()

	// given
	src := "package a\n\nvar x = 1 // note"

	// when
	lines := classify(t, src)

	// then
	assert.Equal(t, 1, lines.Code(3, 3))
}

func Test_Lines_LineCount_shouldReturnZero_whenLinesIsZeroValue(t *testing.T) {
	t.Parallel()

	// given
	var lines linecount.Lines

	// when
	got := lines.LineCount()

	// then
	assert.Equal(t, 0, got)
}

func Test_Lines_Code_shouldReturnZero_whenRangeIsEmpty(t *testing.T) {
	t.Parallel()

	// given
	lines := classify(t, "package a\n\nvar x = 1\n")

	// when
	got := lines.Code(3, 1)

	// then
	assert.Equal(t, 0, got)
}

func Test_Lines_Code_shouldClampRange_whenRangeExceedsFile(t *testing.T) {
	t.Parallel()

	// given
	lines := classify(t, "package a\n\nvar x = 1\nvar y = 2\n")

	// when
	got := lines.Code(-5, 100)

	// then
	assert.Equal(t, 3, got)
}

func Test_Lines_Comment_shouldClampRange_whenRangeExceedsFile(t *testing.T) {
	t.Parallel()

	// given
	lines := classify(t, "// a\n// b\npackage a\n")

	// when
	got := lines.Comment(0, 100)

	// then
	assert.Equal(t, 2, got)
}

func Test_Lines_Code_shouldReturnZero_whenLinesIsZeroValue(t *testing.T) {
	t.Parallel()

	// given
	var lines linecount.Lines

	// when
	got := lines.Code(1, 10)

	// then
	assert.Equal(t, 0, got)
}

func Test_CommentLineFlags_shouldFlagLinesWithContent_whenBlockCommentSpansLines(t *testing.T) {
	t.Parallel()

	// given
	lit := "/* first\n *\n * second\n\n*/"

	// when
	got := linecount.CommentLineFlags(lit)

	// then
	assert.Equal(t, []bool{true, false, true, false, false}, got)
}

func Test_CommentLineFlags_shouldNotFlagLine_whenLineHasOnlyStarsAndSpaces(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		line string
	}{
		{name: "stars", line: " *****"},
		{name: "stars separated by spaces", line: " * * *"},
		{name: "stars separated by tab", line: " *\t*"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// given
			lit := "/* text\n" + tt.line + "\n*/"

			// when
			got := linecount.CommentLineFlags(lit)

			// then
			assert.Equal(t, []bool{true, false, false}, got)
		})
	}
}

func Test_CommentLineFlags_shouldFlagLine_whenStarsSurroundText(t *testing.T) {
	t.Parallel()

	// given
	lit := "/* **bold** */"

	// when
	got := linecount.CommentLineFlags(lit)

	// then
	assert.Equal(t, []bool{true}, got)
}
