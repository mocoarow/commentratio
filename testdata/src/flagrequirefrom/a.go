package flagrequirefrom

func Short() { // want `Short has no doc comment: 4 code lines \(doc required from 4\)`
	_ = 1
	_ = 2
}
