package tlog

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	slogformatter "github.com/samber/slog-formatter"
	"gitlab.com/tozd/go/errors"
)

// Custom log levels extending slog.Level
const (
	LevelTrace  slog.Level = -8
	LevelDebug  slog.Level = slog.LevelDebug
	LevelInfo   slog.Level = slog.LevelInfo
	LevelNotice slog.Level = 2
	LevelWarn   slog.Level = slog.LevelWarn
	LevelError  slog.Level = slog.LevelError
	LevelFatal  slog.Level = 12
)

// LogEvent represents a log event passed to callbacks
type LogEvent struct {
	Level     slog.Level
	Message   string
	Args      []any
	Timestamp time.Time
	Context   context.Context
}

// LogCallback is the function signature for log event callbacks
type LogCallback func(event LogEvent)

// Logger wraps slog.Logger with additional tlog functionality
type Logger struct {
	*slog.Logger
}

// callbackEntry holds a callback with its metadata
type callbackEntry struct {
	callback LogCallback
	id       string
}

// eventProcessor handles asynchronous callback execution
type eventProcessor struct {
	eventChan   chan LogEvent
	callbacks   map[slog.Level][]callbackEntry
	callbacksMu sync.RWMutex
	wg          sync.WaitGroup
	shutdown    chan struct{}
	once        sync.Once
}

// levelNames maps level strings to slog.Level values
var levelNames = map[string]slog.Level{
	"trace":   LevelTrace,
	"debug":   LevelDebug,
	"info":    LevelInfo,
	"notice":  LevelNotice,
	"warn":    LevelWarn,
	"warning": LevelWarn, // alias for warn
	"error":   LevelError,
	"fatal":   LevelFatal,
}

// reverseLevelNames maps slog.Level values to canonical string names
var reverseLevelNames = map[slog.Level]string{
	LevelTrace:  "TRACE",
	LevelDebug:  "DEBUG",
	LevelInfo:   "INFO",
	LevelNotice: "NOTICE",
	LevelWarn:   "WARN",
	LevelError:  "ERROR",
	LevelFatal:  "FATAL",
}

// Color configuration for different log levels
var levelColors = map[slog.Level]*color.Color{
	LevelTrace:  color.New(color.FgHiBlack), // Bright black (gray)
	LevelDebug:  color.New(color.FgCyan),    // Cyan
	LevelInfo:   color.New(color.FgGreen),   // Green
	LevelNotice: color.New(color.FgBlue),    // Blue
	LevelWarn:   color.New(color.FgYellow),  // Yellow
	LevelError:  color.New(color.FgRed),     // Red
	LevelFatal:  color.New(color.FgHiRed),   // Bright red
}

// FormatterConfig holds configuration for log formatting
type FormatterConfig struct {
	EnableColors        bool
	EnableFormatting    bool
	HideSensitiveData   bool
	TimeFormat          string
	MultilineStacktrace bool // Enable multiline display of stacktraces instead of escaped single line
}

// defaultFormatterConfig provides default configuration
var defaultFormatterConfig = FormatterConfig{
	EnableColors:        true, // Will be disabled automatically if terminal doesn't support colors
	EnableFormatting:    true,
	HideSensitiveData:   false,
	TimeFormat:          time.RFC3339,
	MultilineStacktrace: false, // Default to single line for compatibility
}

var (
	programLevel      = new(slog.LevelVar) // Info by default
	mu                sync.RWMutex         // protects logger configuration changes
	processor         *eventProcessor      // global event processor
	processorMu       sync.Mutex           // protects processor initialization
	formatterConfig   FormatterConfig      // current formatter configuration
	formatterConfigMu sync.RWMutex         // protects formatter configuration changes
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Initialize formatter configuration with terminal detection
	formatterConfig = defaultFormatterConfig
	formatterConfig.EnableColors = defaultFormatterConfig.EnableColors && isTerminalSupported()

	initializeProcessor()
	initializeLogger()
}

// initializeProcessor sets up the event processor
func initializeProcessor() {
	processorMu.Lock()
	defer processorMu.Unlock()

	if processor != nil {
		return
	}

	processor = &eventProcessor{
		eventChan: make(chan LogEvent, 1000), // buffered channel for queue
		callbacks: make(map[slog.Level][]callbackEntry),
		shutdown:  make(chan struct{}),
	}

	// Start the processor goroutine
	processor.wg.Add(1)
	go processor.processEvents()
}

// processEvents handles incoming log events and executes callbacks
func (ep *eventProcessor) processEvents() {
	defer ep.wg.Done()

	for {
		select {
		case event := <-ep.eventChan:
			ep.executeCallbacks(event)
		case <-ep.shutdown:
			// Process remaining events before shutdown
			for {
				select {
				case event := <-ep.eventChan:
					ep.executeCallbacks(event)
				default:
					return
				}
			}
		}
	}
}

