package tlog

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
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

var (
	programLevel = new(slog.LevelVar) // Info by default
	mu           sync.RWMutex         // protects logger configuration changes
	processor    *eventProcessor      // global event processor
	processorMu  sync.Mutex           // protects processor initialization
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
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

// initializeLogger sets up the default slog configuration
func initializeLogger() {
	isTerminal := isatty.IsTerminal(os.Stderr.Fd())

	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:       programLevel,
			TimeFormat:  time.RFC3339,
			NoColor:     !isTerminal,
			AddSource:   true,
			ReplaceAttr: replaceLogLevel,
		}),
	))
}

// replaceLogLevel customizes the display names for custom log levels
func replaceLogLevel(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.LevelKey {
		level := a.Value.Any().(slog.Level)
		if name, exists := reverseLevelNames[level]; exists {
			a.Value = slog.StringValue(name)
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
	slog.Default().Log(ctx, LevelTrace, msg, args...)
	emitLogEvent(ctx, LevelTrace, msg, args...)
}

// Debug logs a message at debug level
func Debug(msg string, args ...any) {
	ctx := context.Background()
	slog.Default().Log(ctx, slog.LevelDebug, msg, args...)
	emitLogEvent(ctx, slog.LevelDebug, msg, args...)
}

// DebugContext logs a message at debug level with context
func DebugContext(ctx context.Context, msg string, args ...any) {
	slog.Default().Log(ctx, slog.LevelDebug, msg, args...)
	emitLogEvent(ctx, slog.LevelDebug, msg, args...)
}

// Info logs a message at info level
func Info(msg string, args ...any) {
	ctx := context.Background()
	slog.Default().Log(ctx, slog.LevelInfo, msg, args...)
	emitLogEvent(ctx, slog.LevelInfo, msg, args...)
}

// InfoContext logs a message at info level with context
func InfoContext(ctx context.Context, msg string, args ...any) {
	slog.Default().Log(ctx, slog.LevelInfo, msg, args...)
	emitLogEvent(ctx, slog.LevelInfo, msg, args...)
}

// Notice logs a message at notice level
func Notice(msg string, args ...any) {
	ctx := context.Background()
	slog.Default().Log(ctx, LevelNotice, msg, args...)
	emitLogEvent(ctx, LevelNotice, msg, args...)
}

// NoticeContext logs a message at notice level with context
func NoticeContext(ctx context.Context, msg string, args ...any) {
	slog.Default().Log(ctx, LevelNotice, msg, args...)
	emitLogEvent(ctx, LevelNotice, msg, args...)
}

// Warn logs a message at warning level
func Warn(msg string, args ...any) {
	ctx := context.Background()
	slog.Default().Log(ctx, slog.LevelWarn, msg, args...)
	emitLogEvent(ctx, slog.LevelWarn, msg, args...)
}

// WarnContext logs a message at warning level with context
func WarnContext(ctx context.Context, msg string, args ...any) {
	slog.Default().Log(ctx, slog.LevelWarn, msg, args...)
	emitLogEvent(ctx, slog.LevelWarn, msg, args...)
}

// Error logs a message at error level
func Error(msg string, args ...any) {
	ctx := context.Background()
	slog.Default().Log(ctx, slog.LevelError, msg, args...)
	emitLogEvent(ctx, slog.LevelError, msg, args...)
}

// ErrorContext logs a message at error level with context
func ErrorContext(ctx context.Context, msg string, args ...any) {
	slog.Default().Log(ctx, slog.LevelError, msg, args...)
	emitLogEvent(ctx, slog.LevelError, msg, args...)
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
	slog.Default().Log(ctx, LevelFatal, msg, args...)
	emitLogEvent(ctx, LevelFatal, msg, args...)
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

// WithLevel creates a logger with a specific minimum level
// This is useful for creating loggers with different levels without affecting the global logger
func WithLevel(level slog.Level) *slog.Logger {
	levelVar := new(slog.LevelVar)
	levelVar.Set(level)

	isTerminal := isatty.IsTerminal(os.Stderr.Fd())

	return slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:       levelVar,
			TimeFormat:  time.RFC3339,
			NoColor:     !isTerminal,
			AddSource:   true,
			ReplaceAttr: replaceLogLevel,
		}),
	)
}
