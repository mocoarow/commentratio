package decldocmissing

type Big struct { // want `Big has no doc comment: 5 code lines \(doc required from 5\)`
	A int
	B int
	C int
}

var (
	Table = []int{ // want `Table has no doc comment: 5 code lines \(doc required from 5\)`
		1,
		2,
		3,
	}
)

var lower, Upper = []int{ // want `Upper has no doc comment: 5 code lines \(doc required from 5\)`
	1,
	2,
	3,
}, []int{}