// executeCallbacks runs all callbacks for a specific log level
func (ep *eventProcessor) executeCallbacks(event LogEvent) {
	ep.callbacksMu.RLock()
	callbacks := ep.callbacks[event.Level]
	ep.callbacksMu.RUnlock()

	for _, entry := range callbacks {
		go ep.safeExecuteCallback(entry.callback, event)
	}
}

// safeExecuteCallback executes a callback with panic recovery
func (ep *eventProcessor) safeExecuteCallback(callback LogCallback, event LogEvent) {
	defer func() {
		if r := recover(); r != nil {
			// Log the panic but don't affect the main program
			log.Printf("tlog callback panic recovered: %v", r)
		}
	}()

	// Execute callback with error handling
	func() {
		defer func() {
			if r := recover(); r != nil {
				// This inner defer catches any panics from the callback
				panic(r) // re-panic to be caught by outer defer
			}
		}()
		callback(event)
	}()
}

// RegisterCallback registers a callback for a specific log level
// Returns a callback ID that can be used to unregister the callback
func RegisterCallback(level slog.Level, callback LogCallback) string {
	if processor == nil {
		initializeProcessor()
	}

	processor.callbacksMu.Lock()
	defer processor.callbacksMu.Unlock()

	// Generate unique ID
	id := fmt.Sprintf("callback_%d_%d", level, time.Now().UnixNano())

	entry := callbackEntry{
		callback: callback,
		id:       id,
	}

	processor.callbacks[level] = append(processor.callbacks[level], entry)

	return id
}

// UnregisterCallback removes a callback by its ID
func UnregisterCallback(level slog.Level, callbackID string) bool {
	if processor == nil {
		return false
	}

	processor.callbacksMu.Lock()
	defer processor.callbacksMu.Unlock()

	callbacks := processor.callbacks[level]
	for i, entry := range callbacks {
		if entry.id == callbackID {
			// Remove the callback from the slice
			processor.callbacks[level] = append(callbacks[:i], callbacks[i+1:]...)
			return true
		}
	}

	return false
}

// ClearCallbacks removes all callbacks for a specific level
func ClearCallbacks(level slog.Level) {
	if processor == nil {
		return
	}

	processor.callbacksMu.Lock()
	defer processor.callbacksMu.Unlock()

	delete(processor.callbacks, level)
}

// ClearAllCallbacks removes all registered callbacks
func ClearAllCallbacks() {
	if processor == nil {
		return
	}

	processor.callbacksMu.Lock()
	defer processor.callbacksMu.Unlock()

	processor.callbacks = make(map[slog.Level][]callbackEntry)
}

// GetCallbackCount returns the number of callbacks registered for a level
func GetCallbackCount(level slog.Level) int {
	if processor == nil {
		return 0
	}

	processor.callbacksMu.RLock()
	defer processor.callbacksMu.RUnlock()

	return len(processor.callbacks[level])
}

// Shutdown gracefully shuts down the event processor
func Shutdown() {
	if processor == nil {
		return
	}

	processor.once.Do(func() {
		close(processor.shutdown)
		processor.wg.Wait()
	})
}

// RestartProcessor shuts down the current processor and creates a new one
// This is mainly used for testing to ensure clean state between tests
func RestartProcessor() {
	processorMu.Lock()
	defer processorMu.Unlock()

	if processor != nil {
		// Signal shutdown
		select {
		case <-processor.shutdown:
			// Already shut down
		default:
			close(processor.shutdown)
		}
		processor.wg.Wait()
	}

	// Create new processor
	processor = &eventProcessor{
		eventChan: make(chan LogEvent, 1000),
		callbacks: make(map[slog.Level][]callbackEntry),
		shutdown:  make(chan struct{}),
	}

	// Start the processor goroutine
	processor.wg.Add(1)
	go processor.processEvents()
}

// emitLogEvent sends a log event to the processor if callbacks are registered
func emitLogEvent(ctx context.Context, level slog.Level, msg string, args ...any) {
	if processor == nil {
		return
	}

	// Quick check if there are any callbacks for this level
	processor.callbacksMu.RLock()
	hasCallbacks := len(processor.callbacks[level]) > 0
	processor.callbacksMu.RUnlock()

	if !hasCallbacks {
		return
	}

	event := LogEvent{
		Level:     level,
		Message:   msg,
		Args:      args,
		Timestamp: time.Now(),
		Context:   ctx,
	}

	// Non-blocking send to avoid affecting logging performance
	select {
	case processor.eventChan <- event:
		// Event queued successfully
	default:
		// Channel is full, drop the event to avoid blocking
		log.Println("tlog: callback event queue full, dropping event")
	}
}

// isTerminalSupported checks if the terminal supports colors
func isTerminalSupported() bool {
	return isatty.IsTerminal(os.Stderr.Fd())
}

