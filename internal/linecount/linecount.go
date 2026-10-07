package linecount

import (
	"bytes"
	"errors"
	"fmt"
	"go/scanner"
	"go/token"
	"strings"
	"unicode"
)

// ErrScan is returned when the source cannot be tokenized.
var ErrScan = errors.New("scan source")

// Lines counts code lines and comment lines over line ranges of a file.
type Lines struct {
	code    []int
	comment []int
}

// Classify scans src once and records, for each line, whether it has code and whether it has a comment to count.
// If src cannot be tokenized, it returns the first error, wrapping ErrScan, at its physical position.
func Classify(filename string, src []byte) (Lines, error) {
	fset := token.NewFileSet()
	tf := fset.AddFile(filename, -1, len(src))

	var firstErr error

	handleErr := func(pos token.Position, msg string) {
		if firstErr == nil {
			physical := tf.PositionFor(tf.Pos(pos.Offset), false)
			firstErr = fmt.Errorf("%w: %s: %s", ErrScan, physical, msg)
		}
	}

	var s scanner.Scanner
	s.Init(tf, src, handleErr, scanner.ScanComments)

	code, comment := scanLines(&s, tf, bytes.Count(src, []byte("\n"))+1)
	if firstErr != nil {
		return Lines{}, firstErr
	}

	return Lines{code: prefixSums(code, tf.LineCount()), comment: prefixSums(comment, tf.LineCount())}, nil
}

func scanLines(s *scanner.Scanner, tf *token.File, maxLines int) ([]bool, []bool) {
	code := make([]bool, maxLines+1)
	comment := make([]bool, maxLines+1)

	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			return code, comment
		}

		// Positions ignore //line directives, which would otherwise shift line numbers.
		start := tf.PositionFor(pos, false).Line

		switch {
		case tok == token.SEMICOLON && lit == "\n":
			// Automatically inserted semicolons are not code.
		case tok == token.COMMENT:
			for i, counted := range CommentLineFlags(lit) {
				if counted {
					comment[start+i] = true
				}
			}
		default:
			end := start + strings.Count(lit, "\n")
			for line := start; line <= end; line++ {
				code[line] = true
			}
		}
	}
}

func prefixSums(flags []bool, lineCount int) []int {
	sums := make([]int, lineCount+1)
	for line := 1; line <= lineCount; line++ {
		sums[line] = sums[line-1]
		if flags[line] {
			sums[line]++
		}
	}

	return sums
}

// LineCount returns the number of lines in the file.
func (l Lines) LineCount() int {
	if len(l.code) == 0 {
		return 0
	}

	return len(l.code) - 1
}

// Code returns the number of code lines in the closed range [from, to], clamped to the file.
func (l Lines) Code(from, to int) int {
	return l.count(l.code, from, to)
}

// Comment returns the number of comment lines in the closed range [from, to], clamped to the file.
func (l Lines) Comment(from, to int) int {
	return l.count(l.comment, from, to)
}

func (l Lines) count(sums []int, from, to int) int {
	from = max(from, 1)
	to = min(to, l.LineCount())

	if from > to {
		return 0
	}

	return sums[to] - sums[from-1]
}

// isDirective reports whether a // comment is a directive such as //go:generate or //nolint.
func isDirective(text string) bool {
	for _, prefix := range []string{"//go:", "//nolint", "//lint:", "//line "} {
		if strings.HasPrefix(text, prefix) {
			return true
		}
	}

	return false
}

// CommentLineFlags reports, for each line of a comment token, whether the line is counted as a comment line.
func CommentLineFlags(lit string) []bool {
	if strings.HasPrefix(lit, "//") {
		return []bool{!isDirective(lit) && strings.TrimSpace(lit[len("//"):]) != ""}
	}

	body := strings.TrimSuffix(strings.TrimPrefix(lit, "/*"), "*/")
	parts := strings.Split(body, "\n")
	flags := make([]bool, len(parts))

	for i, part := range parts {
		flags[i] = strings.TrimFunc(part, isDecoration) != ""
	}

	return flags
}

func isDecoration(r rune) bool {
	return r == '*' || unicode.IsSpace(r)
}
