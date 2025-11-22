// Package tlog provides enhanced structured logging capabilities with support for
// gitlab.com/tozd/go/errors, including colored and formatted stacktraces.
//
// # Tozd Errors Integration
//
// The tlog package provides special formatting for errors created with the
// gitlab.com/tozd/go/errors package, displaying stacktraces in a colored and
// structured format.
//
// ## Basic Usage
//
//	package main
//
//	import (
//	    "github.com/dianlight/tlog"
//	    "gitlab.com/tozd/go/errors"
//	)
//
//	func main() {
//	    // Enable colors (automatic if terminal supports it)
//	    tlog.EnableColors(true)
//
//	    // Create an error with stack trace
//	    err := errors.WithStack(errors.New("something went wrong"))
//
//	    // Log the error - stacktrace will be automatically formatted
//	    tlog.Error("Operation failed", "error", err)
//
//	    // Add details to errors
//	    detailedErr := errors.WithDetails(err, "user_id", "12345", "operation", "login")
//	    tlog.Error("Detailed error", "error", detailedErr)
//
//	    // Chain errors
//	    chainedErr := errors.Wrap(detailedErr, "authentication failed")
//	    tlog.Error("Chained error", "error", chainedErr)
//	}
//
// ## Features
//
//   - **Colored Stack Traces**: The most recent stack frames are highlighted in red,
//     intermediate frames in yellow, and deeper frames in gray.
//
//   - **Structured Details**: Error details are displayed in a structured format
//     under the "error.details" group.
//
// - **Error Chains**: Cause errors are displayed under "error.cause".
//
//   - **Stack Frame Limiting**: Stack traces are limited to 20 frames to prevent
//     excessive output.
//
// ## Configuration
//
//	// Enable/disable colors
//	tlog.EnableColors(true)
//
//	// Check if colors are enabled
//	if tlog.IsColorsEnabled() {
//	    // Colors are supported and enabled
//	}
//
//	// Configure formatter settings
//	config := tlog.FormatterConfig{
//	    EnableColors:      true,
//	    EnableFormatting:  true,
//	    HideSensitiveData: false,
//	    TimeFormat:        time.RFC3339,
//	}
//	tlog.SetFormatterConfig(config)
//
// ## Example Output
//
// When logging a tozd error with stack trace:
//
//	2025-08-07T15:14:58+02:00 ERROR main.go:15 Operation failed
//	  error.message="something went wrong"
//	  error.stacktrace.frame_0="main.go:12 main.doSomething"
//	  error.stacktrace.frame_1="main.go:8 main.main"
//	  error.stacktrace.frame_2="runtime/proc.go:250 runtime.main"
//
// With colors enabled, the stack frames will be color-coded:
// - frame_0: Red (most recent)
// - frame_1: Yellow (intermediate)
// - frame_2+: Gray (deeper frames)
package tlog