// supportsUnicode checks if the terminal supports Unicode characters for tree formatting
func supportsUnicode() bool {
	if !isTerminalSupported() {
		return false
	}

	// Check common environment variables that indicate Unicode support
	term := os.Getenv("TERM")
	langVar := os.Getenv("LANG")
	lcAll := os.Getenv("LC_ALL")

	// Most modern terminals support Unicode
	unicodeTerms := []string{"xterm", "screen", "tmux", "alacritty", "kitty", "iterm", "vscode"}
	for _, unicodeTerm := range unicodeTerms {
		if strings.Contains(strings.ToLower(term), unicodeTerm) {
			return true
		}
	}

	// Check for UTF-8 in language settings
	if strings.Contains(strings.ToUpper(langVar), "UTF-8") ||
		strings.Contains(strings.ToUpper(lcAll), "UTF-8") {
		return true
	}

	// Default to true for most modern environments
	return term != "dumb" && term != ""
}

// extractContextValues extracts key-value pairs from context
func extractContextValues(ctx context.Context) []slog.Attr {
	var attrs []slog.Attr

	// Use reflection to inspect context values
	// This is a basic implementation - could be extended
	if ctx != nil {
		// Try common context keys
		commonKeys := []string{"request_id", "user_id", "session_id", "trace_id", "span_id"}
		for _, key := range commonKeys {
			if val := ctx.Value(key); val != nil {
				attrs = append(attrs, slog.Any(key, val))
			}
		}
	}

	return attrs
}

// extractContextToArgs extracts context values and converts them to args format
func extractContextToArgs(ctx context.Context) []any {
	if ctx == nil {
		return nil
	}

	var args []any

	// Try common context keys
	commonKeys := []string{"request_id", "user_id", "session_id", "trace_id", "span_id"}
	for _, key := range commonKeys {
		if val := ctx.Value(key); val != nil {
			args = append(args, key, val)
		}
	}

	return args
}

// HTTPRequestFormatter formats HTTP request information
func HTTPRequestFormatter(key string) slogformatter.Formatter {
	return slogformatter.FormatByType(func(v *http.Request) slog.Value {
		if v == nil {
			return slog.StringValue("<nil>")
		}
		return slog.GroupValue(
			slog.String("method", v.Method),
			slog.String("url", v.URL.String()),
			slog.String("proto", v.Proto),
			slog.Int64("content_length", v.ContentLength),
		)
	})
}

// HTTPResponseFormatter formats HTTP response information
func HTTPResponseFormatter(key string) slogformatter.Formatter {
	return slogformatter.FormatByType(func(v *http.Response) slog.Value {
		if v == nil {
			return slog.StringValue("<nil>")
		}
		return slog.GroupValue(
			slog.String("status", v.Status),
			slog.Int("status_code", v.StatusCode),
			slog.String("proto", v.Proto),
			slog.Int64("content_length", v.ContentLength),
		)
	})
}

// UnixTimestampFormatter formats Unix timestamps
func UnixTimestampFormatter(key string) slogformatter.Formatter {
	return slogformatter.FormatByKey(key, func(v slog.Value) slog.Value {
		var timestamp int64
		var ok bool

		switch val := v.Any().(type) {
		case int64:
			timestamp, ok = val, true
		case int:
			timestamp, ok = int64(val), true
		case string:
			if parsed, err := strconv.ParseInt(val, 10, 64); err == nil {
				timestamp, ok = parsed, true
			}
		case float64:
			timestamp, ok = int64(val), true
		}

		if ok && timestamp > 0 {
			// Convert Unix timestamp to readable time
			t := time.Unix(timestamp, 0)
			return slog.StringValue(t.Format(time.RFC3339))
		}

		return v
	})
}

