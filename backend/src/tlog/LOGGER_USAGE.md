# TLog Package - Logger Struct Usage

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Basic Usage](#basic-usage)
  - [Creating Logger Instances](#creating-logger-instances)
  - [Logger Struct Features](#logger-struct-features)
  - [Available Methods](#available-methods)
  - [Callback Integration](#callback-integration)
  - [Advanced Usage](#advanced-usage)
  - [Migration from Package Functions](#migration-from-package-functions)
- [Constructor Functions](#constructor-functions)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

The tlog package now includes a `Logger` struct that wraps `*slog.Logger` with additional tlog functionality.

## Basic Usage

### Creating Logger Instances

```go
package main

import (
    "context"
    "log/slog"
    "github.com/dianlight/srat/tlog"
)

func main() {
    // Create a logger with default configuration
    logger := tlog.NewLogger()

    // Create a logger with a specific minimum level
    debugLogger := tlog.NewLoggerWithLevel(tlog.LevelDebug)

    // Use the logger methods
    logger.Info("Application started")
    logger.Error("Something went wrong", "error", "connection failed")

    // Use context-aware logging
    ctx := context.Background()
    logger.InfoContext(ctx, "Processing request", "request_id", "abc123")
}
```

### Logger Struct Features

The `Logger` struct provides:

1. **Embedded slog.Logger**: Access to all standard slog functionality
2. **Custom Log Levels**: Support for Trace, Notice, and Fatal levels
3. **Callback System**: Automatic event emission to registered callbacks
4. **Context Support**: Both regular and context-aware logging methods

### Available Methods

```go
// Standard logging methods
logger.Trace("trace message")
logger.Debug("debug message")
logger.Info("info message")
logger.Notice("notice message")
logger.Warn("warning message")
logger.Error("error message")
logger.Fatal("fatal message") // exits program

// Context-aware versions
logger.TraceContext(ctx, "trace with context")
logger.DebugContext(ctx, "debug with context")
logger.InfoContext(ctx, "info with context")
logger.NoticeContext(ctx, "notice with context")
logger.WarnContext(ctx, "warning with context")
logger.ErrorContext(ctx, "error with context")
logger.FatalContext(ctx, "fatal with context") // exits program
```

### Callback Integration

```go
// Register a callback for error events
callbackID := tlog.RegisterCallback(tlog.LevelError, func(event tlog.LogEvent) {
    // Handle error events
    fmt.Printf("Error occurred: %s at %s\n", event.Message, event.Timestamp)
})

// Log an error - this will trigger the callback
logger.Error("Database connection failed", "host", "localhost")

// Unregister when done
tlog.UnregisterCallback(tlog.LevelError, callbackID)
```

### Advanced Usage

Since `Logger` embeds `*slog.Logger`, you can use all slog methods:

```go
// Use slog methods directly
structuredLogger := logger.With("component", "auth", "version", "1.0.0")
structuredLogger.Info("User authenticated", "user_id", 12345)

// Access the underlying slog.Logger
logger.Logger.Log(ctx, slog.LevelWarn, "Direct slog usage")

// Create derived loggers
childLogger := logger.WithGroup("database")
childLogger.Info("Query executed", "duration", "50ms")
```

### Migration from Package Functions

The Logger struct provides instance methods that correspond to the package-level functions:

```go
// Old package-level usage
tlog.Info("message")
tlog.Error("error message")

// New Logger struct usage
logger := tlog.NewLogger()
logger.Info("message")
logger.Error("error message")
```

Both approaches work and emit events to the same callback system.

## Constructor Functions

- `NewLogger()`: Creates a logger with the default configuration
- `NewLoggerWithLevel(level)`: Creates a logger with a specific minimum level
- `WithLevel(level)`: Creates a raw slog.Logger with a specific level (existing function)
