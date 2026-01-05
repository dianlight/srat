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

// hashString computes a hash key for a message string
func hashString(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

// shouldLog checks if a message should be logged (not in cache or expired)
// and caches it with the appropriate TTL if so
func (mc *messageCache) shouldLog(msg string, severity string) bool {
	key := hashString(msg)

	// Check if entry exists and is not expired
	if entry, ok := mc.entries.Load(key); ok {
		if cacheEntry, ok := entry.(messageCacheEntry); ok {
			if time.Now().Before(cacheEntry.expiresAt) {
				return false
			}
		}
	}

	// Determine TTL based on severity
	var ttl time.Duration
	switch severity {
	case "information":
		ttl = msgCacheTTLInformation
	case "warning":
		ttl = msgCacheTTLWarning
	case "error":
		ttl = msgCacheTTLError
	default:
		ttl = msgCacheTTLDefault
	}

	// Store new entry
	mc.entries.Store(key, messageCacheEntry{expiresAt: time.Now().Add(ttl)})
	return true
}
