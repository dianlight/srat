# TLog Package

The `tlog` package provides an enhanced logging system built on top of Go's `log/slog` package with additional custom log levels and improved functionality.

## Features

- **Custom Log Levels**: Extends slog with `TRACE`, `NOTICE`, and `FATAL` levels
- **Thread-Safe**: All operations are protected by mutexes for concurrent access
- **Context Support**: All logging functions have context-aware variants
- **Case-Insensitive Configuration**: Log level strings are handled case-insensitively
- **Flexible Level Management**: Support for both programmatic and string-based level setting
- **Better Error Messages**: Descriptive error messages with supported level listings
- **Terminal Detection**: Automatically enables/disables colors based on terminal capabilities
- **Event Callbacks**: Asynchronous callback system for log events with panic recovery and queuing

## Log Levels

The package supports the following log levels (from lowest to highest priority):

- **TRACE** (-8): Most verbose logging for detailed execution flow
- **DEBUG** (-4): Debug information for troubleshooting
- **INFO** (0): General information messages
- **NOTICE** (2): Important but normal events
- **WARN** (4): Warning messages for potentially harmful situations
- **ERROR** (8): Error messages that don't halt execution
- **FATAL** (12): Critical errors that cause program termination

## Basic Usage

### Simple Logging

```go
import "github.com/dianlight/srat/tlog"

// Basic logging functions
tlog.Trace("Detailed execution trace", "step", 1)
tlog.Debug("Debug information", "value", someValue)
tlog.Info("Application started", "version", "1.0.0")
tlog.Notice("Configuration loaded", "config", "production")
tlog.Warn("Deprecated function used", "function", "oldFunc")
tlog.Error("Failed to connect", "host", "example.com", "error", err)
tlog.Fatal("Critical system failure") // This will exit the program
```

### Context-Aware Logging

```go
ctx := context.WithValue(context.Background(), "requestId", "req-123")

tlog.TraceContext(ctx, "Processing request")
tlog.DebugContext(ctx, "Database query executed", "query", sql)
tlog.InfoContext(ctx, "Request completed", "duration", duration)
tlog.NoticeContext(ctx, "Cache miss", "key", cacheKey)
tlog.WarnContext(ctx, "Rate limit approaching", "current", current, "limit", limit)
tlog.ErrorContext(ctx, "Validation failed", "field", "email", "value", email)
tlog.FatalContext(ctx, "Database connection lost") // This will exit the program
```

## Level Management

### Setting Log Levels

```go
// Set level programmatically
tlog.SetLevel(tlog.LevelDebug)

// Set level from string (case-insensitive)
err := tlog.SetLevelFromString("debug")
if err != nil {
    log.Fatal(err)
}

// These are all equivalent:
tlog.SetLevelFromString("debug")
tlog.SetLevelFromString("DEBUG")
tlog.SetLevelFromString("Debug")
tlog.SetLevelFromString("  debug  ") // whitespace is trimmed
```

### Getting Current Level

```go
// Get current level as slog.Level
level := tlog.GetLevel()

// Get current level as string
levelStr := tlog.GetLevelString() // Returns "DEBUG", "INFO", etc.

// Check if a specific level is enabled
if tlog.IsLevelEnabled(tlog.LevelDebug) {
    // Expensive debug operation here
    tlog.Debug("Debug info", "data", expensiveOperation())
}
```

### Supported Level Strings

The following strings are supported for `SetLevelFromString()`:

- `trace`, `debug`, `info`, `notice`, `warn`, `warning`, `error`, `fatal`
- All strings are case-insensitive
- `warning` is an alias for `warn`
- Leading and trailing whitespace is automatically trimmed

## Advanced Usage

### Event Callbacks

The tlog package supports registering callback functions that are executed asynchronously whenever a log of a specific level is generated. This is useful for implementing custom log handlers, monitoring systems, alerting, or audit logging.

#### Registering Callbacks

