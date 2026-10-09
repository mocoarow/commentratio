package testfile

import "testing"

// F line 1. // want `doc comment of F is too long: 3 comment lines for 2 code lines \(max 2\)`
// line 2.
// line 3.
func F() {
}

// TestLongDoc line 1. // want `doc comment of TestLongDoc is too long: 3 comment lines for 2 code lines \(max 2\)`
// line 2.
// line 3.
func TestLongDoc(t *testing.T) {
}