// TozdErrorFormatter formats gitlab.com/tozd/go/errors with colored stacktraces
func TozdErrorFormatter(key string) slogformatter.Formatter {
	return slogformatter.FormatByType(func(v errors.E) slog.Value {
		// Create formatted error information
		var attrs []slog.Attr

		// Add error message
		attrs = append(attrs, slog.String("message", v.Error()))

		// Check if error has details
		if details := errors.Details(v); len(details) > 0 {
			var detailAttrs []any
			for k, val := range details {
				detailAttrs = append(detailAttrs, slog.Any(k, val))
			}
			attrs = append(attrs, slog.Group("details", detailAttrs...))
		}

		// Check if error has a stack trace
		if stackTracer, ok := v.(interface{ StackTrace() []uintptr }); ok {
			stackTrace := stackTracer.StackTrace()
			if len(stackTrace) > 0 {
				// Use runtime.CallersFrames to get proper frame information
				frames := runtime.CallersFrames(stackTrace)
				frameIndex := 0

				// Determine if we should use tree formatting
				useTreeFormat := supportsUnicode() && IsColorsEnabled()

				// Tree characters for Unicode-supported terminals
				treeChars := struct {
					vertical   string
					branch     string
					lastBranch string
					space      string
				}{
					vertical:   "│ ",
					branch:     "├─ ",
					lastBranch: "└─ ",
					space:      "   ",
				}

				// Fallback ASCII characters for terminals without Unicode support
				if !useTreeFormat {
					treeChars.vertical = "| "
					treeChars.branch = "|- "
					treeChars.lastBranch = "`- "
					treeChars.space = "   "
				}

				var allFrames []string // Collect all frames first to determine which is last

				for {
					frame, more := frames.Next()
					frameInfo := fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function)
					allFrames = append(allFrames, frameInfo)
					frameIndex++

					if !more || frameIndex >= 20 {
						if frameIndex >= 20 && more {
							allFrames = append(allFrames, "... (truncated)")
						}
						break
					}
				}

				// Build the complete stacktrace as a single formatted string
				var stackLines []string

				for i, frameInfo := range allFrames {
					var prefix string
					var coloredFrameInfo string

					// Determine the tree prefix
					if len(allFrames) == 1 {
						prefix = ""
					} else if i == len(allFrames)-1 {
						prefix = treeChars.lastBranch
					} else {
						prefix = treeChars.branch
					}

					// Apply color formatting if colors are enabled
					if IsColorsEnabled() {
						var frameColor *color.Color
						if i == 0 {
							// Highlight the top frame (most recent) in red
							frameColor = color.New(color.FgRed, color.Bold)
						} else if i < 3 {
							// Highlight the next few frames in yellow
							frameColor = color.New(color.FgYellow)
						} else {
							// Use gray for deeper stack frames
							frameColor = color.New(color.FgHiBlack)
						}

						coloredFrameInfo = frameColor.Sprint(frameInfo)

						// Color the tree prefix too
						if prefix != "" {
							coloredPrefix := color.New(color.FgCyan).Sprint(prefix)
							coloredFrameInfo = coloredPrefix + coloredFrameInfo
						} else {
							coloredFrameInfo = prefix + coloredFrameInfo
						}
					} else {
						coloredFrameInfo = prefix + frameInfo
					}

					stackLines = append(stackLines, coloredFrameInfo)
				}

				// Check if multiline stacktrace is enabled
				formatterConfigMu.RLock()
				multilineEnabled := formatterConfig.MultilineStacktrace
				formatterConfigMu.RUnlock()

				if multilineEnabled {
					// For multiline output, add each frame as a separate log message
					// We'll format them as individual slog attributes with proper indentation
					var stackFrames []any
					for i, line := range stackLines {
						stackFrames = append(stackFrames, slog.String(fmt.Sprintf("frame_%d", i), line))
					}
					attrs = append(attrs, slog.Group("stacktrace", stackFrames...))
				} else {
					// Join all lines with newlines to create the single-line tree structure
					stacktraceString := strings.Join(stackLines, "\n")
					attrs = append(attrs, slog.String("stacktrace", stacktraceString))
				}
			}
		}

		// Add cause if available (for error chains)
		if cause := errors.Cause(v); cause != nil && cause != v {
			attrs = append(attrs, slog.String("cause", cause.Error()))
		}

		return slog.GroupValue(attrs...)
	})
}

// createBaseHandler creates the base slog handler with appropriate configuration
func createBaseHandler() slog.Handler {
	formatterConfigMu.RLock()
	config := formatterConfig
	formatterConfigMu.RUnlock()

	isTerminal := isTerminalSupported()

	// Create base tint handler with context extraction
	handler := tint.NewHandler(os.Stderr, &tint.Options{
		Level:      programLevel,
		TimeFormat: config.TimeFormat,
		NoColor:    !isTerminal || !config.EnableColors,
		AddSource:  true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// First apply the level replacement
			a = replaceLogLevel(groups, a)

			// Extract context values and add them as log attributes
			if a.Key == "context" {
				if ctx, ok := a.Value.Any().(context.Context); ok {
					ctxAttrs := extractContextValues(ctx)
					if len(ctxAttrs) > 0 {
						args := make([]any, len(ctxAttrs))
						for i, attr := range ctxAttrs {
							args[i] = attr
						}
						return slog.Group("ctx", args...)
					}
				}
			}
			return a
		},
	})

	// If formatting is enabled, wrap with slog-formatter
	if config.EnableFormatting {
		var formatters []slogformatter.Formatter

		// Add tozd errors formatter for enhanced error display with stacktraces
		formatters = append(formatters, TozdErrorFormatter("error"))

		// Add generic error formatter for better error display (as fallback)
		formatters = append(formatters, slogformatter.ErrorFormatter("error"))

		// Add sensitive data formatter if enabled
		if config.HideSensitiveData {
			formatters = append(formatters,
				// Password and credential fields
				slogformatter.PIIFormatter("password"),
				slogformatter.PIIFormatter("pwd"),
				slogformatter.PIIFormatter("pass"),
				slogformatter.PIIFormatter("passwd"),
				slogformatter.PIIFormatter("token"),
				slogformatter.PIIFormatter("jwt"),
				slogformatter.PIIFormatter("auth_token"),
				slogformatter.PIIFormatter("access_token"),
				slogformatter.PIIFormatter("refresh_token"),
				slogformatter.PIIFormatter("key"),
				slogformatter.PIIFormatter("api_key"),
				slogformatter.PIIFormatter("secret"),
				slogformatter.PIIFormatter("client_secret"),
				slogformatter.PIIFormatter("private_key"),
				// Network addresses
				slogformatter.IPAddressFormatter("ip"),
				slogformatter.IPAddressFormatter("addr"),
				slogformatter.IPAddressFormatter("address"),
				slogformatter.IPAddressFormatter("remote_addr"),
				slogformatter.IPAddressFormatter("client_ip"),
				// Custom additional formatters
				HTTPRequestFormatter("request"),
				HTTPResponseFormatter("response"),
				UnixTimestampFormatter("timestamp"),
				UnixTimestampFormatter("created_at"),
				UnixTimestampFormatter("updated_at"),
			)
		}

		// Add time formatter
		formatters = append(formatters, slogformatter.TimeFormatter(config.TimeFormat, time.Local))

		// Apply formatters if any exist
		if len(formatters) > 0 {
			handler = slogformatter.NewFormatterHandler(formatters...)(handler)
		}
	}

	return handler
}

