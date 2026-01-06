package tlog

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"

	"github.com/k0kubun/pp/v3"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	slogformatter "github.com/samber/slog-formatter"
	slogmulti "github.com/samber/slog-multi"
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

// Logger wraps slog.Logger with additional tlog functionality
type Logger struct {
	*slog.Logger
	commonKeys []string
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

// defaultCommonKeys is the default list of context keys to extract
var defaultCommonKeys = []string{"X-Trace-Id", "X-Span-Id", "request_id", "user_id", "session_id", "trace_id", "span_id", "event_uuid"}

var levelColorNumbers = map[string]uint8{
	"TRACE":  7,
	"DEBUG":  6,
	"INFO":   2,
	"NOTICE": 4,
	"WARN":   3,
	"ERROR":  1,
	"FATAL":  9,
}

// sensitiveKeys holds keys considered sensitive for masking
var sensitiveKeys = map[string]struct{}{
	"password": {}, "pwd": {}, "pass": {}, "passwd": {},
	"token": {}, "jwt": {}, "auth_token": {}, "access_token": {}, "refresh_token": {},
	"key": {}, "api_key": {}, "secret": {}, "client_secret": {}, "private_key": {},
}

// maskString shows first 4 chars then 7 asterisks, matching test expectations
func maskString(s string) string {
	prefix := s
	if len(prefix) > 4 {
		prefix = prefix[:4]
	}
	return prefix + strings.Repeat("*", 7)
}

// maskNestedValue walks nested structures and masks values whose immediate key is sensitive
func maskNestedValue(v any, keyHint string) any {
	if v == nil {
		return nil
	}

	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return v
		}
		rv = rv.Elem()
		v = rv.Interface()
	}

	if keyHint != "" {
		if _, ok := sensitiveKeys[keyHint]; ok {
			if sv, ok := v.(string); ok {
				return maskString(sv)
			}
		}
	}

	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, vv := range val {
			out[k] = maskNestedValue(vv, k)
		}
		return out
	case map[string]string:
		out := make(map[string]string, len(val))
		for k, vv := range val {
			if _, ok := sensitiveKeys[k]; ok {
				out[k] = maskString(vv)
			} else {
				out[k] = vv
			}
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, vv := range val {
			out[i] = maskNestedValue(vv, "")
		}
		return out
	case []map[string]any:
		out := make([]map[string]any, len(val))
		for i, m := range val {
			out[i] = maskNestedValue(m, "").(map[string]any)
		}
		return out
	case []slog.Attr:
		out := make([]slog.Attr, 0, len(val))
		for _, attr := range val {
			masked := maskNestedValue(attr.Value.Any(), attr.Key)
			out = append(out, slog.Any(attr.Key, masked))
		}
		return out
	default:
		rv = reflect.ValueOf(v)
		if !rv.IsValid() {
			return v
		}
		switch rv.Kind() {
		case reflect.Slice, reflect.Array:
			out := make([]any, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				out[i] = maskNestedValue(rv.Index(i).Interface(), "")
			}
			return out
		case reflect.Struct:
			typeOf := rv.Type()
			out := make(map[string]any, rv.NumField())
			for i := 0; i < rv.NumField(); i++ {
				field := typeOf.Field(i)
				if field.PkgPath != "" { // unexported
					continue
				}
				key := field.Name
				if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
					if idx := strings.Index(tag, ","); idx >= 0 {
						key = tag[:idx]
					} else {
						key = tag
					}
				}
				out[key] = maskNestedValue(rv.Field(i).Interface(), key)
			}
			return out
		}

		return v
	}
}

// maskingHandler wraps a slog.Handler and masks sensitive data inside records
type maskingHandler struct{ next slog.Handler }

func (mh *maskingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return mh.next.Enabled(ctx, level)
}

func (mh *maskingHandler) Handle(ctx context.Context, r slog.Record) error {
	nr := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	r.Attrs(func(attr slog.Attr) bool {
		mv := maskNestedValue(attr.Value.Any(), attr.Key)
		nr.Add(slog.Any(attr.Key, mv))
		return true
	})
	return mh.next.Handle(ctx, nr)
}

