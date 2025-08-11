package tlog_test

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dianlight/srat/tlog"
	"github.com/stretchr/testify/suite"
)

type TlogSuite struct {
	suite.Suite
	originalLevel slog.Level
}

func (suite *TlogSuite) SetupTest() {
	// Store the original level to restore it after each test
	suite.originalLevel = tlog.GetLevel()
	// Ensure a clean restart of the processor for each test
	tlog.RestartProcessor()
}

func (suite *TlogSuite) TearDownTest() {
	// Restore the original level
	tlog.SetLevel(suite.originalLevel)
	// Clear callbacks after each test
	tlog.ClearAllCallbacks()
}

func (suite *TlogSuite) TestSetAndGetLevel() {
	// Test setting and getting various levels
	levels := []slog.Level{
		tlog.LevelTrace,
		tlog.LevelDebug,
		tlog.LevelInfo,
		tlog.LevelNotice,
		tlog.LevelWarn,
		tlog.LevelError,
		tlog.LevelFatal,
	}

	for _, level := range levels {
		tlog.SetLevel(level)
		suite.Equal(level, tlog.GetLevel())
	}
}

func (suite *TlogSuite) TestSetLevelFromString() {
	testCases := []struct {
		input         string
		expectedLevel slog.Level
		shouldError   bool
	}{
		{"trace", tlog.LevelTrace, false},
		{"debug", tlog.LevelDebug, false},
		{"info", tlog.LevelInfo, false},
		{"notice", tlog.LevelNotice, false},
		{"warn", tlog.LevelWarn, false},
		{"warning", tlog.LevelWarn, false}, // alias
		{"error", tlog.LevelError, false},
		{"fatal", tlog.LevelFatal, false},

		// Case-insensitive tests
		{"TRACE", tlog.LevelTrace, false},
		{"Debug", tlog.LevelDebug, false},
		{"INFO", tlog.LevelInfo, false},
		{"Notice", tlog.LevelNotice, false},
		{"WARN", tlog.LevelWarn, false},
		{"Warning", tlog.LevelWarn, false},
		{"ERROR", tlog.LevelError, false},
		{"Fatal", tlog.LevelFatal, false},

		// With whitespace
		{"  trace  ", tlog.LevelTrace, false},
		{"\tdebug\n", tlog.LevelDebug, false},

		// Invalid cases
		{"invalid", 0, true},
		{"", 0, true},
		{"tracee", 0, true},
		{"debugg", 0, true},
	}

	for _, tc := range testCases {
		suite.Run(tc.input, func() {
			err := tlog.SetLevelFromString(tc.input)

			if tc.shouldError {
				suite.Error(err)
				if tc.input == "" {
					suite.Contains(err.Error(), "log level cannot be empty")
				} else {
					suite.Contains(err.Error(), "invalid log level")
				}
			} else {
				suite.NoError(err)
				suite.Equal(tc.expectedLevel, tlog.GetLevel())
			}
		})
	}
}

func (suite *TlogSuite) TestGetLevelString() {
	testCases := []struct {
		level          slog.Level
		expectedString string
	}{
		{tlog.LevelTrace, "TRACE"},
		{tlog.LevelDebug, "DEBUG"},
		{tlog.LevelInfo, "INFO"},
		{tlog.LevelNotice, "NOTICE"},
		{tlog.LevelWarn, "WARN"},
		{tlog.LevelError, "ERROR"},
		{tlog.LevelFatal, "FATAL"},
	}

	for _, tc := range testCases {
		suite.Run(tc.expectedString, func() {
			tlog.SetLevel(tc.level)
			result := tlog.GetLevelString()
			suite.Equal(tc.expectedString, result)
		})
	}
}

func (suite *TlogSuite) TestIsLevelEnabled() {
	// Set level to Info
	tlog.SetLevel(tlog.LevelInfo)

	// Levels that should be enabled (>= Info)
	suite.True(tlog.IsLevelEnabled(tlog.LevelInfo))
	suite.True(tlog.IsLevelEnabled(tlog.LevelNotice))
	suite.True(tlog.IsLevelEnabled(tlog.LevelWarn))
	suite.True(tlog.IsLevelEnabled(tlog.LevelError))
	suite.True(tlog.IsLevelEnabled(tlog.LevelFatal))

	// Levels that should be disabled (< Info)
	suite.False(tlog.IsLevelEnabled(tlog.LevelTrace))
	suite.False(tlog.IsLevelEnabled(tlog.LevelDebug))
}

