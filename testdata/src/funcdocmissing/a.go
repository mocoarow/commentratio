package funcdocmissing

func Exact() { // want `Exact has no doc comment: 5 code lines \(doc required from 5\)`
	_ = 1
	_ = 2
	_ = 3
}

type T struct{}

func (t *T) Method() { // want `T.Method has no doc comment: 5 code lines \(doc required from 5\)`
	_ = 1
	_ = 2
	_ = 3
}

type G[X any] struct{}

func (g G[X]) Method() { // want `G.Method has no doc comment: 5 code lines \(doc required from 5\)`
	_ = 1
	_ = 2
	_ = 3
}

//go:noinline
func DirectiveOnly() { // want `DirectiveOnly has no doc comment: 5 code lines \(doc required from 5\)`
	_ = 1
	_ = 2
	_ = 3
}
