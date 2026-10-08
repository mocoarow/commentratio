package funcdocng

// FreeOver line 1. // want `doc comment of FreeOver is too long: 3 comment lines for 2 code lines \(max 2\)`
// line 2.
// line 3.
func FreeOver() {
}

// RatioOver line 1. // want `doc comment of RatioOver is too long: 4 comment lines for 6 code lines \(max 3\)`
// line 2.
// line 3.
// line 4.
func RatioOver() {
	_ = 1
	_ = 2
	_ = 3
	_ = 4
}

// MaxOver line 1. // want `doc comment of MaxOver is too long: 5 comment lines for 10 code lines \(max 4\)`
// line 2.
// line 3.
// line 4.
// line 5.
func MaxOver() {
	_ = 1
	_ = 2
	_ = 3
	_ = 4
	_ = 5
	_ = 6
	_ = 7
	_ = 8
}

// T is a type.
type T struct{}

// M line 1. // want `doc comment of T.M is too long: 3 comment lines for 2 code lines \(max 2\)`
// line 2.
// line 3.
func (t *T) M() {
}

// G is a generic type.
type G[K comparable, V any] struct{}

// M line 1. // want `doc comment of G.M is too long: 3 comment lines for 2 code lines \(max 2\)`
// line 2.
// line 3.
func (g *G[K, V]) M() {
}

// H is a generic type.
type H[K any] struct{}

// M line 1. // want `doc comment of H.M is too long: 3 comment lines for 2 code lines \(max 2\)`
// line 2.
// line 3.
func (h H[K]) M() {
}

// P is a type.
type P struct{}

// M line 1. // want `doc comment of P.M is too long: 3 comment lines for 2 code lines \(max 2\)`
// line 2.
// line 3.
func (p (P)) M() {
}
