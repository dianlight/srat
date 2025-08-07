package main

import (
	"context"
	"fmt"
	"time"

	"github.com/dianlight/srat/tlog"
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
}

func performExpensiveOperation() string {
	// Simulate expensive operation
	time.Sleep(time.Millisecond * 100)
	return "expensive_result"
}
