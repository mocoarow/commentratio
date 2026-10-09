package testfuncs

import "testing"

func TestLong(t *testing.T) {
	_ = 1
	_ = 2
	_ = 3
}

func Test_long(t *testing.T) {
	_ = 1
	_ = 2
	_ = 3
}

func BenchmarkLong(b *testing.B) {
	_ = 1
	_ = 2
	_ = 3
}

func FuzzLong(f *testing.F) {
	_ = 1
	_ = 2
	_ = 3
}

func ExampleLong() {
	_ = 1
	_ = 2
	_ = 3
}

func Example() {
	_ = 1
	_ = 2
	_ = 3
}
