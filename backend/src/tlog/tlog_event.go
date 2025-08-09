package tlog

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"sync"
	"time"
)

// LogEvent represents a log event passed to callbacks
type LogEvent struct {
	Record  slog.Record
	Context context.Context
}

type EventHandler struct {
	enabled bool
}

func NewEventHandler() slog.Handler {
	return &EventHandler{
		enabled: true,
	}
}

func (h *EventHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *EventHandler) WithGroup(name string) slog.Handler {
	// https://cs.opensource.google/go/x/exp/+/46b07846:slog/handler.go;l=247
	if name == "" {
		return h
	}

	return h
}

func (h *EventHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return h.enabled
}

func (h *EventHandler) Handle(ctx context.Context, record slog.Record) error {

	if processor == nil {
		return nil
	}

	// Quick check if there are any callbacks for this level
	processor.callbacksMu.RLock()
	hasCallbacks := len(processor.callbacks[record.Level]) > 0
	processor.callbacksMu.RUnlock()

	if !hasCallbacks {
		return nil
	}

	event := LogEvent{
		Record:  record,
		Context: ctx,
	}

	// Non-blocking send to avoid affecting logging performance
	select {
	case processor.eventChan <- event:
		// Event queued successfully
	default:
		// Channel is full, drop the event to avoid blocking
		log.Println("tlog: callback event queue full, dropping event")
	}
	return nil
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

// processor globals for the event system
var (
	processor   *eventProcessor // global event processor
	processorMu sync.Mutex      // protects processor initialization
)

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
	callbacks := ep.callbacks[event.Record.Level]
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

/*
// applyCustomFormatting applies specific formatting rules based on FormatterConfig
func applyCustomFormatting(key string, value any, config FormatterConfig) any {
	keyStr := strings.ToLower(key)

	// Apply sensitive data hiding if enabled
	if config.HideSensitiveData {
		// Check for sensitive fields
		sensitiveFields := []string{
			"password", "pwd", "pass", "passwd", "token", "jwt",
			"auth_token", "access_token", "refresh_token", "key",
			"api_key", "secret", "client_secret", "private_key",
		}

		for _, sensitive := range sensitiveFields {
			if strings.Contains(keyStr, sensitive) {
				return "[REDACTED]"
			}
		}

		// Handle IP addresses
		if strings.Contains(keyStr, "ip") || strings.Contains(keyStr, "addr") || strings.Contains(keyStr, "address") {
			if str, ok := value.(string); ok {
				// Simple IP address detection and masking
				if strings.Contains(str, ".") && len(strings.Split(str, ".")) == 4 {
					parts := strings.Split(str, ".")
					if len(parts) >= 2 {
						return parts[0] + "." + parts[1] + ".xxx.xxx"
					}
				}
			}
		}
	}

	// Handle error formatting (tozd errors)
	if err, ok := value.(errors.E); ok {
		// Create formatted error information similar to TozdErrorFormatter
		formattedError := map[string]any{
			"message": err.Error(),
		}

		// Add details if available
		if details := errors.Details(err); len(details) > 0 {
			formattedError["details"] = details
		}

		// Add stack trace if available
		if stackTracer, ok := err.(interface{ StackTrace() []uintptr }); ok {
			stackTrace := stackTracer.StackTrace()
			if len(stackTrace) > 0 {
				frames := runtime.CallersFrames(stackTrace)
				var frameStrings []string
				frameIndex := 0

				for {
					frame, more := frames.Next()
					frameInfo := fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function)
					frameStrings = append(frameStrings, frameInfo)
					frameIndex++

					if !more || frameIndex >= 20 {
						break
					}
				}

				if config.MultilineStacktrace {
					formattedError["stacktrace_frames"] = frameStrings
				} else {
					formattedError["stacktrace"] = strings.Join(frameStrings, " -> ")
				}
			}
		}

		// Add cause if available
		if cause := errors.Cause(err); cause != nil && cause != err {
			formattedError["cause"] = cause.Error()
		}

		return formattedError
	}

	// Handle standard error formatting
	if err, ok := value.(error); ok && err != nil {
		return map[string]any{
			"message": err.Error(),
			"type":    fmt.Sprintf("%T", err),
		}
	}

	// Handle time formatting
	if t, ok := value.(time.Time); ok {
		return t.Format(config.TimeFormat)
	}

	// Handle Unix timestamp formatting
	timeFields := []string{"timestamp", "created_at", "updated_at", "time"}
	for _, timeField := range timeFields {
		if strings.Contains(keyStr, timeField) {
			switch v := value.(type) {
			case int64:
				if v > 0 {
					return time.Unix(v, 0).Format(config.TimeFormat)
				}
			case int:
				if v > 0 {
					return time.Unix(int64(v), 0).Format(config.TimeFormat)
				}
			case string:
				if parsed, err := strconv.ParseInt(v, 10, 64); err == nil && parsed > 0 {
					return time.Unix(parsed, 0).Format(config.TimeFormat)
				}
			case float64:
				if v > 0 {
					return time.Unix(int64(v), 0).Format(config.TimeFormat)
				}
			}
			break
		}
	}

	// Return original value if no formatting applied
	return value
}
*/
