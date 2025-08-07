package tlog_test

import (
	"context"
	"testing"

	"github.com/dianlight/srat/tlog"
	"gitlab.com/tozd/go/errors"
)

// TestTozdErrorFormatter demonstrates the formatted output of tozd errors with stacktraces
func TestTozdErrorFormatter(t *testing.T) {
	// Set up tlog with debug level to see all output
	tlog.SetLevel(tlog.LevelDebug)

	// Enable colors for demonstration
	tlog.EnableColors(true)

	// Create a base error
	baseErr := errors.New("database connection failed")

	// Add details to the error
	detailedErr := errors.WithDetails(baseErr, "host", "localhost", "port", 5432, "database", "myapp")

	// Wrap the error with additional context
	wrappedErr := errors.Wrap(detailedErr, "failed to initialize user repository")

	// Add stack trace
	stackErr := errors.WithStack(wrappedErr)

	// Create another error to demonstrate error chains
	chainErr := errors.Wrap(stackErr, "service initialization failed")

	t.Log("Testing tozd error formatter with various error types...")

	// Test simple error
	tlog.Error("Simple tozd error", "error", errors.New("simple error message"))

	// Test error with details
	tlog.Error("Error with details", "error", detailedErr)

	// Test error with stack trace
	tlog.Error("Error with stack trace", "error", stackErr)

	// Test error chain
	tlog.Error("Error chain", "error", chainErr)

	// Test with context
	ctx := context.WithValue(context.Background(), "request_id", "req-12345")
	ctx = context.WithValue(ctx, "user_id", "user-67890")
	tlog.ErrorContext(ctx, "Error with context", "error", stackErr)

	// Test Join errors
	err1 := errors.New("first error")
	err2 := errors.New("second error")
	joinedErr := errors.Join(err1, err2)
	tlog.Error("Joined errors", "error", joinedErr)
}

// helper function to demonstrate stack trace generation
func createNestedError() errors.E {
	return deepFunction()
}

func deepFunction() errors.E {
	return veryDeepFunction()
}

func veryDeepFunction() errors.E {
	return errors.WithDetails(
		errors.New("something went wrong in deep function"),
		"level", "very_deep",
		"function", "veryDeepFunction",
	)
}

// TestNestedStackTrace demonstrates stack traces in nested function calls
func TestNestedStackTrace(t *testing.T) {
	tlog.SetLevel(tlog.LevelDebug)
	tlog.EnableColors(true)

	nestedErr := createNestedError()
	tlog.Error("Nested error with stack trace", "error", nestedErr)
}

// BenchmarkTozdErrorFormatter benchmarks the performance of the formatter
func BenchmarkTozdErrorFormatter(b *testing.B) {
	err := errors.WithDetails(
		errors.WithStack(errors.New("benchmark error")),
		"iteration", 0,
		"benchmark", true,
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tlog.Error("Benchmark error", "error", err, "iteration", i)
	}
}
