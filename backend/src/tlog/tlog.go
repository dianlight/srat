package tlog

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"runtime"
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

var levelColorNumbers = map[string]uint8{
	"TRACE":  7,
	"DEBUG":  6,
	"INFO":   2,
	"NOTICE": 4,
	"WARN":   3,
	"ERROR":  1,
	"FATAL":  9,
}

// FormatterConfig holds configuration for log formatting
type FormatterConfig struct {
	EnableColors      bool
	EnableFormatting  bool
	HideSensitiveData bool
	TimeFormat        string
}

// defaultFormatterConfig provides default configuration
var defaultFormatterConfig = FormatterConfig{
	EnableColors:      true, // Will be disabled automatically if terminal doesn't support colors
	EnableFormatting:  true,
	HideSensitiveData: false,
	TimeFormat:        time.RFC3339,
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
	slog.Info("Checking if terminal supports colors", "term", os.Getenv("TERM"))
	return isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd()) || strings.Contains(os.Getenv("TERM"), "color")
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
	handler := tint.NewHandler(os.Stderr, &tint.Options{
		Level:      level,
		TimeFormat: config.TimeFormat,
		NoColor:    !isTerminal || !config.EnableColors,

		AddSource: true,
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

			// Remove error.org_error from output
			if a.Key == "org_error" {
				return slog.Attr{}
			}

			return a
		},
	})

	// Composite The final Handler
	handler = slogmulti.Fanout(handler, NewEventHandler())

	// If formatting is enabled, wrap with slog-formatter
	if config.EnableFormatting {
		var formatters []slogformatter.Formatter

		// Add tozd errors formatter for enhanced error display with stacktraces
		formatters = append(formatters, TozdErrorFormatter())

		// Add generic error formatter for better error display (as fallback)
		formatters = append(formatters, ErrorFormatter("error"))

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
			handler = slogformatter.NewFormatterHandler(formatters...)(handler)
		}
	}

	return handler
}

var defaultLogger *Logger

// initializeLogger sets up the default slog configuration
func initializeLogger() {
	handler := createBaseHandler(programLevel.Level())
	defaultLogger = &Logger{
		Logger: slog.New(handler),
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

// Trace logs a message at trace level
func Trace(msg string, args ...any) {
	ctx := context.Background()
	defaultLogger.log(ctx, LevelTrace, msg, args...)
}

// TraceContext logs a message at trace level with context
func TraceContext(ctx context.Context, msg string, args ...any) {
	contextArgs := extractContextToArgs(ctx)
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
	contextArgs := extractContextToArgs(ctx)
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
	contextArgs := extractContextToArgs(ctx)
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
	contextArgs := extractContextToArgs(ctx)
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
	contextArgs := extractContextToArgs(ctx)
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
	contextArgs := extractContextToArgs(ctx)
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
	contextArgs := extractContextToArgs(ctx)
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

// WithLevel creates a logger with a specific minimum level
// This is useful for creating loggers with different levels without affecting the global logger
func WithLevel(level slog.Level) *slog.Logger {
	handler := createBaseHandler(level)
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
