package smartmontools

import (
	"hash/fnv"
	"sync"
	"time"
)

// Message cache TTL durations by severity
const (
	msgCacheTTLInformation = 1 * time.Hour
	msgCacheTTLWarning     = 30 * time.Minute
	msgCacheTTLError       = 5 * time.Minute
	msgCacheTTLDefault     = 2 * time.Hour
)

// messageCacheEntry holds a cached message with its expiration time
type messageCacheEntry struct {
	expiresAt time.Time
}

// messageCache provides a simple TTL-based cache for deduplicating messages
type messageCache struct {
	entries sync.Map
}

// globalMessageCache is the package-level cache for smartctl messages
var globalMessageCache = &messageCache{}

// hashString computes a hash key for a message string using FNV-1a algorithm
// optimized for performance with minimal allocations
func hashString(s string) uint64 {
	h := fnv.New64a()
	// Using Write instead of WriteString avoids temporary allocation in some Go versions
	h.Write([]byte(s))
	return h.Sum64()
}

// getTTL returns the appropriate cache TTL based on severity level
// Optimized to reduce string comparisons
func getTTL(severity string) time.Duration {
	switch severity {
	case "information":
		return msgCacheTTLInformation
	case "warning":
		return msgCacheTTLWarning
	case "error":
		return msgCacheTTLError
	default:
		return msgCacheTTLDefault
	}
}

// shouldLog checks if a message should be logged (not in cache or expired)
// and caches it with the appropriate TTL if so.
// Optimized to reduce allocations and improve performance.
func (mc *messageCache) shouldLog(msg string, severity string) bool {
	key := hashString(msg)
	now := time.Now()

	// Check if entry exists and is not expired
	if entry, ok := mc.entries.Load(key); ok {
		if cacheEntry, ok := entry.(messageCacheEntry); ok {
			if now.Before(cacheEntry.expiresAt) {
				return false
			}
		}
	}

	// Determine TTL based on severity
	ttl := getTTL(severity)

	// Store new entry
	mc.entries.Store(key, messageCacheEntry{expiresAt: now.Add(ttl)})
	return true
}
