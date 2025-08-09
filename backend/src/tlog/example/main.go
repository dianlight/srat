package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dianlight/srat/tlog"
	"gitlab.com/tozd/go/errors"
)

func main() {
	fmt.Println("=== TLog Package Demonstration ===")
	fmt.Println()

	// Set initial level to INFO
	tlog.SetLevel(tlog.LevelTrace)

	// Demonstrate all logging functions
	fmt.Println()
	fmt.Println("1. All logging functions:")
	tlog.Trace("[TLOG] This trace message")
	tlog.Debug("[TLOG] This debug message")
	slog.Debug("[SLOG] This debug message")
	tlog.Info("[TLOG] This is an info message", "component", "demo")
	slog.Info("[SLOG] This is an info message", "component", "demo")
	tlog.Notice("[TLOG] This is a notice message", "action", "demonstration")
	tlog.Warn("[TLOG] This is a warning message", "issue", "example")
	slog.Warn("[SLOG] This is a warning message", "issue", "example")
	tlog.Error("[TLOG] This is an error message", "error", "demonstration error")
	slog.Error("[SLOG] This is an error message", "error", "demonstration error")
	// Demonstrate fatal without stopping the demo: wrap in recover to ignore the panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("[demo] Ignored panic from tlog.Fatal: %v\n", r)
			}
		}()
		tlog.Fatal("[TLOG] This is a fatal message, will exit the program")
	}()

	// Demonstrate context logging
	fmt.Println()
	fmt.Println("2. Context Logging:") // FIXME: context is not exposed in log!
	tlog.SetLevel(tlog.LevelTrace)

	ctx := context.WithValue(context.Background(), "requestId", "demo-123")
	ctx = context.WithValue(ctx, "userId", "user-456")

	tlog.TraceContext(ctx, "[TLOG] Processing request", "operation", "demonstration")
	tlog.DebugContext(ctx, "[TLOG] Debug information", "step", 1)
	slog.DebugContext(ctx, "[SLOG] Debug information", "step", 1)
	tlog.InfoContext(ctx, "[TLOG] Request processed", "duration", time.Millisecond*150)
	slog.InfoContext(ctx, "[SLOG] Request processed", "duration", time.Millisecond*150)

	fmt.Println()
	fmt.Println("3. Basic Error with Stack Trace:")
	stdErr0 := fmt.Errorf("[TLOG] standard error: %s", "something went wrong")
	stdErr1 := fmt.Errorf("[SLOG] standard error: %s", "something went wrong")
	tlog.Error("[TLOG] Basic error demonstration (oneline)", "error", stdErr0)
	slog.Error("[SLOG] Basic error demonstration (oneline)", "error", stdErr1)

	fmt.Println()
	fmt.Println("4. Basic Tozd Error with Stack Trace:")
	basicErr := errors.WithStack(errors.New("basic connection error"))
	tlog.Error("[TLOG] Basic tozd error demonstration (oneline)", "error", basicErr)
	slog.Error("[SLOG] Basic tozd error demonstration (oneline)", "error", basicErr)

	fmt.Println()
	fmt.Println("5. Tozd Error with Details:")
	detailedErr := errors.WithDetails(
		errors.New("database connection failed"),
		"host", "localhost",
		"port", 5432,
		"database", "user_db",
		"timeout", "30s",
		"retry_count", 3,
	)
	tlog.Error("[TLOG] Detailed tozd error", "error", detailedErr)
	slog.Error("[SLOG] Detailed tozd error", "error", detailedErr)

	fmt.Println()
	fmt.Println("5. Chained Tozd Errors:")
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

	tlog.Error("[TLOG] Complex error chain", "error", stackErr)
	slog.Error("[SLOG] Complex error chain", "error", stackErr)

	fmt.Println()
	fmt.Println("6. Nested Function Call Stack:")
	nestedErr := createDeepError()
	tlog.Error("[TLOG] Error from nested function calls", "error", nestedErr)
	slog.Error("[SLOG] Error from nested function calls", "error", nestedErr)

	fmt.Println()
	fmt.Println("7. Error with Context Values:")
	ctx = context.WithValue(context.Background(), "request_id", "demo-req-789")
	ctx = context.WithValue(ctx, "user_id", "demo-user-456")
	ctx = context.WithValue(ctx, "session_id", "session-xyz")

	contextErr := errors.WithDetails(
		errors.WithStack(errors.New("permission denied")),
		"resource", "/api/users/secret",
		"required_role", "admin",
		"user_role", "user",
	)
	tlog.ErrorContext(ctx, "[TLOG] Authorization error with context", "error", contextErr)
	slog.ErrorContext(ctx, "[SLOG] Authorization error with context", "error", contextErr)

	fmt.Println()
	fmt.Println("8. Joined Errors:")
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
	tlog.Error("[TLOG] Multiple validation errors", "error", joinedErr)
	slog.Error("[SLOG] Multiple validation errors", "error", joinedErr)

	// Demonstrate time format configuration
	fmt.Println()
	fmt.Println("9. Custom Time Format:")
	originalFormat := tlog.GetTimeFormat()

	tlog.Info("[TLOG] Log with default time format", "timestamp", time.Now())
	slog.Info("[SLOG] Log with default time format", "timestamp", time.Now())

	tlog.SetTimeFormat("2006-01-02 15:04:05")
	tlog.Info("[TLOG] Log with custom time format", "timestamp", time.Now())
	slog.Info("[SLOG] Log with custom time format", "timestamp", time.Now())

	// Restore original format
	tlog.SetTimeFormat(originalFormat)
	tlog.Info("[TLOG] Log with restored time format", "timestamp", time.Now())
	slog.Info("[SLOG] Log with restored time format", "timestamp", time.Now())

	// Demonstrate sensitive data hiding
	fmt.Println()
	fmt.Println("10. Sensitive Data Handling:")

	// First without hiding
	fmt.Println("   Without sensitive data hiding:")
	tlog.Info("[TLOG] User authentication",
		"username", "alice",
		"password", "secret123",
		"token", "jwt-token-xyz",
		"api_key", "sk-1234567890",
		"ip", "192.168.1.100")
	slog.Info("[SLOG] User authentication",
		"username", "alice",
		"password", "secret123",
		"token", "jwt-token-xyz",
		"api_key", "sk-1234567890",
		"ip", "192.168.1.100")

	// Enable sensitive data hiding
	tlog.EnableSensitiveDataHiding(true)
	fmt.Println("   With sensitive data hiding enabled:")
	tlog.Info("[TLOG] User authentication",
		"username", "alice",
		"password", "secret123",
		"token", "jwt-token-xyz",
		"api_key", "sk-1234567890",
		"ip", "192.168.1.100")
	slog.Info("[SLOG] User authentication",
		"username", "alice",
		"password", "secret123",
		"token", "jwt-token-xyz",
		"api_key", "sk-1234567890",
		"ip", "192.168.1.100")

	// Disable sensitive data hiding
	tlog.EnableSensitiveDataHiding(false)

	// Demonstrate context logging with enhanced formatting
	fmt.Println()
	fmt.Println("11. Context-Aware Logging with Formatting:")
	ctx = context.WithValue(context.Background(), "request_id", "req-12345")
	ctx = context.WithValue(ctx, "user_id", "user-456")
	ctx = context.WithValue(ctx, "trace_id", "trace-abc-xyz")

	tlog.InfoContext(ctx, "[TLOG] Processing request",
		"method", "GET",
		"path", "/api/users",
		"user_agent", "Mozilla/5.0")
	slog.InfoContext(ctx, "[SLOG] Processing request",
		"method", "GET",
		"path", "/api/users",
		"user_agent", "Mozilla/5.0")

	tlog.ErrorContext(ctx, "[TLOG] Request failed",
		"error", "database connection timeout",
		"duration", "30s",
		"retry_count", 3)
	slog.ErrorContext(ctx, "[SLOG] Request failed",
		"error", "database connection timeout",
		"duration", "30s",
		"retry_count", 3)

	currentTimestamp := time.Now().Unix()
	tlog.Info("[TLOG] Unix timestamp formatting",
		"timestamp", currentTimestamp,
		"created_at", int64(1609459200), // Jan 1, 2021
		"updated_at", "1640995200") // Jan 1, 2022 as string
	slog.Info("[SLOG] Unix timestamp formatting",
		"timestamp", currentTimestamp,
		"created_at", int64(1609459200), // Jan 1, 2021
		"updated_at", "1640995200") // Jan 1, 2022 as string

	// Simulate HTTP request/response data
	tlog.Info("[TLOG] HTTP request processed",
		"client_ip", "192.168.1.50",
		"remote_addr", "10.0.0.100",
		"auth_token", "Bearer jwt-abc-123-xyz",
		"api_key", "sk-test-key-12345",
		"private_key", "-----BEGIN RSA PRIVATE KEY-----")
	slog.Info("[SLOG] HTTP request processed",
		"client_ip", "192.168.1.50",
		"remote_addr", "10.0.0.100",
		"auth_token", "Bearer jwt-abc-123-xyz",
		"api_key", "sk-test-key-12345",
		"private_key", "-----BEGIN RSA PRIVATE KEY-----")

	// Demonstrate custom formatter configuration
	fmt.Println()
	fmt.Println("12. Custom Formatter Configuration:")

	customConfig := tlog.FormatterConfig{
		EnableColors:      true,
		EnableFormatting:  true,
		HideSensitiveData: true,
		TimeFormat:        "15:04:05.000", // Short time format
	}
	tlog.SetFormatterConfig(customConfig)

	tlog.Info("[TLOG] Message with custom config", "test", "formatting")
	slog.Info("[SLOG] Message with custom config", "test", "formatting")
	tlog.Warn("[TLOG] Warning with custom config", "password", "hidden-secret")
	slog.Warn("[SLOG] Warning with custom config", "password", "hidden-secret")

	fmt.Println("Formatter demonstration complete.")
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