func (suite *TlogSuite) TestWithLevel() {
	// Create a logger with Trace level
	logger := tlog.WithLevel(tlog.LevelTrace)
	suite.NotNil(logger)

	// The global level should remain unchanged
	originalLevel := tlog.GetLevel()
	tlog.SetLevel(tlog.LevelError)

	// Verify the global level changed but our custom logger is separate
	suite.Equal(tlog.LevelError, tlog.GetLevel())

	// Note: We can't directly test the custom logger's level without exposing internals
	// but we can verify it was created successfully
	suite.NotNil(logger)

	// Restore original level
	tlog.SetLevel(originalLevel)
}

func (suite *TlogSuite) TestSetLevelFromStringErrorMessages() {
	err := tlog.SetLevelFromString("")
	suite.Error(err)
	suite.Contains(err.Error(), "log level cannot be empty")

	err = tlog.SetLevelFromString("invalid")
	suite.Error(err)
	suite.Contains(err.Error(), "invalid log level 'invalid'")
	suite.Contains(err.Error(), "supported levels are")

	// Verify the error message contains expected level names (order may vary)
	supportedLevels := []string{"trace", "debug", "info", "notice", "warn", "error", "fatal"}
	for _, level := range supportedLevels {
		suite.Contains(strings.ToLower(err.Error()), level)
	}
	// Also check for the 'warning' alias
	suite.True(strings.Contains(strings.ToLower(err.Error()), "warning") || strings.Contains(strings.ToLower(err.Error()), "warn"))
}

func (suite *TlogSuite) TestCustomLevels() {
	// Test that our custom levels have the expected values
	suite.Equal(tlog.LevelTrace, slog.Level(-8))
	suite.Equal(slog.LevelDebug, tlog.LevelDebug)
	suite.Equal(slog.LevelInfo, tlog.LevelInfo)
	suite.Equal(tlog.LevelNotice, slog.Level(2))
	suite.Equal(slog.LevelWarn, tlog.LevelWarn)
	suite.Equal(slog.LevelError, tlog.LevelError)
	suite.Equal(tlog.LevelFatal, slog.Level(12))
}

func (suite *TlogSuite) TestLoggingFunctions() {
	// Test that logging functions don't panic
	// Note: We can't easily test the output without capturing logs,
	// but we can ensure the functions execute without errors

	suite.NotPanics(func() {
		tlog.Trace("test trace message", "key", "value")
	})

	suite.NotPanics(func() {
		tlog.Debug("test debug message", "key", "value")
	})

	suite.NotPanics(func() {
		tlog.Info("test info message", "key", "value")
	})

	suite.NotPanics(func() {
		tlog.Notice("test notice message", "key", "value")
	})

	suite.NotPanics(func() {
		tlog.Warn("test warn message", "key", "value")
	})

	suite.NotPanics(func() {
		tlog.Error("test error message", "key", "value")
	})
}

func (suite *TlogSuite) TestContextLoggingFunctions() {
	ctx := context.Background()

	// Test that context logging functions don't panic
	suite.NotPanics(func() {
		tlog.TraceContext(ctx, "test trace message", "key", "value")
	})

	suite.NotPanics(func() {
		tlog.DebugContext(ctx, "test debug message", "key", "value")
	})

	suite.NotPanics(func() {
		tlog.InfoContext(ctx, "test info message", "key", "value")
	})

	suite.NotPanics(func() {
		tlog.NoticeContext(ctx, "test notice message", "key", "value")
	})

	suite.NotPanics(func() {
		tlog.WarnContext(ctx, "test warn message", "key", "value")
	})

	suite.NotPanics(func() {
		tlog.ErrorContext(ctx, "test error message", "key", "value")
	})
}

