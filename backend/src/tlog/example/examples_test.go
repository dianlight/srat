package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dianlight/srat/tlog"
)

func ExampleNewLogger() {
	// Create a logger with default configuration
	logger := tlog.NewLogger()
	logger.Info("Application started")
}

func ExampleNewLoggerWithLevel() {
	// Create a logger with debug level enabled
	debugLogger := tlog.NewLoggerWithLevel(tlog.LevelDebug)
	debugLogger.Debug("Debug information")
	debugLogger.Info("Application ready")
}

func ExampleLogger_contextMethods() {
	logger := tlog.NewLogger()
	ctx := context.Background()

	// Use context-aware logging methods
	logger.InfoContext(ctx, "Processing request", "request_id", "abc123")
	logger.ErrorContext(ctx, "Request failed", "error", "timeout")
}

func ExampleLogger_callbacks() {
	logger := tlog.NewLogger()

	// Register a callback for error events
	callbackID := tlog.RegisterCallback(tlog.LevelError, func(event tlog.LogEvent) {
		fmt.Printf("Error callback: %s\n", event.Message)
	})

	// This will trigger the callback
	logger.Error("Database connection failed")

	// Wait for callback to execute (callbacks are asynchronous)
	time.Sleep(10 * time.Millisecond)

	// Clean up
	tlog.UnregisterCallback(tlog.LevelError, callbackID)
	// Output: Error callback: Database connection failed
}

func ExampleLogger_embeddedSlog() {
	logger := tlog.NewLogger()

	// Use embedded slog.Logger functionality
	structuredLogger := logger.With("component", "auth", "version", "1.0.0")
	structuredLogger.Info("User authenticated", "user_id", 12345)

	// Create grouped logger
	dbLogger := logger.WithGroup("database")
	dbLogger.Info("Query executed", "duration", "45ms")

	// Direct access to underlying slog.Logger
	logger.Logger.Log(context.Background(), slog.LevelWarn, "Direct slog usage")
}