// initializeLogger sets up the default slog configuration
func initializeLogger() {
	handler := createBaseHandler()
	slog.SetDefault(slog.New(handler))
}

// replaceLogLevel customizes the display names for custom log levels
func replaceLogLevel(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.LevelKey {
		// Type assertion with check to handle both slog.Level and string values
		switch val := a.Value.Any().(type) {
		case slog.Level:
			if name, exists := reverseLevelNames[val]; exists {
				a.Value = slog.StringValue(name)
			}
		case string:
			// If it's already a string, leave it as is
			// This can happen if it was already processed by another formatter
		default:
			// Try to convert if it's not a string or slog.Level
			// This is a fallback in case the type changes in the future
		}
	}
	return a
}

// Trace logs a message at trace level
func Trace(msg string, args ...any) {
	ctx := context.Background()
	slog.Default().Log(ctx, LevelTrace, msg, args...)
	emitLogEvent(ctx, LevelTrace, msg, args...)
}

// TraceContext logs a message at trace level with context
func TraceContext(ctx context.Context, msg string, args ...any) {
	contextArgs := extractContextToArgs(ctx)
	allArgs := append(args, contextArgs...)
	slog.Default().Log(ctx, LevelTrace, msg, allArgs...)
	emitLogEvent(ctx, LevelTrace, msg, allArgs...)
}

// Debug logs a message at debug level
func Debug(msg string, args ...any) {
	ctx := context.Background()
	slog.Default().Log(ctx, slog.LevelDebug, msg, args...)
	emitLogEvent(ctx, slog.LevelDebug, msg, args...)
}

// DebugContext logs a message at debug level with context
func DebugContext(ctx context.Context, msg string, args ...any) {
	contextArgs := extractContextToArgs(ctx)
	allArgs := append(args, contextArgs...)
	slog.Default().Log(ctx, slog.LevelDebug, msg, allArgs...)
	emitLogEvent(ctx, slog.LevelDebug, msg, allArgs...)
}

// Info logs a message at info level
func Info(msg string, args ...any) {
	ctx := context.Background()
	slog.Default().Log(ctx, slog.LevelInfo, msg, args...)
	emitLogEvent(ctx, slog.LevelInfo, msg, args...)
}

// InfoContext logs a message at info level with context
func InfoContext(ctx context.Context, msg string, args ...any) {
	// Extract context values and add them to args
	contextArgs := extractContextToArgs(ctx)
	allArgs := append(args, contextArgs...)
	slog.Default().Log(ctx, slog.LevelInfo, msg, allArgs...)
	emitLogEvent(ctx, slog.LevelInfo, msg, allArgs...)
}

// Notice logs a message at notice level
func Notice(msg string, args ...any) {
	ctx := context.Background()
	slog.Default().Log(ctx, LevelNotice, msg, args...)
	emitLogEvent(ctx, LevelNotice, msg, args...)
}

// NoticeContext logs a message at notice level with context
func NoticeContext(ctx context.Context, msg string, args ...any) {
	contextArgs := extractContextToArgs(ctx)
	allArgs := append(args, contextArgs...)
	slog.Default().Log(ctx, LevelNotice, msg, allArgs...)
	emitLogEvent(ctx, LevelNotice, msg, allArgs...)
}

// Warn logs a message at warning level
func Warn(msg string, args ...any) {
	ctx := context.Background()
	slog.Default().Log(ctx, slog.LevelWarn, msg, args...)
	emitLogEvent(ctx, slog.LevelWarn, msg, args...)
}

// WarnContext logs a message at warning level with context
func WarnContext(ctx context.Context, msg string, args ...any) {
	contextArgs := extractContextToArgs(ctx)
	allArgs := append(args, contextArgs...)
	slog.Default().Log(ctx, slog.LevelWarn, msg, allArgs...)
	emitLogEvent(ctx, slog.LevelWarn, msg, allArgs...)
}

// Error logs a message at error level
func Error(msg string, args ...any) {
	ctx := context.Background()
	slog.Default().Log(ctx, slog.LevelError, msg, args...)
	emitLogEvent(ctx, slog.LevelError, msg, args...)
}

