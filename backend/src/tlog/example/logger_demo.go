package main

import (
	"context"
	"log/slog"

	"github.com/dianlight/srat/tlog"
)

// ExampleLogger demonstrates how to use the new Logger struct
func ExampleLogger() {
	// Create a new logger with default configuration
	logger := tlog.NewLogger()

	// Create a logger with a specific level
	debugLogger := tlog.NewLoggerWithLevel(tlog.LevelDebug)

	// Use the logger just like slog.Logger (embedded methods)
	logger.Info("This is an info message")
	logger.Error("This is an error message")

	// Use context-aware logging
	ctx := context.Background()
	logger.InfoContext(ctx, "This is an info message with context")

	// Use custom log levels
	logger.Trace("This is a trace message")
	logger.Notice("This is a notice message")

	// The debug logger will show debug messages since we set the level
	debugLogger.Debug("This debug message will be visible")
	debugLogger.Trace("This trace message will also be visible")

	// Register a callback to handle error events
	callbackID := tlog.RegisterCallback(tlog.LevelError, func(event tlog.LogEvent) {
		// Handle error events (e.g., send notifications, write to database)
		println("Error callback triggered:", event.Message)
	})

	// Log an error - this will trigger the callback
	logger.Error("Something went wrong", "code", 500)

	// Unregister the callback when done
	tlog.UnregisterCallback(tlog.LevelError, callbackID)

	// Since Logger embeds *slog.Logger, you can use all slog methods
	logger.With("component", "auth").Info("User logged in", "user_id", 123)

	// You can also use the underlying Logger field directly
	logger.Logger.Log(ctx, slog.LevelWarn, "Direct slog usage")
}
