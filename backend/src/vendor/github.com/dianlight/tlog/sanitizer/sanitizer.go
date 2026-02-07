// Package sanitizer provides functions to mask sensitive data
// in strings, maps, slices, structs, and slog records.
package sanitizer

import (
	"context"
	"log/slog"
	"reflect"
	"strings"
)

const maskChar = "🔒"

// SensitiveKeys holds keys considered sensitive for masking.
var SensitiveKeys = map[string]struct{}{
	"password": {}, "pwd": {}, "pass": {}, "passwd": {},
	"token": {}, "jwt": {}, "auth_token": {}, "access_token": {}, "refresh_token": {},
	"key": {}, "api_key": {}, "secret": {}, "client_secret": {}, "private_key": {},
	"auth": {}, "credential": {},
	"private": {}, "confidential": {}, "secure": {}, "apikey": {},
	"bearer": {}, "authorization": {}, "salt": {}, "hash": {},
}

func MaskString(s string) string {
	return strings.Repeat(maskChar, len(s))
}

// MaskNestedValue walks nested structures and masks values whose
// immediate key matches a sensitive key.
func MaskNestedValue(v any, keyHint string) any {
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
		if _, ok := SensitiveKeys[keyHint]; ok {
			if sv, ok := v.(string); ok {
				return MaskString(sv)
			}
		}
	}

	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, vv := range val {
			out[k] = MaskNestedValue(vv, k)
		}
		return out
	case map[string]string:
		out := make(map[string]string, len(val))
		for k, vv := range val {
			if _, ok := SensitiveKeys[k]; ok {
				out[k] = MaskString(vv)
			} else {
				out[k] = vv
			}
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, vv := range val {
			out[i] = MaskNestedValue(vv, "")
		}
		return out
	case []map[string]any:
		out := make([]map[string]any, len(val))
		for i, m := range val {
			out[i] = MaskNestedValue(m, "").(map[string]any)
		}
		return out
	case []slog.Attr:
		out := make([]slog.Attr, 0, len(val))
		for _, attr := range val {
			masked := MaskNestedValue(attr.Value.Any(), attr.Key)
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
				out[i] = MaskNestedValue(rv.Index(i).Interface(), "")
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
					if before, _, ok := strings.Cut(tag, ","); ok {
						key = before
					} else {
						key = tag
					}
				}
				out[key] = MaskNestedValue(rv.Field(i).Interface(), key)
			}
			return out
		}

		return v
	}
}

// MaskingHandler wraps a slog.Handler and masks sensitive data inside records.
type MaskingHandler struct{ Next slog.Handler }

// Enabled reports whether the handler handles records at the given level.
func (mh *MaskingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return mh.Next.Enabled(ctx, level)
}

// Handle masks sensitive attributes then delegates to the wrapped handler.
func (mh *MaskingHandler) Handle(ctx context.Context, r slog.Record) error {
	nr := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	r.Attrs(func(attr slog.Attr) bool {
		mv := MaskNestedValue(attr.Value.Any(), attr.Key)
		nr.Add(slog.Any(attr.Key, mv))
		return true
	})
	return mh.Next.Handle(ctx, nr)
}

// WithAttrs returns a new MaskingHandler whose underlying handler has the given attrs.
func (mh *MaskingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &MaskingHandler{Next: mh.Next.WithAttrs(attrs)}
}

// WithGroup returns a new MaskingHandler whose underlying handler has the given group.
func (mh *MaskingHandler) WithGroup(group string) slog.Handler {
	return &MaskingHandler{Next: mh.Next.WithGroup(group)}
}