// ErrorContext logs a message at error level with context
func ErrorContext(ctx context.Context, msg string, args ...any) {
	contextArgs := extractContextToArgs(ctx)
	allArgs := append(args, contextArgs...)
	slog.Default().Log(ctx, slog.LevelError, msg, allArgs...)
	emitLogEvent(ctx, slog.LevelError, msg, allArgs...)
}

// Fatal logs a message at fatal level and exits the program
func Fatal(msg string, args ...any) {
	ctx := context.Background()
	slog.Default().Log(ctx, LevelFatal, msg, args...)
	emitLogEvent(ctx, LevelFatal, msg, args...)
	os.Exit(1)
}

// FatalContext logs a message at fatal level with context and exits the program
func FatalContext(ctx context.Context, msg string, args ...any) {
	contextArgs := extractContextToArgs(ctx)
	allArgs := append(args, contextArgs...)
	slog.Default().Log(ctx, LevelFatal, msg, allArgs...)
	emitLogEvent(ctx, LevelFatal, msg, allArgs...)
	os.Exit(1)
}

// SetLevel sets the minimum log level
func SetLevel(level slog.Level) {
	mu.Lock()
	defer mu.Unlock()
	programLevel.Set(level)
}

// GetLevel returns the current minimum log level
func GetLevel() slog.Level {
	mu.RLock()
	defer mu.RUnlock()
	return programLevel.Level()
}

// SetLevelFromString sets the log level from a string representation
// Supported levels: trace, debug, info, notice, warn/warning, error, fatal
// The comparison is case-insensitive
func SetLevelFromString(levelStr string) error {
	if levelStr == "" {
		return fmt.Errorf("log level cannot be empty")
	}

	normalizedLevel := strings.ToLower(strings.TrimSpace(levelStr))

	level, exists := levelNames[normalizedLevel]
	if !exists {
		return fmt.Errorf("invalid log level '%s': supported levels are %s",
			levelStr, getSupportedLevelsString())
	}

	mu.Lock()
	defer mu.Unlock()
	programLevel.Set(level)
	return nil
}

// GetLevelString returns the current log level as a string
func GetLevelString() string {
	level := GetLevel()
	if name, exists := reverseLevelNames[level]; exists {
		return name
	}
	return level.String()
}

// IsLevelEnabled checks if logging is enabled for the given level
func IsLevelEnabled(level slog.Level) bool {
	return GetLevel() <= level
}

// getSupportedLevelsString returns a comma-separated string of supported log levels
func getSupportedLevelsString() string {
	var levels []string
	seen := make(map[slog.Level]bool)

	for name, level := range levelNames {
		if !seen[level] {
			levels = append(levels, name)
			seen[level] = true
		}
	}

	return strings.Join(levels, ", ")
}

// SetFormatterConfig updates the formatter configuration and reinitializes the logger
func SetFormatterConfig(config FormatterConfig) {
	formatterConfigMu.Lock()
	formatterConfig = config
	formatterConfig.EnableColors = config.EnableColors && isTerminalSupported()
	formatterConfigMu.Unlock()

	// Reinitialize the logger with new configuration
	mu.Lock()
	defer mu.Unlock()
	initializeLogger()
}

// GetFormatterConfig returns the current formatter configuration
func GetFormatterConfig() FormatterConfig {
	formatterConfigMu.RLock()
	defer formatterConfigMu.RUnlock()
	return formatterConfig
}

// EnableColors enables or disables colored output
func EnableColors(enabled bool) {
	formatterConfigMu.Lock()
	formatterConfig.EnableColors = enabled && isTerminalSupported()
	formatterConfigMu.Unlock()

	// Reinitialize the logger with new configuration
	mu.Lock()
	defer mu.Unlock()
	initializeLogger()
}

// IsColorsEnabled returns true if colors are enabled and terminal supports them
func IsColorsEnabled() bool {
	formatterConfigMu.RLock()
	defer formatterConfigMu.RUnlock()
	return formatterConfig.EnableColors && isTerminalSupported()
}

// EnableSensitiveDataHiding enables or disables hiding of sensitive data (PII)
func EnableSensitiveDataHiding(enabled bool) {
	formatterConfigMu.Lock()
	formatterConfig.HideSensitiveData = enabled
	formatterConfigMu.Unlock()

	// Reinitialize the logger with new configuration
	mu.Lock()
	defer mu.Unlock()
	initializeLogger()
}

// IsSensitiveDataHidingEnabled returns true if sensitive data hiding is enabled
func IsSensitiveDataHidingEnabled() bool {
	formatterConfigMu.RLock()
	defer formatterConfigMu.RUnlock()
	return formatterConfig.HideSensitiveData
}

// SetTimeFormat sets the time format for log timestamps
func SetTimeFormat(format string) {
	formatterConfigMu.Lock()
	formatterConfig.TimeFormat = format
	formatterConfigMu.Unlock()

	// Reinitialize the logger with new configuration
	mu.Lock()
	defer mu.Unlock()
	initializeLogger()
}

