package funcbodyzero

func F() {
	_ = 1
} // end // want `comments in body of F are too many: 1 comment lines for 3 code lines \(max 0\)`
