package tlog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"
)

// RateLimitConfig configures the LowRate logging variants.
//
// A LowRate log emits the first occurrence of a log line immediately, then
// silences identical lines for Window. At the end of each window a synthetic
// summary line is fabricated (even when no real log arrives), carrying the
// number of silenced lines both as a message prefix and as a structured
// attribute.
type RateLimitConfig struct {
	// Window is the minimum interval between two emissive log lines. A value
	// <= 0 disables deduplication (every call is emitted).
	Window time.Duration

	// IgnoreKeys lists attribute keys excluded from the deduplication
	// fingerprint. It is optional; when nil it defaults to {"time"}. The
	// "time" key is always excluded, since it is time-variant by definition.
	IgnoreKeys []string

	// SuppressedFormat is a fmt format with one %d verb used as the prefix of
	// a fabricated summary. Optional; defaults to "[%d silenced] ".
	SuppressedFormat string

	// SilencedKey is the attribute key used for the silenced count. Optional;
	// defaults to "silenced".
	SilencedKey string
}

const (
	defaultSuppressedFormat = "[%d silenced] "
	defaultSilencedKey      = "silenced"
)

func normalizeRateLimitConfig(cfg RateLimitConfig) RateLimitConfig {
	if cfg.SuppressedFormat == "" {
		cfg.SuppressedFormat = defaultSuppressedFormat
	}
	if cfg.SilencedKey == "" {
		cfg.SilencedKey = defaultSilencedKey
	}

	// The record timestamp must never take part in the fingerprint. Keep the
	// caller's list, but always add the standard time key to it.
	keys := make([]string, 0, len(cfg.IgnoreKeys)+1)
	keys = append(keys, cfg.IgnoreKeys...)
	hasTime := false
	for _, k := range keys {
		if k == slog.TimeKey {
			hasTime = true
			break
		}
	}
	if !hasTime {
		keys = append(keys, slog.TimeKey)
	}
	cfg.IgnoreKeys = keys

	return cfg
}

// context plumbing ---------------------------------------------------------

type rateLimitContextKey struct{}

type rateLimitContextValue struct {
	cfg      RateLimitConfig
	original context.Context // caller context without the internal config value
}

func withRateLimit(ctx context.Context, cfg RateLimitConfig) context.Context {
	return context.WithValue(ctx, rateLimitContextKey{}, rateLimitContextValue{
		cfg:      cfg,
		original: ctx,
	})
}

func rateLimitFromContext(ctx context.Context) (rateLimitContextValue, bool) {
	if ctx == nil {
		return rateLimitContextValue{}, false
	}
	v, ok := ctx.Value(rateLimitContextKey{}).(rateLimitContextValue)
	return v, ok
}

// fingerprint --------------------------------------------------------------

// rateLimitFingerprint builds a stable identity for a log line from its
// message and attributes only. The timestamp, source location and any
// caller-ignored keys are excluded, so the same line deduplicates across
// calls. The window is part of the key so distinct configs stay independent.
func rateLimitFingerprint(msg string, args []any, cfg RateLimitConfig) string {
	parts := make([]string, 0, len(args)/2+2)
	parts = append(parts, "m="+msg)
	for i := 0; i+1 < len(args); i += 2 {
		key := fmt.Sprint(args[i])
		if key == slog.TimeKey || containsString(cfg.IgnoreKeys, key) {
			continue
		}
		parts = append(parts, key+"="+fmt.Sprintf("%v", args[i+1]))
	}
	parts = append(parts, fmt.Sprintf("w=%d", int64(cfg.Window)))
	sort.Strings(parts)

	h := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(h[:])
}

func containsString(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}

// state and central limiter -------------------------------------------------

type rateLimitState struct {
	logger   *Logger
	ctx      context.Context
	level    slog.Level
	msg      string
	args     []any
	pc       uintptr
	cfg      RateLimitConfig
	deadline time.Time
	silenced uint64
	timer    *time.Timer
	gen      uint64
}

type rateLimiter struct {
	mu     sync.Mutex
	states map[string]*rateLimitState
}

var defaultRateLimiter = &rateLimiter{states: make(map[string]*rateLimitState)}

