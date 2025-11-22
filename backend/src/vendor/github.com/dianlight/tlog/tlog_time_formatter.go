package tlog

import (
	"log/slog"
	"strconv"
	"time"

	slogformatter "github.com/samber/slog-formatter"
)

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
