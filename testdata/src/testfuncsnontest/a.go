package testfuncsnontest

func TestInNonTestFile() { // want `TestInNonTestFile has no doc comment: 5 code lines \(doc required from 5\)`
	_ = 1
	_ = 2
	_ = 3
}
