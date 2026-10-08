package funcbodyok

func NoBody()

func Free() {
	// a
	// b
}

func Ratio() {
	// a
	// b
	// c
	_ = 1
	_ = 2
	_ = 3
	_ = 4
}

func Max() {
	// a
	// b
	// c
	// d
	_ = 1
	_ = 2
	_ = 3
	_ = 4
	_ = 5
	_ = 6
	_ = 7
	_ = 8
}