// GetTimeFormat returns the current time format
func GetTimeFormat() string {
	formatterConfigMu.RLock()
	defer formatterConfigMu.RUnlock()
	return formatterConfig.TimeFormat
}

// EnableMultilineStacktrace enables or disables multiline display of stacktraces
// When enabled, stacktraces are displayed as separate log attributes for each frame
// When disabled (default), stacktraces are displayed as a single escaped string
func EnableMultilineStacktrace(enabled bool) {
	formatterConfigMu.Lock()
	formatterConfig.MultilineStacktrace = enabled
	formatterConfigMu.Unlock()

	// Reinitialize the logger with new configuration
	mu.Lock()
	defer mu.Unlock()
	initializeLogger()
}

// IsMultilineStacktraceEnabled returns true if multiline stacktrace display is enabled
func IsMultilineStacktraceEnabled() bool {
	formatterConfigMu.RLock()
	defer formatterConfigMu.RUnlock()
	return formatterConfig.MultilineStacktrace
}

// WithLevel creates a logger with a specific minimum level
// This is useful for creating loggers with different levels without affecting the global logger
func WithLevel(level slog.Level) *slog.Logger {
	levelVar := new(slog.LevelVar)
	levelVar.Set(level)

	formatterConfigMu.RLock()
	config := formatterConfig
	formatterConfigMu.RUnlock()

	isTerminal := isTerminalSupported()

	// Create base handler with specific level
	handler := tint.NewHandler(os.Stderr, &tint.Options{
		Level:       levelVar,
		TimeFormat:  config.TimeFormat,
		NoColor:     !isTerminal || !config.EnableColors,
		AddSource:   true,
		ReplaceAttr: replaceLogLevel,
	})

	// Apply formatters if enabled
	if config.EnableFormatting {
		var formatters []slogformatter.Formatter

		formatters = append(formatters, TozdErrorFormatter("error"))

		// Add generic error formatter for better error display (as fallback)
		formatters = append(formatters, slogformatter.ErrorFormatter("error"))

		if config.HideSensitiveData {
			formatters = append(formatters,
				slogformatter.PIIFormatter("password"),
				slogformatter.PIIFormatter("token"),
				slogformatter.PIIFormatter("key"),
				slogformatter.PIIFormatter("secret"),
				slogformatter.IPAddressFormatter("ip"),
				slogformatter.IPAddressFormatter("addr"),
				slogformatter.IPAddressFormatter("address"),
			)
		}

		formatters = append(formatters, slogformatter.TimeFormatter(config.TimeFormat, time.Local))

		if len(formatters) > 0 {
			handler = slogformatter.NewFormatterHandler(formatters...)(handler)
		}
	}

	return slog.New(handler)
}

// NewLogger creates a new Logger instance with the default configuration
func NewLogger() *Logger {
	return &Logger{
		Logger: slog.Default(),
	}
}

// NewLoggerWithLevel creates a new Logger instance with a specific minimum level
func NewLoggerWithLevel(level slog.Level) *Logger {
	return &Logger{
		Logger: WithLevel(level),
	}
}

// Logger methods that emit events to callbacks

// Trace logs a message at trace level
func (l *Logger) Trace(msg string, args ...any) {
	ctx := context.Background()
	l.Logger.Log(ctx, LevelTrace, msg, args...)
	emitLogEvent(ctx, LevelTrace, msg, args...)
}

// TraceContext logs a message at trace level with context
func (l *Logger) TraceContext(ctx context.Context, msg string, args ...any) {
	l.Logger.Log(ctx, LevelTrace, msg, args...)
	emitLogEvent(ctx, LevelTrace, msg, args...)
}

// Debug logs a message at debug level
func (l *Logger) Debug(msg string, args ...any) {
	ctx := context.Background()
	l.Logger.Log(ctx, slog.LevelDebug, msg, args...)
	emitLogEvent(ctx, slog.LevelDebug, msg, args...)
}

// DebugContext logs a message at debug level with context
func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.Logger.Log(ctx, slog.LevelDebug, msg, args...)
	emitLogEvent(ctx, slog.LevelDebug, msg, args...)
}

// Info logs a message at info level
func (l *Logger) Info(msg string, args ...any) {
	ctx := context.Background()
	l.Logger.Log(ctx, slog.LevelInfo, msg, args...)
	emitLogEvent(ctx, slog.LevelInfo, msg, args...)
}

// InfoContext logs a message at info level with context
func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.Logger.Log(ctx, slog.LevelInfo, msg, args...)
	emitLogEvent(ctx, slog.LevelInfo, msg, args...)
}

// Notice logs a message at notice level
func (l *Logger) Notice(msg string, args ...any) {
	ctx := context.Background()
	l.Logger.Log(ctx, LevelNotice, msg, args...)
	emitLogEvent(ctx, LevelNotice, msg, args...)
}