// filter decides whether a log line should be emitted and rewrites it when a
// window expires with silenced lines. The caller must have already checked
// that the level is enabled. now is the record timestamp.
func (rl *rateLimiter) filter(l *Logger, ctx context.Context, cfg RateLimitConfig, level slog.Level, msg string, args []any, pc uintptr, now time.Time) (string, []any, bool) {
	key := rateLimitFingerprint(msg, args, cfg)

	rl.mu.Lock()
	st := rl.states[key]
	if st == nil {
		st = &rateLimitState{
			logger: l,
			ctx:    ctx,
			level:  level,
			msg:    msg,
			args:   cloneArgs(args),
			pc:     pc,
			cfg:    cfg,
		}
		st.deadline = now.Add(cfg.Window)
		st.gen = 1
		rl.states[key] = st
		rl.schedule(key, st)
		rl.mu.Unlock()
		return msg, args, true
	}

	if now.Before(st.deadline) {
		st.silenced++
		rl.mu.Unlock()
		return "", nil, false
	}

	// The deadline passed, so this real arrival wins the race with the timer.
	n := st.silenced
	st.silenced = 0
	st.deadline = now.Add(cfg.Window)
	st.gen++
	st.logger, st.ctx, st.level = l, ctx, level
	st.msg, st.args, st.pc, st.cfg = msg, cloneArgs(args), pc, cfg
	rl.schedule(key, st)
	rl.mu.Unlock()

	if n > 0 {
		newMsg, newArgs := withSuppressed(cfg, msg, args, n)
		return newMsg, newArgs, true
	}
	return msg, args, true
}

// onDeadline fabricates the summary for an expired window. It runs in the
// timer goroutine; all state access is serialized by rl.mu.
func (rl *rateLimiter) onDeadline(key string, gen uint64) {
	rl.mu.Lock()
	st := rl.states[key]
	if st == nil || st.gen != gen {
		rl.mu.Unlock()
		return
	}

	now := time.Now()
	if now.Before(st.deadline) {
		rl.schedule(key, st)
		rl.mu.Unlock()
		return
	}

	n := st.silenced
	if n == 0 {
		// Nothing was silenced: clear the window and let the next real log
		// start a fresh one (step 1 again).
		delete(rl.states, key)
		rl.mu.Unlock()
		return
	}

	st.silenced = 0
	st.deadline = now.Add(st.cfg.Window) // anchor at the actual emit time
	st.gen++
	msg, args := withSuppressed(st.cfg, st.msg, st.args, n)
	l, ctx, level, pc := st.logger, st.ctx, st.level, st.pc
	rl.schedule(key, st)
	rl.mu.Unlock()

	// The stored context has no rate-limit config, so this bypasses filtering.
	l.logRecord(ctx, level, msg, pc, now, args...)
}

// schedule (re)arms the deadline timer. Must be called with rl.mu held.
func (rl *rateLimiter) schedule(key string, st *rateLimitState) {
	if st.timer != nil {
		st.timer.Stop()
	}
	delay := time.Until(st.deadline)
	if delay < 0 {
		delay = 0
	}
	gen := st.gen
	st.timer = time.AfterFunc(delay, func() { rl.onDeadline(key, gen) })
}

func withSuppressed(cfg RateLimitConfig, msg string, args []any, n uint64) (string, []any) {
	newMsg := fmt.Sprintf(cfg.SuppressedFormat, n) + msg
	newArgs := cloneArgs(args)
	if cfg.SilencedKey != "" {
		newArgs = append(newArgs, cfg.SilencedKey, int(n))
	}
	return newMsg, newArgs
}

func cloneArgs(args []any) []any {
	out := make([]any, len(args), len(args)+2)
	copy(out, args)
	return out
}

// ResetRateLimit clears all pending rate-limit windows, stopping their timers.
// The next LowRate call for any line behaves like a first occurrence.
func ResetRateLimit() {
	rl := defaultRateLimiter
	rl.mu.Lock()
	for key, st := range rl.states {
		if st.timer != nil {
			st.timer.Stop()
		}
		delete(rl.states, key)
	}
	rl.mu.Unlock()
}

