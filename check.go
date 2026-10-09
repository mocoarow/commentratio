package commentratio

import (
	"go/ast"
	"go/token"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/mocoarow/commentratio/internal/judge"
	"github.com/mocoarow/commentratio/internal/linecount"
)

const (
	msgDocTooLong  = "doc comment of %s is too long: %d comment lines for %d code lines (max %d)"
	msgDocMissing  = "%s has no doc comment: %d code lines (doc required from %d)"
	msgBodyTooMany = "comments in body of %s are too many: %d comment lines for %d code lines (max %d)"
	msgFileTooMany = "file comments are too many: %d comment lines for %d code lines (max %d)"
)

func (r Rule) limits() judge.Limits {
	return judge.Limits{FreeLines: r.FreeLines, MaxRatio: r.MaxRatio, MaxLines: r.MaxLines}
}

func (r *runner) check(c fileContext) {
	s := r.settings

	for _, decl := range c.file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if s.FuncDoc.Enabled {
				c.checkFuncDoc(s.FuncDoc, d)
			}

			if s.FuncBody.Enabled {
				c.checkFuncBody(s.FuncBody, d)
			}
		case *ast.GenDecl:
			if s.DeclDoc.Enabled {
				c.checkGenDecl(s.DeclDoc, d)
			}
		}
	}

	if s.File.Enabled {
		c.checkFile(s.File)
	}
}

func (c fileContext) checkFuncDoc(rule DocRule, d *ast.FuncDecl) {
	name := funcName(d)
	code := c.codeLines(d.Pos(), d.End())

	c.reportLongDoc(rule.Rule, d.Doc, name, code)

	if !c.isTestFunc(d) {
		c.reportMissingDoc(rule, d.Name, name, code, c.commentLines(d.Doc) > 0)
	}
}

func (c fileContext) isTestFunc(d *ast.FuncDecl) bool {
	if d.Recv != nil || !strings.HasSuffix(c.tf.Name(), "_test.go") {
		return false
	}

	for _, prefix := range []string{"Test", "Benchmark", "Fuzz", "Example"} {
		rest, ok := strings.CutPrefix(d.Name.Name, prefix)
		first, _ := utf8.DecodeRuneInString(rest)

		if ok && (rest == "" || !unicode.IsLower(first)) {
			return true
		}
	}

	return false
}

func (c fileContext) checkFuncBody(rule Rule, d *ast.FuncDecl) {
	if d.Body == nil {
		return
	}

	from, to := c.line(d.Body.Lbrace), c.line(d.Body.Rbrace)
	code, comment := c.lines.Code(from, to), c.lines.Comment(from, to)

	if judge.TooMany(comment, code, rule.limits()) {
		pos := c.firstCountedComment(from, to, d.Body.Lbrace)
		c.pass.Reportf(pos, msgBodyTooMany, funcName(d), comment, code, judge.Limit(code, rule.limits()))
	}
}

func (c fileContext) checkGenDecl(rule DocRule, d *ast.GenDecl) {
	if d.Tok == token.IMPORT || len(d.Specs) == 0 {
		return
	}

	_, firstNames := specDocAndNames(d.Specs[0])
	c.reportLongDoc(rule.Rule, d.Doc, firstNames[0].Name, c.codeLines(d.Pos(), d.End()))

	groupHasDoc := c.commentLines(d.Doc) > 0

	for _, spec := range d.Specs {
		specDoc, names := specDocAndNames(spec)
		code := c.codeLines(spec.Pos(), spec.End())

		c.reportLongDoc(rule.Rule, specDoc, names[0].Name, code)

		id := firstExported(names)
		c.reportMissingDoc(rule, id, id.Name, code, groupHasDoc || c.commentLines(specDoc) > 0)
	}
}

func (c fileContext) checkFile(rule Rule) {
	last := c.lines.LineCount()
	code := c.lines.Code(1, last)
	comment := c.lines.Comment(c.line(c.file.Package), last)

	if judge.TooMany(comment, code, rule.limits()) {
		c.pass.Reportf(c.file.Package, msgFileTooMany, comment, code, judge.Limit(code, rule.limits()))
	}
}

func (c fileContext) reportLongDoc(rule Rule, doc *ast.CommentGroup, name string, code int) {
	comment := c.commentLines(doc)
	if doc != nil && judge.TooMany(comment, code, rule.limits()) {
		c.pass.Reportf(doc.Pos(), msgDocTooLong, name, comment, code, judge.Limit(code, rule.limits()))
	}
}

func (c fileContext) reportMissingDoc(rule DocRule, id *ast.Ident, name string, code int, hasDoc bool) {
	if judge.RequiresDoc(code, rule.RequireFrom, id.IsExported(), hasDoc) {
		c.pass.Reportf(id.Pos(), msgDocMissing, name, code, rule.RequireFrom)
	}
}

func (c fileContext) firstCountedComment(from, to int, fallback token.Pos) token.Pos {
	for _, group := range c.file.Comments {
		if c.line(group.Pos()) > to {
			break
		}

		for _, comment := range group.List {
			start := c.line(comment.Pos())
			for i, counted := range linecount.CommentLineFlags(comment.Text) {
				if counted && from <= start+i && start+i <= to {
					return comment.Pos()
				}
			}
		}
	}

	return fallback
}

func funcName(d *ast.FuncDecl) string {
	if d.Recv == nil || len(d.Recv.List) == 0 {
		return d.Name.Name
	}

	return receiverTypeName(d.Recv.List[0].Type) + "." + d.Name.Name
}

func receiverTypeName(expr ast.Expr) string {
	for {
		switch e := expr.(type) {
		case *ast.StarExpr:
			expr = e.X
		case *ast.ParenExpr:
			expr = e.X
		case *ast.IndexExpr:
			expr = e.X
		case *ast.IndexListExpr:
			expr = e.X
		case *ast.Ident:
			return e.Name
		default:
			return ""
		}
	}
}

func specDocAndNames(spec ast.Spec) (*ast.CommentGroup, []*ast.Ident) {
	switch s := spec.(type) {
	case *ast.ValueSpec:
		return s.Doc, s.Names
	case *ast.TypeSpec:
		return s.Doc, []*ast.Ident{s.Name}
	default:
		return nil, nil
	}
}

func firstExported(names []*ast.Ident) *ast.Ident {
	for _, name := range names {
		if name.IsExported() {
			return name
		}
	}

	return names[0]
}
