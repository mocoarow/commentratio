package commentratio

import (
	"flag"
	"fmt"
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"

	"github.com/mocoarow/commentratio/internal/linecount"
)

// Name is the name of the analyzer.
const Name = "commentratio"

const doc = "report comments that are too long for their code and exported declarations without doc comments"

// NewAnalyzer returns an analyzer that checks with s. Its flags override s.
func NewAnalyzer(s Settings) (*analysis.Analyzer, error) {
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("new analyzer: %w", err)
	}

	r := &runner{settings: s}
	a := &analysis.Analyzer{
		Name:             Name,
		Doc:              doc,
		URL:              "https://github.com/mocoarow/commentratio",
		Flags:            flag.FlagSet{Usage: nil},
		Run:              r.run,
		RunDespiteErrors: false,
		Requires:         nil,
		ResultType:       nil,
		FactTypes:        nil,
	}
	registerFlags(&a.Flags, &r.settings)

	return a, nil
}

type runner struct {
	settings Settings
}

func (r *runner) run(pass *analysis.Pass) (any, error) {
	// Flags are parsed after NewAnalyzer, so their values are validated here.
	if err := r.settings.Validate(); err != nil {
		return nil, fmt.Errorf("validate settings: %w", err)
	}

	for _, f := range pass.Files {
		if ast.IsGenerated(f) {
			continue
		}

		c, err := newFileContext(pass, f)
		if err != nil {
			return nil, err
		}

		r.check(c)
	}

	return nil, nil //nolint:nilnil // an analyzer without ResultType must return a nil result
}

type fileContext struct {
	pass  *analysis.Pass
	file  *ast.File
	tf    *token.File
	lines linecount.Lines
}

func newFileContext(pass *analysis.Pass, f *ast.File) (fileContext, error) {
	tf := pass.Fset.File(f.Pos())

	src, err := pass.ReadFile(tf.Name())
	if err != nil {
		return fileContext{}, fmt.Errorf("read %s: %w", tf.Name(), err)
	}

	lines, err := linecount.Classify(tf.Name(), src)
	if err != nil {
		return fileContext{}, fmt.Errorf("classify %s: %w", tf.Name(), err)
	}

	return fileContext{pass: pass, file: f, tf: tf, lines: lines}, nil
}

// line ignores //line directives so that it matches the lines counted by linecount.
func (c fileContext) line(p token.Pos) int {
	return c.tf.PositionFor(p, false).Line
}

func (c fileContext) codeLines(from, to token.Pos) int {
	return c.lines.Code(c.line(from), c.line(to))
}

func (c fileContext) commentLines(g *ast.CommentGroup) int {
	if g == nil {
		return 0
	}

	return c.lines.Comment(c.line(g.Pos()), c.line(g.End()))
}