func (suite *TlogSuite) TestConcurrency() {
	// Test that concurrent access to level setting/getting is safe
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(level slog.Level) {
			defer func() { done <- true }()

			tlog.SetLevel(level)
			retrievedLevel := tlog.GetLevel()
			suite.IsType(slog.Level(0), retrievedLevel)

			// Test level string conversion
			levelStr := tlog.GetLevelString()
			suite.NotEmpty(levelStr)

			// Test IsLevelEnabled
			enabled := tlog.IsLevelEnabled(level)
			suite.IsType(true, enabled)
		}(slog.Level(i - 4)) // Use various levels including negative ones
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Callback Tests

func (suite *TlogSuite) TestRegisterCallback() {
	var receivedEvent tlog.LogEvent
	var callbackExecuted bool
	var mu sync.Mutex

	callback := func(event tlog.LogEvent) {
		mu.Lock()
		defer mu.Unlock()
		receivedEvent = event
		callbackExecuted = true
	}

	// Register callback for error level
	callbackID := tlog.RegisterCallback(tlog.LevelError, callback)
	suite.NotEmpty(callbackID)

	// Verify callback count
	suite.Equal(1, tlog.GetCallbackCount(tlog.LevelError))

	// Trigger an error log
	testMessage := "test error message"
	tlog.Error(testMessage, "key", "value", "pippo", "pluto")

	// Wait for callback to execute
	suite.Eventually(func() bool {
		mu.Lock()
		defer mu.Unlock()
		return callbackExecuted
	}, time.Second*2, time.Millisecond*50)

	// Verify callback was called with correct event
	mu.Lock()
	defer mu.Unlock()
	suite.True(callbackExecuted)
	suite.Equal(tlog.LevelError, receivedEvent.Record.Level)
	suite.Equal(testMessage, receivedEvent.Record.Message)
	suite.Equal(2, receivedEvent.Record.NumAttrs())
	//	suite.Equal([]any{"key", "value"}, receivedEvent.Record.NumAttrs())
	suite.NotZero(receivedEvent.Record.Time)
	suite.NotNil(receivedEvent.Context)
}

func (suite *TlogSuite) TestRegisterCallbackForError() {
	var receivedEvent tlog.LogEvent
	var callbackExecuted bool
	var mu sync.Mutex

	callback := func(event tlog.LogEvent) {
		mu.Lock()
		defer mu.Unlock()
		receivedEvent = event
		callbackExecuted = true
	}

	// Register callback for error level
	callbackID := tlog.RegisterCallback(tlog.LevelError, callback)
	suite.NotEmpty(callbackID)

	// Verify callback count
	suite.Equal(1, tlog.GetCallbackCount(tlog.LevelError))

	// Trigger an error log
	testMessage := "test error message"
	tlog.Error(testMessage, "key", "value", "pippo", "pluto", "error", fmt.Errorf("XX test error"))

	// Wait for callback to execute
	suite.Eventually(func() bool {
		mu.Lock()
		defer mu.Unlock()
		return callbackExecuted
	}, time.Second*2, time.Millisecond*50)

	// Verify callback was called with correct event
	mu.Lock()
	defer mu.Unlock()
	suite.True(callbackExecuted)
	suite.Equal(tlog.LevelError, receivedEvent.Record.Level)
	suite.Equal(testMessage, receivedEvent.Record.Message)
	suite.Equal(3, receivedEvent.Record.NumAttrs())
	var extractedErr error
	receivedEvent.Record.Attrs(func(attr slog.Attr) bool {
		suite.T().Log("Attr:", attr.Key, "=", attr.Value.Any())
		key := strings.ToLower(attr.Key)
		if key == "error" || key == "err" {
			v := attr.Value.Any()
			switch vv := v.(type) {
			case error:
				extractedErr = vv
				return false
			case []slog.Attr:
				// When formatted, error may be represented as slice of Attrs, first usually message
				if len(vv) > 0 {
					for _, a := range vv {
						if a.Key == "org_error" {
							extractedErr = a.Value.Any().(error)
							return false
						}
					}
				}
			}
		}
		return true
	})

	suite.Error(extractedErr)
	suite.Error(extractedErr)
	suite.Equal("XX test error", extractedErr.Error(), "Extracted error should match the original")

	suite.NotZero(receivedEvent.Record.Time)
	suite.NotNil(receivedEvent.Context)
}

func (suite *TlogSuite) TestUnregisterCallback() {
	var callbackExecuted bool
	var mu sync.Mutex

	callback := func(event tlog.LogEvent) {
		mu.Lock()
		defer mu.Unlock()
		callbackExecuted = true
	}

	// Register callback
	callbackID := tlog.RegisterCallback(tlog.LevelInfo, callback)
	suite.Equal(1, tlog.GetCallbackCount(tlog.LevelInfo))

	// Unregister callback
	success := tlog.UnregisterCallback(tlog.LevelInfo, callbackID)
	suite.True(success)
	suite.Equal(0, tlog.GetCallbackCount(tlog.LevelInfo))

	// Trigger log - callback should not execute
	tlog.Info("test message")

	// Wait a bit to ensure callback doesn't execute
	time.Sleep(time.Millisecond * 100)

	mu.Lock()
	defer mu.Unlock()
	suite.False(callbackExecuted)

	// Try to unregister non-existent callback
	success = tlog.UnregisterCallback(tlog.LevelInfo, "non-existent")
	suite.False(success)
}

func (suite *TlogSuite) TestMultipleCallbacks() {
	var callback1Count, callback2Count int32

	callback1 := func(event tlog.LogEvent) {
		atomic.AddInt32(&callback1Count, 1)
	}

	callback2 := func(event tlog.LogEvent) {
		atomic.AddInt32(&callback2Count, 1)
	}

	// Register multiple callbacks for the same level
	id1 := tlog.RegisterCallback(tlog.LevelWarn, callback1)
	id2 := tlog.RegisterCallback(tlog.LevelWarn, callback2)

	suite.Equal(2, tlog.GetCallbackCount(tlog.LevelWarn))

	// Trigger warning
	tlog.Warn("test warning")

	// Wait for callbacks to execute
	suite.Eventually(func() bool {
		return atomic.LoadInt32(&callback1Count) > 0 && atomic.LoadInt32(&callback2Count) > 0
	}, time.Second*2, time.Millisecond*50)

	suite.Equal(int32(1), atomic.LoadInt32(&callback1Count))
	suite.Equal(int32(1), atomic.LoadInt32(&callback2Count))

	// Unregister one callback
	success := tlog.UnregisterCallback(tlog.LevelWarn, id1)
	suite.True(success)
	suite.Equal(1, tlog.GetCallbackCount(tlog.LevelWarn))

	// Reset counters
	atomic.StoreInt32(&callback1Count, 0)
	atomic.StoreInt32(&callback2Count, 0)

	// Trigger another warning
	tlog.Warn("another test warning")

	// Wait for callback to execute
	suite.Eventually(func() bool {
		return atomic.LoadInt32(&callback2Count) > 0
	}, time.Second*2, time.Millisecond*50)

	// Only callback2 should have executed
	suite.Equal(int32(0), atomic.LoadInt32(&callback1Count))
	suite.Equal(int32(1), atomic.LoadInt32(&callback2Count))

	// Clean up
	tlog.UnregisterCallback(tlog.LevelWarn, id2)
}

func (suite *TlogSuite) TestClearCallbacks() {
	callback := func(event tlog.LogEvent) {}

	// Register callbacks for different levels
	tlog.RegisterCallback(tlog.LevelError, callback)
	tlog.RegisterCallback(tlog.LevelWarn, callback)
	tlog.RegisterCallback(tlog.LevelInfo, callback)

	suite.Equal(1, tlog.GetCallbackCount(tlog.LevelError))
	suite.Equal(1, tlog.GetCallbackCount(tlog.LevelWarn))
	suite.Equal(1, tlog.GetCallbackCount(tlog.LevelInfo))

	// Clear callbacks for one level
	tlog.ClearCallbacks(tlog.LevelError)
	suite.Equal(0, tlog.GetCallbackCount(tlog.LevelError))
	suite.Equal(1, tlog.GetCallbackCount(tlog.LevelWarn))
	suite.Equal(1, tlog.GetCallbackCount(tlog.LevelInfo))

	// Clear all callbacks
	tlog.ClearAllCallbacks()
	suite.Equal(0, tlog.GetCallbackCount(tlog.LevelError))
	suite.Equal(0, tlog.GetCallbackCount(tlog.LevelWarn))
	suite.Equal(0, tlog.GetCallbackCount(tlog.LevelInfo))
}

func (suite *TlogSuite) TestCallbackPanicRecovery() {
	var normalCallbackExecuted bool
	var mu sync.Mutex

	// Callback that panics
	panicCallback := func(event tlog.LogEvent) {
		panic("test panic in callback")
	}

	// Normal callback
	normalCallback := func(event tlog.LogEvent) {
		mu.Lock()
		defer mu.Unlock()
		normalCallbackExecuted = true
	}

	// Register both callbacks
	tlog.RegisterCallback(tlog.LevelError, panicCallback)
	tlog.RegisterCallback(tlog.LevelError, normalCallback)

	// Trigger error log - should not crash the program
	suite.NotPanics(func() {
		tlog.Error("test error with panic callback")
	})

	// Normal callback should still execute despite panic in other callback
	suite.Eventually(func() bool {
		mu.Lock()
		defer mu.Unlock()
		return normalCallbackExecuted
	}, time.Second*2, time.Millisecond*50)
}

func (suite *TlogSuite) TestCallbackWithContextMethods() {
	var receivedEvents []tlog.LogEvent
	var mu sync.Mutex

	callback := func(event tlog.LogEvent) {
		mu.Lock()
		defer mu.Unlock()
		receivedEvents = append(receivedEvents, event)
	}

	// Register callback for debug level
	tlog.RegisterCallback(tlog.LevelDebug, callback)

	ctx := context.WithValue(context.Background(), "test-key", "test-value")

	// Test context methods
	tlog.DebugContext(ctx, "debug with context", "debug", true)

	// Wait for callback
	suite.Eventually(func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(receivedEvents) > 0
	}, time.Second*2, time.Millisecond*50)

	mu.Lock()
	defer mu.Unlock()
	suite.Len(receivedEvents, 1)
	suite.Equal(tlog.LevelDebug, receivedEvents[0].Record.Level)
	suite.Equal("debug with context", receivedEvents[0].Record.Message)
	suite.Equal(ctx, receivedEvents[0].Context)
}

