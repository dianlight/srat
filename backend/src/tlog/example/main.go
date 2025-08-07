package main

import (
	"context"
	"fmt"
	"time"

	"github.com/dianlight/srat/tlog"
	"gitlab.com/tozd/go/errors"
)

func main() {
	fmt.Println("=== TLog Package Demonstration ===")
	fmt.Println()

	// Set initial level to INFO
	tlog.SetLevel(tlog.LevelInfo)
	fmt.Printf("Initial log level: %s\n", tlog.GetLevelString())

	// Demonstrate basic logging
	fmt.Println()
	fmt.Println("1. Basic Logging Functions:")
	tlog.Trace("This trace message won't appear (level too low)")
	tlog.Debug("This debug message won't appear (level too low)")
	tlog.Info("This is an info message", "component", "demo")
	tlog.Notice("This is a notice message", "action", "demonstration")
	tlog.Warn("This is a warning message", "issue", "example")
	tlog.Error("This is an error message", "error", "demonstration error")

	// Demonstrate level changing from strings
	fmt.Println()
	fmt.Println("2. Level Management:")
	levels := []string{"trace", "DEBUG", "Info", "NOTICE", "warn", "ERROR"}

	for _, level := range levels {
		err := tlog.SetLevelFromString(level)
		if err != nil {
			fmt.Printf("Error setting level '%s': %v\n", level, err)
			continue
		}
		fmt.Printf("Set level to: %s (current: %s)\n", level, tlog.GetLevelString())

		// Show what's enabled at this level
		fmt.Printf("  - Trace enabled: %v\n", tlog.IsLevelEnabled(tlog.LevelTrace))
		fmt.Printf("  - Debug enabled: %v\n", tlog.IsLevelEnabled(tlog.LevelDebug))
		fmt.Printf("  - Info enabled: %v\n", tlog.IsLevelEnabled(tlog.LevelInfo))
		fmt.Printf("  - Error enabled: %v\n", tlog.IsLevelEnabled(tlog.LevelError))
	}

	// Demonstrate error handling
	fmt.Println()
	fmt.Println("3. Error Handling:")
	err := tlog.SetLevelFromString("invalid_level")
	if err != nil {
		fmt.Printf("Expected error: %v\n", err)
	}

	err = tlog.SetLevelFromString("")
	if err != nil {
		fmt.Printf("Expected error: %v\n", err)
	}

	// Demonstrate context logging
	fmt.Println()
	fmt.Println("4. Context Logging:")
	tlog.SetLevel(tlog.LevelTrace)

	ctx := context.WithValue(context.Background(), "requestId", "demo-123")
	ctx = context.WithValue(ctx, "userId", "user-456")

	tlog.TraceContext(ctx, "Processing request", "operation", "demonstration")
	tlog.DebugContext(ctx, "Debug information", "step", 1)
	tlog.InfoContext(ctx, "Request processed", "duration", time.Millisecond*150)

	// Demonstrate level checking for expensive operations
	fmt.Println()
	fmt.Println("5. Performance Optimization:")
	tlog.SetLevel(tlog.LevelWarn)

	if tlog.IsLevelEnabled(tlog.LevelDebug) {
		// This expensive operation won't run because debug is disabled
		expensiveResult := performExpensiveOperation()
		tlog.Debug("Expensive operation result", "result", expensiveResult)
	} else {
		fmt.Println("Skipped expensive debug operation (debug level disabled)")
	}

	// Demonstrate custom logger
	fmt.Println()
	fmt.Println("6. Custom Logger Instance:")
	customLogger := tlog.WithLevel(tlog.LevelTrace)
	fmt.Println("Created custom logger with TRACE level (global level is still WARN)")

	// Note: We can't easily demonstrate the custom logger output here
	// but it would log at TRACE level while global logger remains at WARN
	_ = customLogger

	// Demonstrate callback functionality
	fmt.Println()
	fmt.Println("7. Event Callbacks:")

	// Counter for tracking callback executions
	var errorCount, warnCount int

	// Register callback for error level logs
	errorCallbackID := tlog.RegisterCallback(tlog.LevelError, func(event tlog.LogEvent) {
		errorCount++
		fmt.Printf("  [CALLBACK] Error detected: %s (count: %d)\n", event.Message, errorCount)

		// Example: Could send to monitoring system
		// monitoring.RecordError(event.Message, event.Args)
	})

	// Register callback for warn level logs
	warnCallbackID := tlog.RegisterCallback(tlog.LevelWarn, func(event tlog.LogEvent) {
		warnCount++
		fmt.Printf("  [CALLBACK] Warning logged: %s (count: %d)\n", event.Message, warnCount)

		// Example: Could log to audit system
		// auditLog.Record(event.Level, event.Message, event.Timestamp)
	})

	fmt.Printf("Registered callbacks - Error: %d, Warn: %d\n",
		tlog.GetCallbackCount(tlog.LevelError),
		tlog.GetCallbackCount(tlog.LevelWarn))

	// Trigger some logs that will execute callbacks
	tlog.SetLevel(tlog.LevelWarn)
	tlog.Warn("First warning message", "source", "demo")
	tlog.Error("First error message", "code", 500)
	tlog.Warn("Second warning message", "source", "demo")
	tlog.Error("Second error message", "code", 404)

	// Wait a moment for callbacks to execute (they're async)
	time.Sleep(time.Millisecond * 50)

	fmt.Printf("Final counts - Errors: %d, Warnings: %d\n", errorCount, warnCount)

	// Demonstrate callback management
	fmt.Println()
	fmt.Println("8. Callback Management:")

	// Show callback counts
	fmt.Printf("Current callback counts - Error: %d, Warn: %d\n",
		tlog.GetCallbackCount(tlog.LevelError),
		tlog.GetCallbackCount(tlog.LevelWarn))

	// Unregister error callback
	success := tlog.UnregisterCallback(tlog.LevelError, errorCallbackID)
	fmt.Printf("Unregistered error callback: %v\n", success)

	// Clear all warn callbacks
	tlog.ClearCallbacks(tlog.LevelWarn)
	fmt.Printf("Cleared all warn callbacks\n")

	// Suppress unused variable warning
	_ = warnCallbackID

	// Verify no callbacks remain
	fmt.Printf("Remaining callback counts - Error: %d, Warn: %d\n",
		tlog.GetCallbackCount(tlog.LevelError),
		tlog.GetCallbackCount(tlog.LevelWarn))

	// Test that callbacks no longer execute
	fmt.Println("Testing with no callbacks:")
	tlog.Error("This error won't trigger a callback")
	tlog.Warn("This warning won't trigger a callback")

	// Demonstrate callback error handling
	fmt.Println()
	fmt.Println("9. Callback Error Handling:")

	// Register a callback that will panic
	panicCallbackID := tlog.RegisterCallback(tlog.LevelError, func(event tlog.LogEvent) {
		fmt.Printf("  [CALLBACK] About to panic for: %s\n", event.Message)
		panic("intentional panic in callback")
	})

	// Register a normal callback
	normalCallbackID := tlog.RegisterCallback(tlog.LevelError, func(event tlog.LogEvent) {
		fmt.Printf("  [CALLBACK] Normal callback still works: %s\n", event.Message)
	})

	fmt.Println("Triggering error with panic callback...")
	tlog.Error("Test error for panic demonstration", "test", true)

	// Wait for callbacks to execute
	time.Sleep(time.Millisecond * 100)

	fmt.Println("Note: The panic in the callback was recovered automatically.")
	fmt.Println("Check the log output above for the panic recovery message.")

	// Clean up callbacks
	tlog.UnregisterCallback(tlog.LevelError, panicCallbackID)
	tlog.UnregisterCallback(tlog.LevelError, normalCallbackID)

	// Ensure graceful shutdown of callback processor
	defer tlog.Shutdown()

	fmt.Println()
	fmt.Println("=== Demonstration Complete ===")

	// Run the enhanced formatter demonstration
	formatterDemo()
	formatterColorDemo()
	treeDemo()
	fmt.Println("Formatter demonstration complete.")
}