func (mh *maskingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &maskingHandler{next: mh.next.WithAttrs(attrs)}
}
func (mh *maskingHandler) WithGroup(group string) slog.Handler {
	return &maskingHandler{next: mh.next.WithGroup(group)}
}

// FormatterConfig holds configuration for log formatting
type FormatterConfig struct {
	EnableColors        bool
	EnableFormatting    bool
	HideSensitiveData   bool
	TimeFormat          string
	MultilineStacktrace bool
}

// defaultFormatterConfig provides default configuration
var defaultFormatterConfig = FormatterConfig{
	EnableColors:        true, // Will be disabled automatically if terminal doesn't support colors
	EnableFormatting:    true,
	HideSensitiveData:   false,
	TimeFormat:          time.RFC3339,
	MultilineStacktrace: false,
}

var (
	programLevel      = new(slog.LevelVar) // Info by default
	mu                sync.RWMutex         // protects logger configuration changes
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

// ... callback and event-related code moved to tlog_event.go ...

// isTerminalSupported checks if the terminal supports colors
func isTerminalSupported() bool {
	//slog.Info("Checking if terminal supports colors", "term", os.Getenv("TERM"))
	return isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd()) || strings.Contains(os.Getenv("TERM"), "color")
}

// extractContextValues extracts key-value pairs from context
func extractContextValues(ctx context.Context, commonKeys []string) []slog.Attr {
	var attrs []slog.Attr

	// Use reflection to inspect context values
	// This is a basic implementation - could be extended
	if ctx != nil {
		// Try common context keys
		if commonKeys == nil {
			commonKeys = defaultCommonKeys
		}
		for _, key := range commonKeys {
			if val := ctx.Value(key); val != nil {
				attrs = append(attrs, slog.Any(key, val))
			}
		}
	}

	return attrs
}

// extractContextToArgs extracts context values and converts them to args format
func extractContextToArgs(ctx context.Context, commonKeys []string) []any {
	if ctx == nil {
		return nil
	}

	var args []any

	// Try common context keys
	if commonKeys == nil {
		commonKeys = defaultCommonKeys
	}
	for _, key := range commonKeys {
		if val := ctx.Value(key); val != nil {
			args = append(args, key, val)
		}
	}

	return args
}

// createBaseHandler creates the base slog handler with appropriate configuration
func createBaseHandler(level slog.Level) slog.Handler {
	formatterConfigMu.RLock()
	config := formatterConfig
	formatterConfigMu.RUnlock()

	isTerminal := isTerminalSupported()

	pp.SetDefaultOutput(os.Stderr)
	pp.Default.SetColoringEnabled(config.EnableColors && isTerminal)

	color.NoColor = !isTerminal || !config.EnableColors

	// Create base tint handler with context extraction
	tintHandler := tint.NewHandler(os.Stderr, &tint.Options{
		Level:      level,
		TimeFormat: config.TimeFormat,
		NoColor:    !isTerminal || !config.EnableColors,

		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// First apply the level replacement
			a = replaceLogLevel(groups, a)

			// Optionally hide sensitive data even inside nested payloads (only when formatting enabled)
			if config.EnableFormatting && config.HideSensitiveData && a.Key != slog.LevelKey {
				masked := maskNestedValue(a.Value.Any(), a.Key)
				a = slog.Any(a.Key, masked)
			}

			// Extract context values and add them as log attributes
			if a.Key == "context" {
				if ctx, ok := a.Value.Any().(context.Context); ok {
					ctxAttrs := extractContextValues(ctx, defaultCommonKeys)
					if len(ctxAttrs) > 0 {
						args := make([]any, len(ctxAttrs))
						for i, attr := range ctxAttrs {
							args[i] = attr
						}
						return slog.Group("ctx", args...)
					}
				}
			}

			// Remove error.org_error from output
			if a.Key == "org_error" {
				return slog.Attr{}
			}

			return a
		},
	})

	eventHandler := NewEventHandler()
	callbackHandler := slog.Handler(eventHandler)

	formattedTintHandler := slog.Handler(tintHandler)

	// Ensure callbacks see masked values too (only when formatting enabled)
	if config.EnableFormatting && config.HideSensitiveData {
		formattedTintHandler = &maskingHandler{next: formattedTintHandler}
		callbackHandler = &maskingHandler{next: callbackHandler}
	}

	// If formatting is enabled, wrap with slog-formatter
	if config.EnableFormatting {
		var formatters []slogformatter.Formatter

		// Add tozd errors formatter for enhanced error display with stacktraces
		formatters = append(formatters, TozdErrorFormatter())

		// Add generic error formatter for better error display (as fallback)
		formatters = append(formatters, ErrorFormatter("error"))

		// Add sensitive data formatter if enabled
		if config.HideSensitiveData {
			// Build PII formatters dynamically from sensitiveKeys for determinism
			piiKeys := make([]string, 0, len(sensitiveKeys))
			for k := range sensitiveKeys {
				piiKeys = append(piiKeys, k)
			}
			sort.Strings(piiKeys)
			for _, k := range piiKeys {
				formatters = append(formatters, slogformatter.PIIFormatter(k))
			}

			// Additional non-PII formatters
			formatters = append(formatters,
				// Network addresses
				slogformatter.IPAddressFormatter("ip"),
				slogformatter.IPAddressFormatter("addr"),
				slogformatter.IPAddressFormatter("address"),
				slogformatter.IPAddressFormatter("remote_addr"),
				slogformatter.IPAddressFormatter("client_ip"),
				// Custom additional formatters
				slogformatter.UnixTimestampFormatter(time.Millisecond),
				slogformatter.HTTPRequestFormatter(false),
				slogformatter.HTTPResponseFormatter(false),
				slogformatter.TimeFormatter(config.TimeFormat, time.Local),
			)
		}

		// Add time formatter
		formatters = append(formatters, slogformatter.TimeFormatter(config.TimeFormat, time.Local))

		// Apply formatters if any exist
		if len(formatters) > 0 {
			formatterHandler := slogformatter.NewFormatterHandler(formatters...)
			formattedTintHandler = formatterHandler(tintHandler)
			callbackHandler = formatterHandler(callbackHandler)
		}
	}

	return slogmulti.Fanout(formattedTintHandler, callbackHandler)
}