// LowRate logging functions -------------------------------------------------
//
// One variant per level and context style. Each attaches the config to the
// context and delegates to the central logging path, where the shared limiter
// applies the deduplication. Existing logging functions are unaffected.

// TraceLowRate logs a message at trace level with rate limiting.
func TraceLowRate(cfg RateLimitConfig, msg string, args ...any) {
	defaultLogger.log(withRateLimit(context.Background(), cfg), LevelTrace, msg, args...)
}

// TraceLowRateContext logs a message at trace level with context and rate limiting.
func TraceLowRateContext(ctx context.Context, cfg RateLimitConfig, msg string, args ...any) {
	contextArgs := extractContextToArgs(ctx, defaultLogger.commonKeys)
	allArgs := append(args, contextArgs...)
	defaultLogger.log(withRateLimit(ctx, cfg), LevelTrace, msg, allArgs...)
}

// DebugLowRate logs a message at debug level with rate limiting.
func DebugLowRate(cfg RateLimitConfig, msg string, args ...any) {
	defaultLogger.log(withRateLimit(context.Background(), cfg), slog.LevelDebug, msg, args...)
}

// DebugLowRateContext logs a message at debug level with context and rate limiting.
func DebugLowRateContext(ctx context.Context, cfg RateLimitConfig, msg string, args ...any) {
	contextArgs := extractContextToArgs(ctx, defaultLogger.commonKeys)
	allArgs := append(args, contextArgs...)
	defaultLogger.log(withRateLimit(ctx, cfg), slog.LevelDebug, msg, allArgs...)
}

// InfoLowRate logs a message at info level with rate limiting.
func InfoLowRate(cfg RateLimitConfig, msg string, args ...any) {
	defaultLogger.log(withRateLimit(context.Background(), cfg), slog.LevelInfo, msg, args...)
}

// InfoLowRateContext logs a message at info level with context and rate limiting.
func InfoLowRateContext(ctx context.Context, cfg RateLimitConfig, msg string, args ...any) {
	contextArgs := extractContextToArgs(ctx, defaultLogger.commonKeys)
	allArgs := append(args, contextArgs...)
	defaultLogger.log(withRateLimit(ctx, cfg), slog.LevelInfo, msg, allArgs...)
}

// NoticeLowRate logs a message at notice level with rate limiting.
func NoticeLowRate(cfg RateLimitConfig, msg string, args ...any) {
	defaultLogger.log(withRateLimit(context.Background(), cfg), LevelNotice, msg, args...)
}

// NoticeLowRateContext logs a message at notice level with context and rate limiting.
func NoticeLowRateContext(ctx context.Context, cfg RateLimitConfig, msg string, args ...any) {
	contextArgs := extractContextToArgs(ctx, defaultLogger.commonKeys)
	allArgs := append(args, contextArgs...)
	defaultLogger.log(withRateLimit(ctx, cfg), LevelNotice, msg, allArgs...)
}

// WarnLowRate logs a message at warning level with rate limiting.
func WarnLowRate(cfg RateLimitConfig, msg string, args ...any) {
	defaultLogger.log(withRateLimit(context.Background(), cfg), slog.LevelWarn, msg, args...)
}

// WarnLowRateContext logs a message at warning level with context and rate limiting.
func WarnLowRateContext(ctx context.Context, cfg RateLimitConfig, msg string, args ...any) {
	contextArgs := extractContextToArgs(ctx, defaultLogger.commonKeys)
	allArgs := append(args, contextArgs...)
	defaultLogger.log(withRateLimit(ctx, cfg), slog.LevelWarn, msg, allArgs...)
}

// ErrorLowRate logs a message at error level with rate limiting.
func ErrorLowRate(cfg RateLimitConfig, msg string, args ...any) {
	defaultLogger.log(withRateLimit(context.Background(), cfg), slog.LevelError, msg, args...)
}

// ErrorLowRateContext logs a message at error level with context and rate limiting.
func ErrorLowRateContext(ctx context.Context, cfg RateLimitConfig, msg string, args ...any) {
	contextArgs := extractContextToArgs(ctx, defaultLogger.commonKeys)
	allArgs := append(args, contextArgs...)
	defaultLogger.log(withRateLimit(ctx, cfg), slog.LevelError, msg, allArgs...)
}
