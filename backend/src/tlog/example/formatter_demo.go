package main

import (
	"context"
	"fmt"
	"time"

	"github.com/dianlight/srat/tlog"
)

func formatterColorDemo() {
	fmt.Println("\n=== TLog Enhanced Formatter Demonstration ===")
	fmt.Println()

	// Demonstrate current formatter configuration
	fmt.Println("1. Current Formatter Configuration:")
	config := tlog.GetFormatterConfig()
	fmt.Printf("  - Colors Enabled: %v\n", config.EnableColors)
	fmt.Printf("  - Formatting Enabled: %v\n", config.EnableFormatting)
	fmt.Printf("  - Hide Sensitive Data: %v\n", config.HideSensitiveData)
	fmt.Printf("  - Time Format: %s\n", config.TimeFormat)
	fmt.Printf("  - Terminal Colors Supported: %v\n", tlog.IsColorsEnabled())

	// Demonstrate color printing functions
	fmt.Println()
	fmt.Println("2. Color-Enhanced Printing:")
	tlog.ColorTrace("This is a TRACE message with color\n")
	tlog.ColorDebug("This is a DEBUG message with color\n")
	tlog.ColorInfo("This is an INFO message with color\n")
	tlog.ColorNotice("This is a NOTICE message with color\n")
	tlog.ColorWarn("This is a WARN message with color\n")
	tlog.ColorError("This is an ERROR message with color\n")

	fmt.Println()
	fmt.Println("3. Messages with Level Prefixes (All Levels):")
	tlog.PrintWithLevelAll("Sample message for all levels")

	fmt.Println()
	fmt.Println("3b. Individual Level Prefix Examples:")
	tlog.PrintWithLevel(tlog.LevelTrace, "TRACE level with colored prefix only\n")
	tlog.PrintWithLevel(tlog.LevelDebug, "DEBUG level with colored prefix only\n")
	tlog.PrintWithLevel(tlog.LevelInfo, "INFO level with colored prefix only\n")
	tlog.PrintWithLevel(tlog.LevelNotice, "NOTICE level with colored prefix only\n")
	tlog.PrintWithLevel(tlog.LevelWarn, "WARN level with full message colored\n")
	tlog.PrintWithLevel(tlog.LevelError, "ERROR level with full message colored\n")
	tlog.PrintWithLevel(tlog.LevelFatal, "FATAL level with full message colored\n")

	// Demonstrate formatted printing with arguments
	fmt.Println()
	fmt.Println("4. Formatted Color Messages:")
	tlog.ColorPrint(tlog.LevelInfo, "User %s logged in at %s\n", "alice", time.Now().Format("15:04:05"))
	tlog.ColorPrintln(tlog.LevelWarn, "Connection timeout after %d seconds", 30)
	tlog.ColorPrintln(tlog.LevelError, "Failed to process %d out of %d items", 5, 100)

	// Demonstrate time format configuration
	fmt.Println()
	fmt.Println("5. Custom Time Format:")
	originalFormat := tlog.GetTimeFormat()

	tlog.Info("Log with default time format", "timestamp", time.Now())

	tlog.SetTimeFormat("2006-01-02 15:04:05")
	tlog.Info("Log with custom time format", "timestamp", time.Now())

	// Restore original format
	tlog.SetTimeFormat(originalFormat)
	tlog.Info("Log with restored time format", "timestamp", time.Now())

	// Demonstrate sensitive data hiding
	fmt.Println()
	fmt.Println("6. Sensitive Data Handling:")

	// First without hiding
	fmt.Println("   Without sensitive data hiding:")
	tlog.Info("User authentication",
		"username", "alice",
		"password", "secret123",
		"token", "jwt-token-xyz",
		"api_key", "sk-1234567890",
		"ip", "192.168.1.100")

	// Enable sensitive data hiding
	tlog.EnableSensitiveDataHiding(true)
	fmt.Println("   With sensitive data hiding enabled:")
	tlog.Info("User authentication",
		"username", "alice",
		"password", "secret123",
		"token", "jwt-token-xyz",
		"api_key", "sk-1234567890",
		"ip", "192.168.1.100")

	// Disable sensitive data hiding
	tlog.EnableSensitiveDataHiding(false)

	// Demonstrate enhanced logger instances
	fmt.Println()
	fmt.Println("7. Enhanced Logger Instances:")

	// Create logger with specific level
	debugLogger := tlog.WithLevel(tlog.LevelDebug)
	debugLogger.Debug("Debug message from custom logger")
	debugLogger.Info("Info message from custom logger", "source", "custom-logger")

	// Create standard logger instances
	logger1 := tlog.NewLogger()
	logger2 := tlog.NewLoggerWithLevel(tlog.LevelWarn)

	logger1.Info("Message from standard logger")
	logger2.Warn("Warning from warn-level logger", "min_level", "WARN")

	// Demonstrate context logging with enhanced formatting
	fmt.Println()
	fmt.Println("8. Context-Aware Logging with Formatting:")
	ctx := context.WithValue(context.Background(), "request_id", "req-12345")
	ctx = context.WithValue(ctx, "user_id", "user-456")
	ctx = context.WithValue(ctx, "trace_id", "trace-abc-xyz")

	tlog.InfoContext(ctx, "Processing request",
		"method", "GET",
		"path", "/api/users",
		"user_agent", "Mozilla/5.0")

	tlog.ErrorContext(ctx, "Request failed",
		"error", "database connection timeout",
		"duration", "30s",
		"retry_count", 3)

	// Demonstrate additional formatters
	fmt.Println()
	fmt.Println("8b. Additional Formatters (Unix Timestamps, HTTP data):")

	// Enable sensitive data hiding for this demo
	tlog.EnableSensitiveDataHiding(true)

	currentTimestamp := time.Now().Unix()
	tlog.Info("Unix timestamp formatting",
		"timestamp", currentTimestamp,
		"created_at", int64(1609459200), // Jan 1, 2021
		"updated_at", "1640995200") // Jan 1, 2022 as string

	// Simulate HTTP request/response data
	tlog.Info("HTTP request processed",
		"client_ip", "192.168.1.50",
		"remote_addr", "10.0.0.100",
		"auth_token", "Bearer jwt-abc-123-xyz",
		"api_key", "sk-test-key-12345",
		"private_key", "-----BEGIN RSA PRIVATE KEY-----")

	// Restore original sensitive data setting
	tlog.EnableSensitiveDataHiding(false)

	// Demonstrate custom formatter configuration
	fmt.Println()
	fmt.Println("9. Custom Formatter Configuration:")

	customConfig := tlog.FormatterConfig{
		EnableColors:      true,
		EnableFormatting:  true,
		HideSensitiveData: true,
		TimeFormat:        "15:04:05.000", // Short time format
	}

	fmt.Println("   Applying custom configuration...")
	tlog.SetFormatterConfig(customConfig)

	tlog.Info("Message with custom config", "test", "formatting")
	tlog.Warn("Warning with custom config", "password", "hidden-secret")

	// Restore original configuration
	fmt.Println("   Restoring original configuration...")
	originalConfig := tlog.FormatterConfig{
		EnableColors:      config.EnableColors,
		EnableFormatting:  config.EnableFormatting,
		HideSensitiveData: config.HideSensitiveData,
		TimeFormat:        config.TimeFormat,
	}
	tlog.SetFormatterConfig(originalConfig)

	tlog.Info("Message with restored config", "status", "complete")

	fmt.Println()
	fmt.Println("=== End of Formatter Demonstration ===")
}
