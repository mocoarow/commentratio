package testfuncslookalike

func Testing() { // want `Testing has no doc comment: 5 code lines \(doc required from 5\)`
	_ = 1
	_ = 2
	_ = 3
}

type T struct{}

func (T) TestMethod() { // want `T.TestMethod has no doc comment: 5 code lines \(doc required from 5\)`
	_ = 1
	_ = 2
	_ = 3
}
