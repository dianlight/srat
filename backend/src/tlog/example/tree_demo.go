package main

import (
	"github.com/dianlight/srat/tlog"
	"gitlab.com/tozd/go/errors"
)

func treeDemo() {
	// Enable colors for demo
	tlog.EnableColors(true)
	tlog.SetLevel(tlog.LevelDebug)

	// Create a nested error to demonstrate tree formatting
	err := createSampleError()

	println("=== Tree-formatted stack trace demo ===")
	tlog.Error("Demo: Nested error with tree-formatted stack", "error", err)

	println("\n=== ASCII fallback demo ===")
	tlog.EnableColors(false)
	tlog.Error("Demo: ASCII fallback formatting", "error", err)
}

func createSampleError() errors.E {
	return level1()
}

func level1() errors.E {
	return level2()
}

func level2() errors.E {
	return errors.WithStack(
		errors.WithDetails(
			errors.New("sample error for tree demonstration"),
			"component", "demo",
			"level", 2,
		),
	)
}