func (suite *TlogSuite) TestCallbackConcurrency() {
	var callbackCount int32
	var wg sync.WaitGroup

	callback := func(event tlog.LogEvent) {
		defer wg.Done()
		atomic.AddInt32(&callbackCount, 1)
		// Simulate some work
		time.Sleep(time.Millisecond * 10)
	}

	// Register callback
	tlog.RegisterCallback(tlog.LevelInfo, callback)

	// Trigger multiple logs concurrently
	numLogs := 10
	for i := 0; i < numLogs; i++ {
		wg.Add(1)
		go func(i int) {
			tlog.Info("concurrent log", "iteration", i)
		}(i)
	}

	// Wait for all callbacks to complete
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(time.Second * 5):
		suite.Fail("Callbacks did not complete within timeout")
	}

	suite.Equal(int32(numLogs), atomic.LoadInt32(&callbackCount))
}

func (suite *TlogSuite) TestAllLogLevelsWithCallbacks() {
	var receivedEvents []tlog.LogEvent
	var mu sync.Mutex

	callback := func(event tlog.LogEvent) {
		mu.Lock()
		defer mu.Unlock()
		receivedEvents = append(receivedEvents, event)
	}

	// Register callbacks for all levels
	levels := []slog.Level{
		tlog.LevelTrace,
		tlog.LevelDebug,
		tlog.LevelInfo,
		tlog.LevelNotice,
		tlog.LevelWarn,
		tlog.LevelError,
	}

	for _, level := range levels {
		tlog.RegisterCallback(level, callback)
	}

	// Trigger logs for each level
	tlog.Trace("trace message")
	tlog.Debug("debug message")
	tlog.Info("info message")
	tlog.Notice("notice message")
	tlog.Warn("warn message")
	tlog.Error("error message")

	// Wait for callbacks
	suite.Eventually(func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(receivedEvents) == len(levels)
	}, time.Second*2, time.Millisecond*50)

	mu.Lock()
	defer mu.Unlock()

	// Verify all events were received
	suite.Len(receivedEvents, len(levels))

	// Create maps for easier verification since order is not guaranteed
	eventsByLevel := make(map[slog.Level]tlog.LogEvent)
	for _, event := range receivedEvents {
		eventsByLevel[event.Record.Level] = event
	}

	// Verify event details
	expectedMessages := map[slog.Level]string{
		tlog.LevelTrace:  "trace message",
		tlog.LevelDebug:  "debug message",
		tlog.LevelInfo:   "info message",
		tlog.LevelNotice: "notice message",
		tlog.LevelWarn:   "warn message",
		tlog.LevelError:  "error message",
	}

	for level, expectedMsg := range expectedMessages {
		event, exists := eventsByLevel[level]
		suite.True(exists, "Event for level %v should exist", level)
		suite.Equal(level, event.Record.Level)
		suite.Equal(expectedMsg, event.Record.Message)
	}
}

