package main

import (
	"log/slog"
	"os"

	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/mocoarow/commentratio"
)

func main() {
	a, err := commentratio.NewAnalyzer(commentratio.DefaultSettings())
	if err != nil {
		slog.Error("create analyzer", "error", err)
		os.Exit(1)
	}

	singlechecker.Main(a)
}