```go
// Register a callback for error-level logs
errorCallbackID := tlog.RegisterCallback(tlog.LevelError, func(event tlog.LogEvent) {
    // Send to monitoring system
    sendToMonitoring(event.Message, event.Level, event.Args)
    
    // Or send email alert
    if event.Level >= tlog.LevelError {
        sendEmailAlert(event.Message, event.Timestamp)
    }
})

// Register a callback for all info-level logs
infoCallbackID := tlog.RegisterCallback(tlog.LevelInfo, func(event tlog.LogEvent) {
    // Write to audit log
    auditLogger.Log(event.Context, event.Level, event.Message, event.Args...)
})
```

#### LogEvent Structure

The `LogEvent` passed to callbacks contains:

```go
type LogEvent struct {
    Level     slog.Level    // Log level (LevelError, LevelInfo, etc.)
    Message   string        // Log message
    Args      []any         // Key-value pairs passed to the log function
    Timestamp time.Time     // When the log was generated
    Context   context.Context // Context passed to logging function
}
```

#### Managing Callbacks

```go
// Unregister a specific callback
success := tlog.UnregisterCallback(tlog.LevelError, errorCallbackID)

// Clear all callbacks for a specific level
tlog.ClearCallbacks(tlog.LevelError)

// Clear all callbacks for all levels
tlog.ClearAllCallbacks()

// Get callback count for debugging
count := tlog.GetCallbackCount(tlog.LevelError)
fmt.Printf("Number of error callbacks: %d\n", count)
```

#### Callback Safety Features

- **Async Execution**: Callbacks are executed in separate goroutines and won't block logging
- **Panic Recovery**: If a callback panics, it's recovered and logged without affecting the main program
- **Error Isolation**: Failed callbacks don't affect other callbacks or normal logging
- **Buffered Queue**: Events are queued (buffer size: 1000) for processing
- **Non-blocking**: If the queue is full, events are dropped to prevent blocking

#### Example: Monitoring Integration

```go
// Set up error monitoring
tlog.RegisterCallback(tlog.LevelError, func(event tlog.LogEvent) {
    // Extract error information
    var errorMsg string
    var errorCode int
    
    for i := 0; i < len(event.Args); i += 2 {
        if i+1 < len(event.Args) {
            key := fmt.Sprintf("%v", event.Args[i])
            value := event.Args[i+1]
            
            switch key {
            case "error":
                errorMsg = fmt.Sprintf("%v", value)
            case "code":
                if code, ok := value.(int); ok {
                    errorCode = code
                }
            }
        }
    }
    
    // Send to monitoring service
    monitoring.RecordError(monitoring.ErrorEvent{
        Message:   event.Message,
        Error:     errorMsg,
        Code:      errorCode,
        Timestamp: event.Timestamp,
        Level:     event.Level.String(),
    })
})

// Now any error log will trigger monitoring
tlog.Error("Database connection failed", "error", err, "code", 500)
```

#### Graceful Shutdown

When your application shuts down, ensure callbacks are processed:

```go
// Process remaining events and shut down gracefully
defer tlog.Shutdown()
```

### Custom Logger Instances

Create a logger with a specific minimum level without affecting the global logger:

```go
// Create a trace-level logger for detailed debugging
debugLogger := tlog.WithLevel(tlog.LevelTrace)

// Use the custom logger
debugLogger.Log(context.Background(), tlog.LevelTrace, "Detailed trace info")
```

### Error Handling

The package provides descriptive error messages:

```go
err := tlog.SetLevelFromString("invalid")
if err != nil {
    fmt.Println(err)
    // Output: invalid log level 'invalid': supported levels are trace, debug, info, notice, warn, warning, error, fatal
}

err = tlog.SetLevelFromString("")
if err != nil {
    fmt.Println(err)
    // Output: log level cannot be empty
}
```

## Thread Safety

All tlog operations are thread-safe:

```go
// Safe to call concurrently from multiple goroutines
go func() {
    tlog.SetLevel(tlog.LevelDebug)
    tlog.Debug("Debug from goroutine 1")
}()

go func() {
    level := tlog.GetLevel()
    tlog.Info("Current level", "level", level)
}()
```