func (suite *TlogSuite) TestCallbackArgsFormatting() {
	// Test that args are formatted according to FormatterConfig before passing to callbacks
	var receivedEvents []tlog.LogEvent
	var mu sync.Mutex

	callback := func(event tlog.LogEvent) {
		mu.Lock()
		defer mu.Unlock()
		receivedEvents = append(receivedEvents, event)
	}

	// Register callback for info level
	tlog.RegisterCallback(tlog.LevelInfo, callback)

	// Test 1: Sensitive data hiding when enabled
	suite.Run("SensitiveDataHiding", func() {
		// Enable sensitive data hiding
		config := tlog.GetFormatterConfig()
		config.HideSensitiveData = true
		tlog.SetFormatterConfig(config)

		// Clear previous events
		mu.Lock()
		receivedEvents = nil
		mu.Unlock()

		// Log with sensitive data
		tlog.Info("test message", "password", "secret123", "token", "abc123", "normal_field", "visible")

		// Wait for callback
		suite.Eventually(func() bool {
			mu.Lock()
			defer mu.Unlock()
			return len(receivedEvents) > 0
		}, time.Second*2, time.Millisecond*50)

		// Verify sensitive data is redacted in callback args
		mu.Lock()
		defer mu.Unlock()
		suite.Require().Len(receivedEvents, 1)

		event := receivedEvents[0]
		argsMap := make(map[string]any)
		event.Record.Attrs(func(attr slog.Attr) bool {
			argsMap[attr.Key] = attr.Value.Any()
			return true
		})

		suite.Equal("secr*******", argsMap["password"])
		suite.Equal("abc1*******", argsMap["token"])
		suite.Equal("visible", argsMap["normal_field"])
	})

	// Test 2: Error formatting
	suite.Run("ErrorFormatting", func() {
		// Enable formatting
		config := tlog.GetFormatterConfig()
		config.EnableFormatting = true
		config.HideSensitiveData = false
		tlog.SetFormatterConfig(config)

		// Clear previous events
		mu.Lock()
		receivedEvents = nil
		mu.Unlock()

		// Log with standard error
		testErr := fmt.Errorf("test error message")
		tlog.Info("error occurred", "error", testErr, "context", "test")

		// Wait for callback
		suite.Eventually(func() bool {
			mu.Lock()
			defer mu.Unlock()
			return len(receivedEvents) > 0
		}, time.Second*2, time.Millisecond*50)

		// Verify error is formatted in callback args
		mu.Lock()
		defer mu.Unlock()
		suite.Require().Len(receivedEvents, 1)

		event := receivedEvents[0]
		argsMap := make(map[string]any)
		event.Record.Attrs(func(attr slog.Attr) bool {
			argsMap[attr.Key] = attr.Value.Any()
			return true
		})

		// Error should be formatted as a map
		errorValue := argsMap["error"]
		suite.Require().IsType([]slog.Attr{}, errorValue)

		errorMap := errorValue.([]slog.Attr)
		suite.Equal("test error message", errorMap[0].Value.String())
		suite.Contains(errorMap[1].Value.String(), "errorString") // fmt.Errorf creates *errors.errorString
		suite.Equal("test", argsMap["context"])
	})

	// Test 4: No formatting when disabled
	suite.Run("NoFormattingWhenDisabled", func() {
		// Disable formatting
		config := tlog.GetFormatterConfig()
		config.EnableFormatting = false
		tlog.SetFormatterConfig(config)

		// Clear previous events
		mu.Lock()
		receivedEvents = nil
		mu.Unlock()

		// Log with data that would normally be formatted
		testErr := fmt.Errorf("test error")
		tlog.Info("no formatting test", "error", testErr, "password", "secret123")

		// Wait for callback
		suite.Eventually(func() bool {
			mu.Lock()
			defer mu.Unlock()
			return len(receivedEvents) > 0
		}, time.Second*2, time.Millisecond*50)

		// Verify args are not formatted
		mu.Lock()
		defer mu.Unlock()
		suite.Require().Len(receivedEvents, 1)

		event := receivedEvents[0]
		argsMap := make(map[string]any)
		event.Record.Attrs(func(attr slog.Attr) bool {
			argsMap[attr.Key] = attr.Value.Any()
			return true
		})

		// Args should remain unformatted
		suite.Equal(testErr, argsMap["error"])        // Original error object
		suite.Equal("secret123", argsMap["password"]) // Not redacted
	})

	// Test 5: IP address masking
	suite.Run("IPAddressMasking", func() {
		// Enable sensitive data hiding
		config := tlog.GetFormatterConfig()
		config.HideSensitiveData = true
		config.EnableFormatting = true
		tlog.SetFormatterConfig(config)

		// Clear previous events
		mu.Lock()
		receivedEvents = nil
		mu.Unlock()

		// Log with IP addresses
		tlog.Info("network info", "client_ip", "192.168.1.100", "server_address", "10.0.0.1", "port", 8080)

		// Wait for callback
		suite.Eventually(func() bool {
			mu.Lock()
			defer mu.Unlock()
			return len(receivedEvents) > 0
		}, time.Second*2, time.Millisecond*50)

		// Verify IP addresses are masked
		mu.Lock()
		defer mu.Unlock()
		suite.Require().Len(receivedEvents, 1)

		event := receivedEvents[0]
		argsMap := make(map[string]any)
		event.Record.Attrs(func(attr slog.Attr) bool {
			argsMap[attr.Key] = attr.Value.Any()
			return true
		})

		// IP addresses should be masked
		suite.Equal("*******", argsMap["client_ip"])
		suite.Equal("10.0.0.1", argsMap["server_address"])
		suite.Equal(int64(8080), argsMap["port"]) // Non-IP field should remain unchanged
	})
}

