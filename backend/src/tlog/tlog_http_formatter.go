package tlog

import (
	"log/slog"
	"net/http"

	slogformatter "github.com/samber/slog-formatter"
)

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