// NoticeContext logs a message at notice level with context
func (l *Logger) NoticeContext(ctx context.Context, msg string, args ...any) {
	l.Logger.Log(ctx, LevelNotice, msg, args...)
	emitLogEvent(ctx, LevelNotice, msg, args...)
}

// Warn logs a message at warning level
func (l *Logger) Warn(msg string, args ...any) {
	ctx := context.Background()
	l.Logger.Log(ctx, slog.LevelWarn, msg, args...)
	emitLogEvent(ctx, slog.LevelWarn, msg, args...)
}

// WarnContext logs a message at warning level with context
func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.Logger.Log(ctx, slog.LevelWarn, msg, args...)
	emitLogEvent(ctx, slog.LevelWarn, msg, args...)
}

// Error logs a message at error level
func (l *Logger) Error(msg string, args ...any) {
	ctx := context.Background()
	l.Logger.Log(ctx, slog.LevelError, msg, args...)
	emitLogEvent(ctx, slog.LevelError, msg, args...)
}

// ErrorContext logs a message at error level with context
func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.Logger.Log(ctx, slog.LevelError, msg, args...)
	emitLogEvent(ctx, slog.LevelError, msg, args...)
}

// Fatal logs a message at fatal level and exits the program
func (l *Logger) Fatal(msg string, args ...any) {
	ctx := context.Background()
	l.Logger.Log(ctx, LevelFatal, msg, args...)
	emitLogEvent(ctx, LevelFatal, msg, args...)
	os.Exit(1)
}

// FatalContext logs a message at fatal level with context and exits the program
func (l *Logger) FatalContext(ctx context.Context, msg string, args ...any) {
	l.Logger.Log(ctx, LevelFatal, msg, args...)
	emitLogEvent(ctx, LevelFatal, msg, args...)
	os.Exit(1)
}

// Color-enabled printing functions for enhanced output

// ColorPrint prints a message with color for the specified log level
func ColorPrint(level slog.Level, message string, args ...interface{}) {
	if !IsColorsEnabled() {
		fmt.Printf(message, args...)
		return
	}

	if colorFunc, exists := levelColors[level]; exists {
		colorFunc.Printf(message, args...)
	} else {
		fmt.Printf(message, args...)
	}
}

// ColorPrintln prints a message with color and newline for the specified log level
func ColorPrintln(level slog.Level, message string, args ...interface{}) {
	ColorPrint(level, message+"\n", args...)
}

// ColorTrace prints a trace message with color (if enabled)
func ColorTrace(message string, args ...interface{}) {
	ColorPrint(LevelTrace, message, args...)
}

// ColorDebug prints a debug message with color (if enabled)
func ColorDebug(message string, args ...interface{}) {
	ColorPrint(LevelDebug, message, args...)
}

// ColorInfo prints an info message with color (if enabled)
func ColorInfo(message string, args ...interface{}) {
	ColorPrint(LevelInfo, message, args...)
}

// ColorNotice prints a notice message with color (if enabled)
func ColorNotice(message string, args ...interface{}) {
	ColorPrint(LevelNotice, message, args...)
}

// ColorWarn prints a warning message with color (if enabled)
func ColorWarn(message string, args ...interface{}) {
	ColorPrint(LevelWarn, message, args...)
}

// ColorError prints an error message with color (if enabled)
func ColorError(message string, args ...interface{}) {
	ColorPrint(LevelError, message, args...)
}

// ColorFatal prints a fatal message with color (if enabled)
func ColorFatal(message string, args ...interface{}) {
	ColorPrint(LevelFatal, message, args...)
}

// PrintWithLevel prints a message with the appropriate level prefix and selective coloring
func PrintWithLevel(level slog.Level, message string, args ...interface{}) {
	levelName := reverseLevelNames[level]
	if levelName == "" {
		levelName = level.String()
	}

	// Format the message with arguments
	formattedMessage := fmt.Sprintf(message, args...)

	if !IsColorsEnabled() {
		fmt.Printf("[%s] %s", levelName, formattedMessage)
		return
	}

	// For levels below WARN (TRACE, DEBUG, INFO, NOTICE), color only the prefix
	if level < LevelWarn {
		if colorFunc, exists := levelColors[level]; exists {
			colorFunc.Printf("[%s]", levelName)
			fmt.Printf(" %s", formattedMessage)
		} else {
			fmt.Printf("[%s] %s", levelName, formattedMessage)
		}
	} else {
		// For WARN and above, color the entire message
		ColorPrint(level, "[%s] %s", levelName, formattedMessage)
	}
}

// PrintWithLevelAll demonstrates all log levels with their respective formatting
func PrintWithLevelAll(message string, args ...interface{}) {
	levels := []slog.Level{
		LevelTrace,
		LevelDebug,
		LevelInfo,
		LevelNotice,
		LevelWarn,
		LevelError,
		LevelFatal,
	}

	fmt.Println("\nAll Log Levels with Prefixes:")
	for _, level := range levels {
		PrintWithLevel(level, message+"\n", args...)
	}
}