func (suite *TlogSuite) TestGetCallbackCount() {
	// Initially no callbacks
	suite.Equal(0, tlog.GetCallbackCount(tlog.LevelError))

	callback := func(event tlog.LogEvent) {}

	// Add callbacks
	tlog.RegisterCallback(tlog.LevelError, callback)
	suite.Equal(1, tlog.GetCallbackCount(tlog.LevelError))

	tlog.RegisterCallback(tlog.LevelError, callback)
	suite.Equal(2, tlog.GetCallbackCount(tlog.LevelError))

	// Different level should still be 0
	suite.Equal(0, tlog.GetCallbackCount(tlog.LevelWarn))
}

func (suite *TlogSuite) TestCallbacksWithNoRegistrations() {
	// Should not panic when logging with no callbacks registered
	suite.NotPanics(func() {
		tlog.Info("info without callbacks")
		tlog.Error("error without callbacks")
		tlog.Warn("warn without callbacks")
	})
}

// Logger struct tests

func (suite *TlogSuite) TestNewLogger() {
	logger := tlog.NewLogger()
	suite.NotNil(logger)
	suite.NotNil(logger.Logger)
}

func (suite *TlogSuite) TestNewLoggerWithLevel() {
	logger := tlog.NewLoggerWithLevel(tlog.LevelDebug)
	suite.NotNil(logger)
	suite.NotNil(logger.Logger)
}