## Configuration

The package automatically configures itself on initialization:

- **Terminal Detection**: Colors are automatically enabled/disabled based on whether output is a terminal
- **Source Location**: File and line information is included in log output
- **Time Format**: Uses RFC3339 format for timestamps
- **Output**: Logs are written to stderr

## Best Practices

1. **Use appropriate levels**: Use TRACE for very detailed information, DEBUG for troubleshooting, INFO for general information, NOTICE for important events, WARN for potential issues, ERROR for actual problems, and FATAL only for critical failures.

2. **Leverage context**: Use the context variants (`InfoContext`, `ErrorContext`, etc.) when you have relevant context information like request IDs or user information.

3. **Check level enablement**: For expensive operations, check if the level is enabled before performing the work:
   ```go
   if tlog.IsLevelEnabled(tlog.LevelDebug) {
       expensive := doExpensiveCalculation()
       tlog.Debug("Calculation result", "result", expensive)
   }
   ```

4. **Use structured logging**: Prefer key-value pairs over string formatting:
   ```go
   // Good
   tlog.Info("User logged in", "userId", user.ID, "email", user.Email)
   
   // Less ideal
   tlog.Info(fmt.Sprintf("User %s (%d) logged in", user.Email, user.ID))
   ```

5. **Handle level setting errors**: Always check errors when setting levels from strings:
   ```go
   if err := tlog.SetLevelFromString(configLevel); err != nil {
       tlog.Error("Invalid log level in config", "level", configLevel, "error", err)
       tlog.SetLevel(tlog.LevelInfo) // fallback to reasonable default
   }
   ```

6. **Use callbacks judiciously**: Register callbacks only for levels that need special handling. Keep callback functions lightweight to avoid impacting performance:
   ```go
   // Good - lightweight callback
   tlog.RegisterCallback(tlog.LevelError, func(event tlog.LogEvent) {
       errorCounter.Inc()
       errorQueue <- event
   })
   
   // Avoid - heavy operations in callbacks
   tlog.RegisterCallback(tlog.LevelInfo, func(event tlog.LogEvent) {
       // Don't do expensive operations here
       sendEmailNotification(event) // This could block
   })
   ```

7. **Clean up callbacks**: Remember to unregister callbacks or clear them when they're no longer needed:
   ```go
   callbackID := tlog.RegisterCallback(tlog.LevelError, errorHandler)
   defer tlog.UnregisterCallback(tlog.LevelError, callbackID)
   ```

## Migration from Previous Version

The improved tlog package is fully backward compatible. Existing code will continue to work without changes. However, you can take advantage of new features:

- Replace `tlog.Info("message")` with `tlog.InfoContext(ctx, "message")` when context is available
- Use `tlog.IsLevelEnabled()` for expensive debug operations
- Take advantage of better error messages in level configuration code
- Register callbacks for critical log levels to implement monitoring and alerting:
  ```go
  // Add monitoring for errors
  tlog.RegisterCallback(tlog.LevelError, func(event tlog.LogEvent) {
      monitoring.RecordError(event.Message, event.Args)
  })
  ```

## API Reference

### Callback Functions

- `RegisterCallback(level slog.Level, callback LogCallback) string` - Register a callback for a log level, returns callback ID
- `UnregisterCallback(level slog.Level, callbackID string) bool` - Remove a specific callback by ID
- `ClearCallbacks(level slog.Level)` - Remove all callbacks for a log level
- `ClearAllCallbacks()` - Remove all callbacks for all levels
- `GetCallbackCount(level slog.Level) int` - Get the number of registered callbacks for a level
- `Shutdown()` - Gracefully shutdown the callback processor
- `RestartProcessor()` - Restart the callback processor (mainly for testing)

### LogCallback Type

```go
type LogCallback func(event LogEvent)

type LogEvent struct {
    Level     slog.Level
    Message   string
    Args      []any
    Timestamp time.Time
    Context   context.Context
}
```
