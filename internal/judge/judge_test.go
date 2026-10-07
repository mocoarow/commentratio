package judge_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mocoarow/commentratio/internal/judge"
)

func funcDocLimits() judge.Limits {
	return judge.Limits{FreeLines: 3, MaxRatio: 0.2, MaxLines: 15}
}

func Test_Limit_shouldReturnSpecExample_whenCodeLinesVary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		code int
		want int
	}{
		{name: "3 lines", code: 3, want: 3},
		{name: "10 lines", code: 10, want: 3},
		{name: "20 lines", code: 20, want: 4},
		{name: "50 lines", code: 50, want: 10},
		{name: "100 lines", code: 100, want: 15},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// given
			limits := funcDocLimits()

			// when
			got := judge.Limit(tt.code, limits)

			// then
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_Limit_shouldReturnFreeLines_whenCodeIsZero(t *testing.T) {
	t.Parallel()

	// given
	limits := funcDocLimits()

	// when
	got := judge.Limit(0, limits)

	// then
	assert.Equal(t, limits.FreeLines, got)
}

func Test_Limit_shouldReturnFreeLines_whenRatioIsZero(t *testing.T) {
	t.Parallel()

	// given
	limits := judge.Limits{FreeLines: 3, MaxRatio: 0, MaxLines: 15}

	// when
	got := judge.Limit(100, limits)

	// then
	assert.Equal(t, limits.FreeLines, got)
}

func Test_Limit_shouldNotUnderflow_whenProductIsInexact(t *testing.T) {
	t.Parallel()

	// given
	limits := judge.Limits{FreeLines: 0, MaxRatio: 0.29, MaxLines: 100}

	// when
	got := judge.Limit(100, limits)

	// then
	assert.Equal(t, 29, got)
}

func Test_Limit_shouldReturnMaxLines_whenProductExceedsIntRange(t *testing.T) {
	t.Parallel()

	// given
	limits := judge.Limits{FreeLines: 3, MaxRatio: 1e300, MaxLines: 15}

	// when
	got := judge.Limit(100, limits)

	// then
	assert.Equal(t, limits.MaxLines, got)
}

func Test_Limit_shouldReturnMaxLines_whenMaxLinesIsMaxInt(t *testing.T) {
	t.Parallel()

	// given
	limits := judge.Limits{FreeLines: 3, MaxRatio: 1e300, MaxLines: math.MaxInt}

	// when
	got := judge.Limit(100, limits)

	// then
	assert.Equal(t, limits.MaxLines, got)
}

func Test_Limit_shouldReturnMaxLines_whenProductEqualsMaxLinesAsFloat(t *testing.T) {
	t.Parallel()

	// given
	limits := judge.Limits{FreeLines: 0, MaxRatio: math.Exp2(63), MaxLines: math.MaxInt}

	// when
	got := judge.Limit(1, limits)

	// then
	assert.Equal(t, limits.MaxLines, got)
}

func Test_Limit_shouldReturnMaxLines_whenFreeLinesExceedsMaxLines(t *testing.T) {
	t.Parallel()

	// given
	limits := judge.Limits{FreeLines: 20, MaxRatio: 0.2, MaxLines: 15}

	// when
	got := judge.Limit(10, limits)

	// then
	assert.Equal(t, limits.MaxLines, got)
}

func Test_Limit_shouldReturnMaxInt_whenFreeLinesAndMaxLinesAreMaxInt(t *testing.T) {
	t.Parallel()

	// given
	limits := judge.Limits{FreeLines: math.MaxInt, MaxRatio: 0, MaxLines: math.MaxInt}

	// when
	got := judge.Limit(100, limits)

	// then
	assert.Equal(t, math.MaxInt, got)
}

type limitBoundaryCase struct {
	name  string
	code  int
	limit int
}

func limitBoundaryCases() []limitBoundaryCase {
	return []limitBoundaryCase{
		{name: "limit set by free-lines", code: 10, limit: 3},
		{name: "limit set by max-ratio", code: 20, limit: 4},
		{name: "limit set by max-lines", code: 100, limit: 15},
	}
}

func Test_TooMany_shouldReturnFalse_whenCommentEqualsLimit(t *testing.T) {
	t.Parallel()

	for _, tt := range limitBoundaryCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// given
			limits := funcDocLimits()

			// when
			got := judge.TooMany(tt.limit, tt.code, limits)

			// then
			assert.False(t, got)
		})
	}
}

func Test_TooMany_shouldReturnTrue_whenCommentExceedsLimitByOne(t *testing.T) {
	t.Parallel()

	for _, tt := range limitBoundaryCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// given
			limits := funcDocLimits()

			// when
			got := judge.TooMany(tt.limit+1, tt.code, limits)

			// then
			assert.True(t, got)
		})
	}
}

func Test_RequiresDoc_shouldReturnTrue_whenCodeEqualsRequireFrom(t *testing.T) {
	t.Parallel()

	// given
	code, requireFrom := 10, 10

	// when
	got := judge.RequiresDoc(code, requireFrom, true, false)

	// then
	assert.True(t, got)
}

func Test_RequiresDoc_shouldReturnFalse_whenNotRequired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		code        int
		requireFrom int
		exported    bool
		hasDoc      bool
	}{
		{name: "code is below require-from", code: 9, requireFrom: 10, exported: true, hasDoc: false},
		{name: "require-from is zero", code: 100, requireFrom: 0, exported: true, hasDoc: false},
		{name: "identifier is not exported", code: 10, requireFrom: 10, exported: false, hasDoc: false},
		{name: "doc exists", code: 10, requireFrom: 10, exported: true, hasDoc: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// given
			code, requireFrom := tt.code, tt.requireFrom

			// when
			got := judge.RequiresDoc(code, requireFrom, tt.exported, tt.hasDoc)

			// then
			assert.False(t, got)
		})
	}
}
