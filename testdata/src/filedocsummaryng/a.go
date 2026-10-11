package filedocsummaryng // want `file comments are too many: 6 comment lines for 8 code lines \(max 4\)`

// Imports doc.
import "fmt"

// G does g.
// More about g.
func G() { _ = fmt.Sprint() }

//go:noinline
func H() {}

// Group doc.
// More about group.
const (
	// C is c.
	// More about c.
	C = 3
)

// T is t.
// More about t.
type T struct{}