var defaultLogger *Logger

// initializeLogger sets up the default slog configuration
func initializeLogger() {
	handler := createBaseHandler(programLevel.Level())
	defaultLogger = &Logger{
		Logger:     slog.New(handler),
		commonKeys: defaultCommonKeys,
	}
	slog.SetDefault(defaultLogger.Logger)
}

// replaceLogLevel customizes the display names for custom log levels
func replaceLogLevel(_ []string, a slog.Attr) slog.Attr {
	if a.Key == slog.LevelKey {
		// Type assertion with check to handle both slog.Level and string values
		switch val := a.Value.Any().(type) {
		case slog.Level:
			if name, exists := reverseLevelNames[val]; exists {
				a.Value = slog.StringValue(name)
				a = tint.Attr(levelColorNumbers[name], a)
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

// getCallerInfo returns the caller's file, function, and line number
// skipFrames parameter allows skipping additional stack frames (default 0 means immediate caller)
func getCallerInfo(skipFrames int) (file string, function string, line int) {
	pc := make([]uintptr, 1)
	// skip [runtime.Callers, getCallerInfo, caller of getCallerInfo, ...additional frames]
	runtime.Callers(2+skipFrames, pc[:])
	if pc[0] == 0 {
		return "unknown", "unknown", 0
	}
	fn := runtime.FuncForPC(pc[0])
	if fn == nil {
		return "unknown", "unknown", 0
	}
	file, line = fn.FileLine(pc[0])
	// Extract just the filename without the full path
	if idx := strings.LastIndex(file, "/"); idx != -1 {
		file = file[idx+1:]
	}
	// Extract just the function name without the package path
	funcName := fn.Name()
	if idx := strings.LastIndex(funcName, "."); idx != -1 {
		funcName = funcName[idx+1:]
	}
	return file, funcName, line
}

// WithCaller is a helper function that adds caller information to the log args
// The skipFrames parameter allows skipping additional stack frames beyond the default
// Usage: tlog.Debug("message", tlog.WithCaller(0)..., "key", "value")
func WithCaller(skipFrames int) []any {
	file, function, line := getCallerInfo(2 + skipFrames)
	return []any{"caller", fmt.Sprintf("%s:%s:%d", file, function, line)}
}

// Trace logs a message at trace level
func Trace(msg string, args ...any) {
	ctx := context.Background()
	defaultLogger.log(ctx, LevelTrace, msg, args...)
}

// TraceContext logs a message at trace level with context
func TraceContext(ctx context.Context, msg string, args ...any) {
	contextArgs := extractContextToArgs(ctx, defaultLogger.commonKeys)
	allArgs := append(args, contextArgs...)
	defaultLogger.log(ctx, LevelTrace, msg, allArgs...)
}

// Debug logs a message at debug level
func Debug(msg string, args ...any) {
	ctx := context.Background()
	defaultLogger.log(ctx, slog.LevelDebug, msg, args...)
}

// log is the low-level logging method for methods that take ...any.
// It must always be called directly by an exported logging method
// or function, because it uses a fixed call depth to obtain the pc.
func (l *Logger) log(ctx context.Context, level slog.Level, msg string, args ...any) {
	if ctx == nil {
		ctx = context.Background()
	}
	if !l.Enabled(ctx, level) {
		return
	}
	//var pc uintptr
	var pcs []uintptr = make([]uintptr, 50)
	// skip [runtime.Callers, this function, this function's caller]
	runtime.Callers(3, pcs[:])
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	r.Add(args...)
	_ = l.Handler().Handle(ctx, r)
}

// DebugContext logs a message at debug level with context
func DebugContext(ctx context.Context, msg string, args ...any) {
	contextArgs := extractContextToArgs(ctx, defaultLogger.commonKeys)
	allArgs := append(args, contextArgs...)
	defaultLogger.log(ctx, slog.LevelDebug, msg, allArgs...)
}

// Info logs a message at info level
func Info(msg string, args ...any) {
	ctx := context.Background()
	defaultLogger.log(ctx, slog.LevelInfo, msg, args...)
}

// InfoContext logs a message at info level with context
func InfoContext(ctx context.Context, msg string, args ...any) {
	// Extract context values and add them to args
	contextArgs := extractContextToArgs(ctx, defaultLogger.commonKeys)
	allArgs := append(args, contextArgs...)
	defaultLogger.log(ctx, slog.LevelInfo, msg, allArgs...)
}

// Notice logs a message at notice level
func Notice(msg string, args ...any) {
	ctx := context.Background()
	defaultLogger.log(ctx, LevelNotice, msg, args...)
}

// NoticeContext logs a message at notice level with context
func NoticeContext(ctx context.Context, msg string, args ...any) {
	contextArgs := extractContextToArgs(ctx, defaultLogger.commonKeys)
	allArgs := append(args, contextArgs...)
	defaultLogger.log(ctx, LevelNotice, msg, allArgs...)
}

// Warn logs a message at warning level
func Warn(msg string, args ...any) {
	ctx := context.Background()
	defaultLogger.log(ctx, slog.LevelWarn, msg, args...)
}

// WarnContext logs a message at warning level with context
func WarnContext(ctx context.Context, msg string, args ...any) {
	contextArgs := extractContextToArgs(ctx, defaultLogger.commonKeys)
	allArgs := append(args, contextArgs...)
	defaultLogger.log(ctx, slog.LevelWarn, msg, allArgs...)
}

// Error logs a message at error level
func Error(msg string, args ...any) {
	ctx := context.Background()
	defaultLogger.log(ctx, slog.LevelError, msg, args...)
}

// ErrorContext logs a message at error level with context
func ErrorContext(ctx context.Context, msg string, args ...any) {
	contextArgs := extractContextToArgs(ctx, defaultLogger.commonKeys)
	allArgs := append(args, contextArgs...)
	defaultLogger.log(ctx, slog.LevelError, msg, allArgs...)
}

// Fatal logs a message at fatal level and exits the program
func Fatal(msg string, args ...any) {
	ctx := context.Background()
	defaultLogger.log(ctx, LevelFatal, msg, args...)
	os.Exit(1)
}

// FatalContext logs a message at fatal level with context and exits the program
func FatalContext(ctx context.Context, msg string, args ...any) {
	contextArgs := extractContextToArgs(ctx, defaultLogger.commonKeys)
	allArgs := append(args, contextArgs...)
	defaultLogger.log(ctx, LevelFatal, msg, allArgs...)
	panic("Fatal log called, exiting program") // Use panic to ensure all deferred functions run
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
	initializeLogger() // Reinitialize logger with new level
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

// EnableMultilineStacktrace toggles multi-line stack trace formatting.
func EnableMultilineStacktrace(enabled bool) {
	formatterConfigMu.Lock()
	formatterConfig.MultilineStacktrace = enabled
	formatterConfigMu.Unlock()

	mu.Lock()
	defer mu.Unlock()
	initializeLogger()
}

// IsMultilineStacktraceEnabled returns true if multi-line stack traces are enabled.
func IsMultilineStacktraceEnabled() bool {
	formatterConfigMu.RLock()
	defer formatterConfigMu.RUnlock()
	return formatterConfig.MultilineStacktrace
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

// withLevelLogger creates a slog.Logger with a specific minimum level (unexported helper).
// This replaces the former exported WithLevel constructor to allow adding a WithLevel LoggerOption.
func withLevelLogger(level slog.Level) *slog.Logger {
	handler := createBaseHandler(level)
	return slog.New(handler)
}

// LoggerOption is a functional option for configuring a Logger
type LoggerOption func(*Logger)

// WithCommonKeys sets custom context keys to extract from context.Context
func WithCommonKeys(keys []string) LoggerOption {
	return func(l *Logger) {
		l.commonKeys = keys
	}
}

// WithAddCommonKeys adds custom context keys to the default set of common keys
func WithAddCommonKeys(keys []string) LoggerOption {
	return func(l *Logger) {
		// Create a copy of defaultCommonKeys and append new keys
		l.commonKeys = make([]string, len(defaultCommonKeys)+len(keys))
		copy(l.commonKeys, defaultCommonKeys)
		copy(l.commonKeys[len(defaultCommonKeys):], keys)
	}
}

// WithLevel sets the minimum log level for the logger instance (functional option).
// This does not affect the global default logger's level unless used during its creation.
func WithLevel(level slog.Level) LoggerOption {
	return func(l *Logger) {
		l.Logger = withLevelLogger(level)
	}
}

// NewLogger creates a new Logger instance with the default configuration
func NewLogger(opts ...LoggerOption) *Logger {
	logger := &Logger{
		Logger:     slog.Default(),
		commonKeys: defaultCommonKeys,
	}
	for _, opt := range opts {
		opt(logger)
	}
	return logger
}

// NewLoggerWithLevel creates a new Logger instance with a specific minimum level
func NewLoggerWithLevel(level slog.Level, opts ...LoggerOption) *Logger {
	// Prepend the level option so explicit opts can override if they also set level later
	opts = append([]LoggerOption{WithLevel(level)}, opts...)
	return NewLogger(opts...)
}

// Logger methods that emit events to callbacks

// Trace logs a message at trace level
func (l *Logger) Trace(msg string, args ...any) {
	ctx := context.Background()
	l.Logger.Log(ctx, LevelTrace, msg, args...)
}

// TraceContext logs a message at trace level with context
func (l *Logger) TraceContext(ctx context.Context, msg string, args ...any) {
	l.Logger.Log(ctx, LevelTrace, msg, args...)
}

// Notice logs a message at notice level
func (l *Logger) Notice(msg string, args ...any) {
	ctx := context.Background()
	l.Logger.Log(ctx, LevelNotice, msg, args...)
}

// NoticeContext logs a message at notice level with context
func (l *Logger) NoticeContext(ctx context.Context, msg string, args ...any) {
	l.Logger.Log(ctx, LevelNotice, msg, args...)
}

// Fatal logs a message at fatal level and exits the program
func (l *Logger) Fatal(msg string, args ...any) {
	ctx := context.Background()
	l.Logger.Log(ctx, LevelFatal, msg, args...)
	panic("Fatal log called, exiting program") // Use panic to ensure all deferred functions run
}

// FatalContext logs a message at fatal level with context and exits the program
func (l *Logger) FatalContext(ctx context.Context, msg string, args ...any) {
	l.Logger.Log(ctx, LevelFatal, msg, args...)
	panic("Fatal log called, exiting program") // Use panic to ensure all deferred functions run
}
