package ignorepatterns

func MarkersOnly() {
	// given
	x := 1
	// when
	y := x
	// then
	_ = y
}

func MarkersAndProse() {
	// given
	// prose 1 // want `comments in body of MarkersAndProse are too many: 3 comment lines for 4 code lines \(max 2\)`
	// prose 2
	x := 1
	// prose 3
	// then
	_ = x
}
