package funcbodyng

func Free() {
	// a // want `comments in body of Free are too many: 3 comment lines for 2 code lines \(max 2\)`
	// b
	// c
}

func Trailing() {
	x := 1 // a // want `comments in body of Trailing are too many: 3 comment lines for 4 code lines \(max 2\)`
	_ = x // b
	// c
}

func Max() {
	// a // want `comments in body of Max are too many: 5 comment lines for 10 code lines \(max 4\)`
	// b
	// c
	// d
	// e
	_ = 1
	_ = 2
	_ = 3
	_ = 4
	_ = 5
	_ = 6
	_ = 7
	_ = 8
}

func Blank() {
	//
	// a // want `comments in body of Blank are too many: 3 comment lines for 2 code lines \(max 2\)`
	// b
	// c
}

type T struct{}

func (t T) M() {
	// a // want `comments in body of T.M are too many: 3 comment lines for 2 code lines \(max 2\)`
	// b
	// c
}