func (suite *TlogSuite) TestLoggerMethods() {
	logger := tlog.NewLogger()

	// Test that logger methods don't panic
	suite.NotPanics(func() {
		logger.Trace("trace message")
		logger.Debug("debug message")
		logger.Info("info message")
		logger.Notice("notice message")
		logger.Warn("warn message")
		logger.Error("error message")
	})
}

func (suite *TlogSuite) TestLoggerContextMethods() {
	logger := tlog.NewLogger()
	ctx := context.Background()

	// Test that logger context methods don't panic
	suite.NotPanics(func() {
		logger.TraceContext(ctx, "trace message")
		logger.DebugContext(ctx, "debug message")
		logger.InfoContext(ctx, "info message")
		logger.NoticeContext(ctx, "notice message")
		logger.WarnContext(ctx, "warn message")
		logger.ErrorContext(ctx, "error message")
	})
}

func (suite *TlogSuite) TestLoggerCallbackIntegration() {
	logger := tlog.NewLogger()

	var callbackTriggered int32
	var capturedEvent tlog.LogEvent
	var wg sync.WaitGroup

	wg.Add(1)
	callbackID := tlog.RegisterCallback(tlog.LevelError, func(event tlog.LogEvent) {
		atomic.StoreInt32(&callbackTriggered, 1)
		capturedEvent = event
		wg.Done()
	})

	// Log an error message
	testMessage := "test error message"
	logger.Error(testMessage, "key", "value")

	// Wait for callback to be triggered (with timeout)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Callback was triggered
	case <-time.After(100 * time.Millisecond):
		suite.Fail("Callback was not triggered within timeout")
	}

	suite.Equal(int32(1), atomic.LoadInt32(&callbackTriggered))
	suite.Equal(testMessage, capturedEvent.Record.Message)
	suite.Equal(tlog.LevelError, capturedEvent.Record.Level)

	// Clean up
	tlog.UnregisterCallback(tlog.LevelError, callbackID)
}

func (suite *TlogSuite) TestLoggerEmbeddedSlogFunctionality() {
	logger := tlog.NewLogger()

	// Test that we can use embedded slog.Logger methods
	suite.NotPanics(func() {
		// Use slog methods through embedding
		structuredLogger := logger.With("component", "test")
		structuredLogger.Info("test message")

		// Use WithGroup
		groupedLogger := logger.WithGroup("group")
		groupedLogger.Info("grouped message")

		// Direct access to underlying Logger
		logger.Logger.Log(context.Background(), slog.LevelInfo, "direct slog usage")
	})
}

