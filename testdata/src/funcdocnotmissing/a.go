package funcdocnotmissing

func Short() {
	_ = 1
	_ = 2
}

func unexported() {
	_ = 1
	_ = 2
	_ = 3
}

func _() {
	_ = 1
	_ = 2
	_ = 3
}

func NoBody(a, b int) int