func performExpensiveOperation() string {
	// Simulate expensive operation
	time.Sleep(time.Millisecond * 100)
	return "expensive_result"
}

// formatterDemo demonstrates the enhanced tozd error formatter capabilities
func formatterDemo() {
	fmt.Println()
	fmt.Println("=== Enhanced Tozd Error Formatter Demonstration ===")

	// Ensure we're at a level that shows errors
	tlog.SetLevel(tlog.LevelDebug)

	// Enable colors for better demonstration
	tlog.EnableColors(true)
	fmt.Printf("Colors enabled: %v\n", tlog.IsColorsEnabled())

	fmt.Println()
	fmt.Println("10. Basic Tozd Error with Stack Trace:")
	basicErr := errors.WithStack(errors.New("basic connection error"))
	tlog.Error("Basic tozd error demonstration", "error", basicErr)

	fmt.Println()
	fmt.Println("11. Tozd Error with Details:")
	detailedErr := errors.WithDetails(
		errors.New("database connection failed"),
		"host", "localhost",
		"port", 5432,
		"database", "user_db",
		"timeout", "30s",
		"retry_count", 3,
	)
	tlog.Error("Detailed tozd error", "error", detailedErr)

	fmt.Println()
	fmt.Println("12. Chained Tozd Errors:")
	baseErr := errors.WithDetails(
		errors.New("network timeout"),
		"endpoint", "https://api.example.com",
		"timeout_ms", 5000,
	)
	serviceErr := errors.Wrap(baseErr, "user service unavailable")
	controllerErr := errors.WithDetails(
		errors.Wrap(serviceErr, "failed to process user request"),
		"user_id", "12345",
		"request_id", "req-abc-def",
		"timestamp", time.Now().Unix(),
	)
	stackErr := errors.WithStack(controllerErr)

	tlog.Error("Complex error chain", "error", stackErr)

	fmt.Println()
	fmt.Println("13. Nested Function Call Stack:")
	nestedErr := createDeepError()
	tlog.Error("Error from nested function calls", "error", nestedErr)

	fmt.Println()
	fmt.Println("14. Error with Context Values:")
	ctx := context.WithValue(context.Background(), "request_id", "demo-req-789")
	ctx = context.WithValue(ctx, "user_id", "demo-user-456")
	ctx = context.WithValue(ctx, "session_id", "session-xyz")

	contextErr := errors.WithDetails(
		errors.WithStack(errors.New("permission denied")),
		"resource", "/api/users/secret",
		"required_role", "admin",
		"user_role", "user",
	)
	tlog.ErrorContext(ctx, "Authorization error with context", "error", contextErr)

	fmt.Println()
	fmt.Println("15. Joined Errors:")
	err1 := errors.WithStack(errors.New("first validation error"))
	err2 := errors.WithDetails(
		errors.New("second validation error"),
		"field", "email",
		"value", "invalid@",
	)
	err3 := errors.WithDetails(
		errors.New("third validation error"),
		"field", "age",
		"value", -5,
	)
	joinedErr := errors.Join(err1, err2, err3)
	tlog.Error("Multiple validation errors", "error", joinedErr)

	fmt.Println()
	fmt.Println("16. Performance Comparison:")
	fmt.Println("Standard error:")
	stdErr := fmt.Errorf("standard error: %s", "something went wrong")
	tlog.Error("Standard Go error", "error", stdErr)

	fmt.Println()
	fmt.Println("Tozd error with same message:")
	tozdErr := errors.WithStack(errors.New("something went wrong"))
	tlog.Error("Enhanced tozd error", "error", tozdErr)

	fmt.Println()
	fmt.Println("17. Color Demonstration (if terminal supports colors):")
	if tlog.IsColorsEnabled() {
		fmt.Println("The stack traces above should show:")
		fmt.Println("  - Recent frames in RED")
		fmt.Println("  - Intermediate frames in YELLOW")
		fmt.Println("  - Deeper frames in GRAY")
	} else {
		fmt.Println("Colors are not enabled (terminal may not support them)")
	}

	fmt.Println()
	fmt.Println("=== Tozd Error Formatter Demonstration Complete ===")
}

// Helper functions to create nested error stack traces
func createDeepError() errors.E {
	return levelOneFunction()
}

func levelOneFunction() errors.E {
	return levelTwoFunction()
}

func levelTwoFunction() errors.E {
	return levelThreeFunction()
}

func levelThreeFunction() errors.E {
	return errors.WithDetails(
		errors.WithStack(errors.New("deep nested error occurred")),
		"level", "three",
		"function", "levelThreeFunction",
		"operation", "data_processing",
		"file_path", "/tmp/data.json",
		"line_number", 42,
	)
}
