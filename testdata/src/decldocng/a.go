package decldocng

// Plain line 1. // want `doc comment of Plain is too long: 3 comment lines for 3 code lines \(max 2\)`
// line 2.
// line 3.
type Plain struct {
	A int
}

// GroupA line 1. // want `doc comment of GroupA is too long: 3 comment lines for 4 code lines \(max 2\)`
// line 2.
// line 3.
const (
	GroupA = 1
	GroupB = 2
)

var (
	// SpecA line 1. // want `doc comment of SpecA is too long: 3 comment lines for 1 code lines \(max 2\)`
	// line 2.
	// line 3.
	SpecA = 1
)

// Fields line 1. // want `doc comment of Fields is too long: 3 comment lines for 4 code lines \(max 2\)`
// line 2.
// line 3.
type Fields struct {
	// A is a.
	A int
	// B is b.
	B int
}
