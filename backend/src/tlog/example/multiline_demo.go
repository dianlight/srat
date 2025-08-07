package main

import (
	"log/slog"

	"github.com/dianlight/srat/tlog"
)

func multilineDemo() {
	// Enable colors for demo
	tlog.EnableColors(true)
	tlog.SetLevel(tlog.LevelDebug)

	// Create a nested error to demonstrate both formats
	err := createSampleError()

	println("=== Default single-line stacktrace (escaped) ===")
	tlog.EnableMultilineStacktrace(false)
	tlog.Error("Demo: Single-line stacktrace", "error", err)

	println("\n=== Multiline stacktrace (separate log attributes) ===")
	tlog.EnableMultilineStacktrace(true)
	tlog.Error("Demo: Multiline stacktrace", "error", err)

	println("\n=== Manual demonstration of multiline format ===")
	// Show what the multiline format looks like when formatted directly
	formatter := tlog.TozdErrorFormatter("error")
	value, _ := formatter(nil, slog.Attr{
		Key:   "error",
		Value: slog.AnyValue(err),
	})

	attrs := value.Group()
	for _, attr := range attrs {
		if attr.Key == "stacktrace" {
			println("Stacktrace frames:")
			if attr.Value.Kind() == slog.KindGroup {
				stackAttrs := attr.Value.Group()
				for _, frameAttr := range stackAttrs {
					println("  ", frameAttr.Key, ":", frameAttr.Value.String())
				}
			} else {
				println("Single string format:")
				println(attr.Value.String())
			}
			break
		}
	}
}
