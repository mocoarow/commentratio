package decldocok

import "fmt"

var _ = fmt.Sprint

// Plain line 1.
// line 2.
type Plain struct {
	A int
}

// GroupA line 1.
// line 2.
const (
	GroupA = 1
	GroupB = 2
)

var (
	// SpecA line 1.
	// line 2.
	SpecA = 1
)

const ()
