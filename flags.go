package commentratio

import "flag"

func registerFlags(fs *flag.FlagSet, s *Settings) {
	registerDocRuleFlags(fs, keyFuncDoc, &s.FuncDoc)
	registerDocRuleFlags(fs, keyDeclDoc, &s.DeclDoc)
	registerRuleFlags(fs, keyFuncBody, &s.FuncBody)
	registerRuleFlags(fs, keyFile, &s.File)
}

func registerRuleFlags(fs *flag.FlagSet, prefix string, r *Rule) {
	fs.BoolVar(&r.Enabled, prefix+"."+keyEnabled, r.Enabled, "enable the "+prefix+" check")
	fs.IntVar(&r.FreeLines, prefix+"."+keyFreeLines, r.FreeLines, "comment lines always allowed")
	fs.Float64Var(&r.MaxRatio, prefix+"."+keyMaxRatio, r.MaxRatio, "maximum ratio of comment lines to code lines")
	fs.IntVar(&r.MaxLines, prefix+"."+keyMaxLines, r.MaxLines, "maximum comment lines")
}

func registerDocRuleFlags(fs *flag.FlagSet, prefix string, r *DocRule) {
	registerRuleFlags(fs, prefix, &r.Rule)
	fs.IntVar(&r.RequireFrom, prefix+"."+keyRequireFrom, r.RequireFrom,
		"code lines from which a doc comment is required (0 disables)")
}