func (suite *TlogSuite) TestLoggerWithLevelFunctionality() {
	// Create logger with debug level
	debugLogger := tlog.NewLoggerWithLevel(tlog.LevelDebug)

	// Set global level to error to ensure our logger has its own level
	tlog.SetLevel(tlog.LevelError)

	// The debug logger should still be able to log at debug level
	suite.NotPanics(func() {
		debugLogger.Debug("debug message from level-specific logger")
		debugLogger.Info("info message from level-specific logger")
	})
}

// Test formatter configuration
func (suite *TlogSuite) TestFormatterConfig() {
	// Test default configuration
	config := tlog.GetFormatterConfig()
	suite.True(config.EnableFormatting)

	// Test setting custom configuration
	newConfig := tlog.FormatterConfig{
		EnableColors:      true,
		EnableFormatting:  true,
		HideSensitiveData: true,
		TimeFormat:        "2006-01-02 15:04:05",
	}

	tlog.SetFormatterConfig(newConfig)
	updatedConfig := tlog.GetFormatterConfig()
	suite.Equal(newConfig.EnableFormatting, updatedConfig.EnableFormatting)
	suite.Equal(newConfig.HideSensitiveData, updatedConfig.HideSensitiveData)
	suite.Equal(newConfig.TimeFormat, updatedConfig.TimeFormat)
}

// Test color configuration
func (suite *TlogSuite) TestColorConfiguration() {
	// Test enabling colors
	tlog.EnableColors(true)
	// Note: IsColorsEnabled() depends on terminal support, so we just test the function exists
	_ = tlog.IsColorsEnabled()

	// Test disabling colors
	tlog.EnableColors(false)
	suite.False(tlog.IsColorsEnabled()) // Should be false when explicitly disabled
}

// Test sensitive data hiding
func (suite *TlogSuite) TestSensitiveDataHiding() {
	// Test enabling sensitive data hiding
	tlog.EnableSensitiveDataHiding(true)
	suite.True(tlog.IsSensitiveDataHidingEnabled())

	// Test disabling sensitive data hiding
	tlog.EnableSensitiveDataHiding(false)
	suite.False(tlog.IsSensitiveDataHidingEnabled())
}

// Test time format configuration
func (suite *TlogSuite) TestTimeFormatConfiguration() {
	customFormat := "2006-01-02 15:04:05"

	tlog.SetTimeFormat(customFormat)
	suite.Equal(customFormat, tlog.GetTimeFormat())

	// Restore default
	tlog.SetTimeFormat("2006-01-02T15:04:05Z07:00")
	suite.Equal("2006-01-02T15:04:05Z07:00", tlog.GetTimeFormat())
}

// Test enhanced logger creation
func (suite *TlogSuite) TestEnhancedLoggerCreation() {
	// Test creating logger with level
	logger := tlog.WithLevel(tlog.LevelDebug)
	suite.NotNil(logger)

	// Test creating new logger instances
	logger1 := tlog.NewLogger()
	logger2 := tlog.NewLoggerWithLevel(tlog.LevelError)

	suite.NotNil(logger1)
	suite.NotNil(logger2)

	// Test that methods exist on Logger
	suite.NotPanics(func() {
		logger1.Info("test message")
		logger2.Error("error message")
	})
}

// Test formatter integration with logging
func (suite *TlogSuite) TestFormatterIntegrationWithLogging() {
	// Enable formatting and sensitive data hiding
	tlog.EnableSensitiveDataHiding(true)

	// Test that logging with potentially sensitive data doesn't panic
	suite.NotPanics(func() {
		tlog.Info("user login", "password", "secret123", "token", "abc123")
		tlog.Error("connection failed", "ip", "192.168.1.1", "addr", "10.0.0.1")
	})

	// Test with context
	ctx := context.Background()
	suite.NotPanics(func() {
		tlog.InfoContext(ctx, "context logging", "key", "secret")
	})

	// Clean up
	tlog.EnableSensitiveDataHiding(false)
}

func TestTlogSuite(t *testing.T) {
	// Ensure cleanup after all tests
	defer func() {
		tlog.ClearAllCallbacks()
		tlog.Shutdown()
	}()

	suite.Run(t, new(TlogSuite))
}
