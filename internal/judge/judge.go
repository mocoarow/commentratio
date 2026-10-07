package judge

import "math"

const ratioEpsilon = 1e-9

// Limits holds the values that decide the maximum number of comment lines.
type Limits struct {
	FreeLines int
	MaxRatio  float64
	MaxLines  int
}

// Limit returns the maximum number of comment lines allowed for code lines of code.
func Limit(code int, l Limits) int {
	byRatio := math.Floor(float64(code)*l.MaxRatio + ratioEpsilon)
	if byRatio >= float64(l.MaxLines) {
		return l.MaxLines
	}

	return min(l.MaxLines, max(l.FreeLines, int(byRatio)))
}

// TooMany reports whether comment lines exceed the limit for code lines of code.
func TooMany(comment, code int, l Limits) bool {
	return comment > Limit(code, l)
}

// RequiresDoc reports whether a declaration with code lines of code must have a doc comment but has none.
func RequiresDoc(code, requireFrom int, exported, hasDoc bool) bool {
	return requireFrom > 0 && code >= requireFrom && exported && !hasDoc
}
