// Package sanitizer provides functions to mask sensitive data
// in strings, maps, slices, structs, and slog records.
package sanitizer

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

const maskChar = "🔒"

func isFullyMasked(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if string(r) != maskChar {
			return false
		}
	}
	return true
}

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

var xmlTagPattern = regexp.MustCompile(`(?i)<\s*([A-Za-z0-9_.:-]+)\s*>([^<]*)<\s*/\s*([A-Za-z0-9_.:-]+)\s*>`)
var xmlCDataPattern = regexp.MustCompile(`(?is)<\s*([A-Za-z0-9_.:-]+)\s*>\s*<!\[CDATA\[(.*?)\]\]>\s*<\s*/\s*([A-Za-z0-9_.:-]+)\s*>`)
var xmlAttrPattern = regexp.MustCompile(`(?i)(\s+)([A-Za-z0-9_.:-]+)\s*=\s*("([^"]*)"|'([^']*)')`)
var keyValuePattern = regexp.MustCompile(`(?i)(^|[\s,;{\[])"?([A-Za-z0-9_.:-]+)"?\s*[:=]\s*("([^"]*)"|'([^']*)'|([^\s,\]};]+))`)
var logKeyValuePattern = regexp.MustCompile(`(?i)(^|[\s,;])((?:api[_ -]?key|password|pwd|pass(?:wd)?|token|jwt|auth[_ -]?token|access[_ -]?token|refresh[_ -]?token|client[_ -]?secret|private[_ -]?key|secret|auth|credential|bearer|authorization|salt|hash))(?:\s+(?:used|is|was|with|set|as))?(?:\s+|\s*[:=]\s*)([^\s,;]+)`)

func isSensitiveKeyLike(key string) bool {
	if key == "" {
		return false
	}
	key = strings.ToLower(key)
	if _, ok := SensitiveKeys[key]; ok {
		return true
	}
	for k := range SensitiveKeys {
		if strings.Contains(key, k) {
			return true
		}
	}
	return false
}

func adjustSensitiveJSONKeys(original any, masked any) any {
	switch ov := original.(type) {
	case map[string]any:
		maskedMap, _ := masked.(map[string]any)
		out := make(map[string]any, len(ov))
		for k, v := range ov {
			mv := maskedMap[k]
			adjustedValue := adjustSensitiveJSONKeys(v, mv)
			newKey := k
			if isSensitiveKeyLike(k) {
				if sv, ok := v.(string); ok && strings.EqualFold(sv, k) {
					newKey = MaskString(k)
				}
			}
			out[newKey] = adjustedValue
		}
		return out
	case []any:
		maskedSlice, _ := masked.([]any)
		out := make([]any, len(ov))
		for i := range ov {
			var mv any
			if i < len(maskedSlice) {
				mv = maskedSlice[i]
			}
			out[i] = adjustSensitiveJSONKeys(ov[i], mv)
		}
		return out
	default:
		return masked
	}
}

func tryMaskJSON(val string) (string, bool) {
	trimmed := strings.TrimSpace(val)
	if trimmed == "" {
		return "", false
	}

	if strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"") {
		unquoted, err := strconv.Unquote(trimmed)
		if err == nil {
			if masked, ok := tryMaskJSON(unquoted); ok {
				return strconv.Quote(masked), true
			}
		}
	}

	if !(strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") ||
		strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
		return "", false
	}

	var parsed any
	if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
		masked := MaskNestedValue(parsed, "")
		adjusted := adjustSensitiveJSONKeys(parsed, masked)
		bytes, err := json.Marshal(adjusted)
		if err == nil {
			return string(bytes), true
		}
	}

	if strings.Contains(trimmed, `\"`) || strings.Contains(trimmed, `\\`) {
		unescaped := strings.ReplaceAll(trimmed, `\"`, `"`)
		unescaped = strings.ReplaceAll(unescaped, `\\`, `\`)
		if err := json.Unmarshal([]byte(unescaped), &parsed); err == nil {
			masked := MaskNestedValue(parsed, "")
			adjusted := adjustSensitiveJSONKeys(parsed, masked)
			bytes, err := json.Marshal(adjusted)
			if err == nil {
				return string(bytes), true
			}
		}
	}

	return "", false
}

func tryMaskURL(val string) (string, bool) {
	trimmed := strings.TrimSpace(val)
	if trimmed == "" {
		return "", false
	}
	isURLish := strings.Contains(trimmed, "://") || strings.Contains(trimmed, "?") || strings.Contains(trimmed, "@") || strings.Contains(trimmed, "#")
	hasSemicolon := strings.Contains(trimmed, ";")
	if !(isURLish || (hasSemicolon && strings.Contains(trimmed, "/"))) {
		return "", false
	}

	masked := trimmed
	changed := false

	if schemeIndex := strings.Index(masked, "://"); schemeIndex != -1 {
		afterScheme := masked[schemeIndex+3:]
		atIndex := strings.Index(afterScheme, "@")
		if atIndex != -1 {
			userinfo := afterScheme[:atIndex]
			if colonIndex := strings.Index(userinfo, ":"); colonIndex != -1 {
				user := userinfo[:colonIndex]
				pass := userinfo[colonIndex+1:]
				if pass != "" {
					maskedUserinfo := user + ":" + MaskString(pass)
					masked = masked[:schemeIndex+3] + maskedUserinfo + "@" + afterScheme[atIndex+1:]
					changed = true
				}
			}
		}
	}

	queryIndex := strings.Index(masked, "?")
	fragmentIndex := strings.Index(masked, "#")
	pathEnd := len(masked)
	if queryIndex != -1 && (fragmentIndex == -1 || queryIndex < fragmentIndex) {
		pathEnd = queryIndex
	}
	if fragmentIndex != -1 && (queryIndex == -1 || fragmentIndex < queryIndex) {
		pathEnd = fragmentIndex
	}

	if semiIndex := strings.Index(masked[:pathEnd], ";"); semiIndex != -1 {
		params := masked[semiIndex+1 : pathEnd]
		if maskedParams, ok := maskDelimitedParams(params, ";"); ok {
			masked = masked[:semiIndex+1] + maskedParams + masked[pathEnd:]
			changed = true
		}
	}

	if queryIndex != -1 {
		end := len(masked)
		if fragmentIndex != -1 && fragmentIndex > queryIndex {
			end = fragmentIndex
		}
		query := masked[queryIndex+1 : end]
		if maskedQuery, ok := maskRawQuery(query); ok {
			masked = masked[:queryIndex+1] + maskedQuery + masked[end:]
			changed = true
		}
	}

	if fragmentIndex != -1 {
		fragment := masked[fragmentIndex+1:]
		if maskedFragment, ok := maskRawQuery(fragment); ok {
			masked = masked[:fragmentIndex+1] + maskedFragment
			changed = true
		}
	}

	if !changed {
		return "", false
	}
	return masked, true
}

func maskDelimitedParams(params, sep string) (string, bool) {
	if params == "" {
		return params, false
	}
	parts := strings.Split(params, sep)
	changed := false
	for i, part := range parts {
		if part == "" {
			continue
		}
		if key, value, ok := strings.Cut(part, "="); ok {
			if isSensitiveKeyLike(key) && value != "" {
				parts[i] = key + "=" + MaskString(value)
				changed = true
				continue
			}
		}

		decoded, err := url.QueryUnescape(part)
		if err != nil {
			continue
		}
		if key, value, ok := strings.Cut(decoded, "="); ok && isSensitiveKeyLike(key) {
			parts[i] = key + "=" + MaskString(value)
			changed = true
		}
	}

	if !changed {
		return params, false
	}
	return strings.Join(parts, sep), true
}

func maskRawQuery(query string) (string, bool) {
	if query == "" {
		return query, false
	}

	parts := strings.Split(query, "&")
	changed := false
	for i, part := range parts {
		if part == "" {
			continue
		}
		if key, value, ok := strings.Cut(part, "="); ok {
			if isSensitiveKeyLike(key) && value != "" {
				parts[i] = key + "=" + MaskString(value)
				changed = true
				continue
			}
		}

		decoded, err := url.QueryUnescape(part)
		if err != nil {
			continue
		}
		if key, value, ok := strings.Cut(decoded, "="); ok && isSensitiveKeyLike(key) {
			parts[i] = key + "=" + MaskString(value)
			changed = true
		}
	}

	if !changed {
		return query, false
	}
	return strings.Join(parts, "&"), true
}

func tryMaskYAML(val string) (string, bool) {
	if !strings.Contains(val, ":") {
		return "", false
	}
	if shouldSkipYAMLInline(val) {
		return "", false
	}

	lines := strings.Split(val, "\n")
	changed := false
	blockActive := false
	blockIndent := -1
	pendingIndex := -1
	pendingIsList := false
	pendingKey := ""
	pendingPrefix := ""
	pendingHasContent := false
	pendingBlockStyle := ""
	var pendingBlockLines []string

	for i, line := range lines {
		indent, prefix := leadingWhitespace(line)
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if blockActive {
			if indent > blockIndent {
				pendingBlockLines = append(pendingBlockLines, strings.TrimRight(line, " \t"))
				if maskedLine, ok := maskYAMLBlockLine(line, prefix); ok {
					lines[i] = maskedLine
					changed = true
					pendingHasContent = true
				}
				continue
			}
			if pendingIndex != -1 && pendingHasContent {
				if combined, ok := buildYAMLBlockContent(pendingBlockLines, pendingBlockStyle); ok {
					maskedCombined := MaskString(combined)
					if pendingIsList {
						lines[pendingIndex] = pendingPrefix + "- " + pendingKey + ": " + maskedCombined
					} else {
						lines[pendingIndex] = pendingPrefix + pendingKey + ": " + maskedCombined
					}
					changed = true
				}
			}
			if pendingIndex != -1 && !pendingHasContent {
				if pendingIsList {
					lines[pendingIndex] = pendingPrefix + "- " + pendingKey + ": " + maskChar
				} else {
					lines[pendingIndex] = pendingPrefix + pendingKey + ": " + maskChar
				}
				changed = true
			}
			blockActive = false
			blockIndent = -1
			pendingIndex = -1
			pendingIsList = false
			pendingKey = ""
			pendingPrefix = ""
			pendingHasContent = false
			pendingBlockStyle = ""
			pendingBlockLines = nil
		}

		isListItem := false
		lineBody := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(lineBody, "- ") {
			isListItem = true
			lineBody = strings.TrimSpace(strings.TrimPrefix(lineBody, "- "))
		}

		key, rest, ok := strings.Cut(lineBody, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" || !isSensitiveKeyLike(key) {
			continue
		}

		value := strings.TrimSpace(rest)
		if value == "" || isYAMLBlockScalar(value) {
			blockActive = true
			blockIndent = indent
			pendingIndex = i
			pendingIsList = isListItem
			pendingKey = key
			pendingPrefix = prefix
			pendingHasContent = false
			pendingBlockStyle = strings.TrimSpace(value)
			pendingBlockLines = nil
			continue
		}

		maskedValue, ok := maskYAMLScalar(value)
		if !ok {
			continue
		}

		if isListItem {
			lines[i] = prefix + "- " + key + ": " + maskedValue
		} else {
			lines[i] = prefix + key + ": " + maskedValue
		}
		changed = true
	}

	if blockActive && pendingIndex != -1 && pendingHasContent {
		if combined, ok := buildYAMLBlockContent(pendingBlockLines, pendingBlockStyle); ok {
			maskedCombined := MaskString(combined)
			if pendingIsList {
				lines[pendingIndex] = pendingPrefix + "- " + pendingKey + ": " + maskedCombined
			} else {
				lines[pendingIndex] = pendingPrefix + pendingKey + ": " + maskedCombined
			}
			changed = true
		}
	}
	if blockActive && pendingIndex != -1 && !pendingHasContent {
		if pendingIsList {
			lines[pendingIndex] = pendingPrefix + "- " + pendingKey + ": " + maskChar
		} else {
			lines[pendingIndex] = pendingPrefix + pendingKey + ": " + maskChar
		}
		changed = true
	}

	if !changed {
		return "", false
	}
	return strings.Join(lines, "\n"), true
}

func leadingWhitespace(line string) (int, string) {
	count := 0
	for count < len(line) {
		switch line[count] {
		case ' ', '\t':
			count++
		default:
			return count, line[:count]
		}
	}
	return count, line
}

func isYAMLBlockScalar(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "|" || trimmed == ">" {
		return true
	}
	return strings.HasPrefix(trimmed, "|") || strings.HasPrefix(trimmed, ">")
}

func shouldSkipYAMLInline(val string) bool {
	trimmed := strings.TrimSpace(val)
	if strings.Contains(trimmed, "\n") {
		return false
	}
	if strings.HasPrefix(trimmed, "map[") {
		return true
	}

	lineBody := strings.TrimLeft(trimmed, " \t")
	if strings.HasPrefix(lineBody, "- ") {
		lineBody = strings.TrimSpace(strings.TrimPrefix(lineBody, "- "))
	}

	_, rest, ok := strings.Cut(lineBody, ":")
	if !ok {
		return false
	}
	restTrim := strings.TrimSpace(rest)
	if restTrim == "" || isYAMLBlockScalar(restTrim) {
		return false
	}
	if strings.HasPrefix(restTrim, "{") || strings.HasPrefix(restTrim, "[") {
		return false
	}

	return keyValuePattern.MatchString(restTrim)
}

func buildYAMLBlockContent(lines []string, style string) (string, bool) {
	if len(lines) == 0 {
		return "", false
	}
	trimmedStyle := strings.TrimSpace(style)
	isFolded := strings.HasPrefix(trimmedStyle, ">")

	if isFolded {
		parts := make([]string, 0, len(lines))
		for _, line := range lines {
			parts = append(parts, strings.TrimSpace(line))
		}
		return strings.Join(parts, " "), true
	}

	parts := make([]string, 0, len(lines))
	for i, line := range lines {
		content := strings.TrimRight(line, " \t")
		if i == 0 {
			content = strings.TrimLeft(content, " \t")
		}
		parts = append(parts, content)
	}
	return strings.Join(parts, "\n"), true
}

func splitYAMLComment(value string) (string, string, bool) {
	inSingle := false
	inDouble := false
	for i := 0; i < len(value); i++ {
		switch value[i] {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '#':
			if inSingle || inDouble {
				continue
			}
			start := i
			for start > 0 {
				prev := value[start-1]
				if prev != ' ' && prev != '\t' {
					break
				}
				start--
			}
			if start == i {
				if i == 0 {
					return "", value, true
				}
				return value, "", false
			}
			content := strings.TrimRight(value[:start], " \t")
			return content, value[start:], true
		}
	}
	return value, "", false
}

func maskYAMLScalar(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}

	content, comment, hasComment := splitYAMLComment(trimmed)
	if hasComment {
		if strings.TrimSpace(content) == "" {
			return maskChar + comment, true
		}
		masked, ok := maskYAMLScalarContent(content)
		if !ok {
			return "", false
		}
		return masked + comment, true
	}

	return maskYAMLScalarContent(trimmed)
}

func maskYAMLScalarContent(trimmed string) (string, bool) {

	if masked, ok := maskKeyValuePairs(trimmed); ok {
		return masked, true
	}

	if strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"") && len(trimmed) >= 2 {
		inner := trimmed[1 : len(trimmed)-1]
		return "\"" + MaskString(inner) + "\"", true
	}
	if strings.HasPrefix(trimmed, "'") && strings.HasSuffix(trimmed, "'") && len(trimmed) >= 2 {
		inner := trimmed[1 : len(trimmed)-1]
		return "'" + MaskString(inner) + "'", true
	}

	return MaskString(trimmed), true
}

func maskYAMLBlockLine(line, prefix string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return line, false
	}

	lineBody := strings.TrimLeft(line, " \t")
	if strings.HasPrefix(lineBody, "- ") {
		item := strings.TrimSpace(strings.TrimPrefix(lineBody, "- "))
		if item == "" {
			return line, false
		}
		return prefix + "- " + MaskString(item), true
	}

	return prefix + MaskString(trimmed), true
}

func maskXMLTags(val string) (string, bool) {
	masked := val
	changed := false

	if attrMasked, ok := maskXMLAttributes(masked); ok {
		masked = attrMasked
		changed = true
	}

	if cdataMasked, ok := maskXMLTagContent(masked, xmlCDataPattern); ok {
		masked = cdataMasked
		changed = true
	}

	if textMasked, ok := maskXMLTagContent(masked, xmlTagPattern); ok {
		masked = textMasked
		changed = true
	}

	if !changed {
		return val, false
	}
	return masked, true
}

func maskXMLAttributes(val string) (string, bool) {
	matches := xmlAttrPattern.FindAllStringSubmatchIndex(val, -1)
	if len(matches) == 0 {
		return val, false
	}

	var b strings.Builder
	last := 0
	changed := false
	for _, match := range matches {
		keyStart, keyEnd := match[4], match[5]
		if keyStart < 0 {
			continue
		}
		key := val[keyStart:keyEnd]
		if !isSensitiveKeyLike(key) {
			continue
		}

		valueStart, valueEnd := -1, -1
		if len(match) >= 10 && match[8] >= 0 {
			valueStart = match[8]
			valueEnd = match[9]
		}
		if valueStart < 0 && len(match) >= 12 && match[10] >= 0 {
			valueStart = match[10]
			valueEnd = match[11]
		}
		if valueStart < 0 {
			continue
		}
		if isFullyMasked(val[valueStart:valueEnd]) {
			continue
		}

		b.WriteString(val[last:valueStart])
		b.WriteString(MaskString(val[valueStart:valueEnd]))
		last = valueEnd
		changed = true
	}

	if !changed {
		return val, false
	}
	b.WriteString(val[last:])
	return b.String(), true
}

func maskXMLTagContent(val string, pattern *regexp.Regexp) (string, bool) {
	matches := pattern.FindAllStringSubmatchIndex(val, -1)
	if len(matches) == 0 {
		return val, false
	}

	var b strings.Builder
	last := 0
	changed := false
	for _, match := range matches {
		keyStart, keyEnd := match[2], match[3]
		valueStart, valueEnd := match[4], match[5]
		endKeyStart, endKeyEnd := match[6], match[7]
		if keyStart < 0 || valueStart < 0 || endKeyStart < 0 {
			continue
		}
		key := val[keyStart:keyEnd]
		endKey := val[endKeyStart:endKeyEnd]
		if !strings.EqualFold(key, endKey) {
			continue
		}
		if !isSensitiveKeyLike(key) {
			continue
		}
		b.WriteString(val[last:valueStart])
		b.WriteString(MaskString(val[valueStart:valueEnd]))
		last = valueEnd
		changed = true
	}
	if !changed {
		return val, false
	}
	b.WriteString(val[last:])
	return b.String(), true
}

func maskKeyValuePairs(val string) (string, bool) {
	matches := keyValuePattern.FindAllStringSubmatchIndex(val, -1)
	if len(matches) == 0 {
		return val, false
	}

	var b strings.Builder
	last := 0
	changed := false
	for _, match := range matches {
		if len(match) < 6 {
			continue
		}
		keyStart, keyEnd := match[4], match[5]
		if keyStart < 0 {
			continue
		}
		key := val[keyStart:keyEnd]
		if !isSensitiveKeyLike(key) {
			continue
		}

		valueStart, valueEnd := -1, -1
		if len(match) >= 10 && match[8] >= 0 {
			valueStart = match[8]
			valueEnd = match[9]
		}
		if valueStart < 0 && len(match) >= 12 && match[10] >= 0 {
			valueStart = match[10]
			valueEnd = match[11]
		}
		if valueStart < 0 && len(match) >= 14 && match[12] >= 0 {
			valueStart = match[12]
			valueEnd = match[13]
		}
		if valueStart < 0 {
			continue
		}

		b.WriteString(val[last:valueStart])
		b.WriteString(MaskString(val[valueStart:valueEnd]))
		last = valueEnd
		changed = true
	}
	if !changed {
		return val, false
	}
	b.WriteString(val[last:])
	return b.String(), true
}

func maskLogValueToken(token string) (string, bool) {
	if token == "" {
		return token, false
	}

	prefix := ""
	suffix := ""
	core := token

	for len(core) > 0 {
		last := core[len(core)-1]
		switch last {
		case ',', ';', ')', ']', '}':
			suffix = string(last) + suffix
			core = core[:len(core)-1]
		default:
			goto trailingDone
		}
	}

trailingDone:
	if len(core) >= 2 {
		if (core[0] == '"' && core[len(core)-1] == '"') || (core[0] == '\'' && core[len(core)-1] == '\'') {
			prefix = core[:1]
			suffix = core[len(core)-1:] + suffix
			core = core[1 : len(core)-1]
		}
	}

	if core == "" {
		return token, false
	}
	if isFullyMasked(core) {
		return token, false
	}

	return prefix + MaskString(core) + suffix, true
}

func tryMaskLog(val string) (string, bool) {
	trimmed := strings.TrimSpace(val)
	if trimmed == "" {
		return "", false
	}

	matches := logKeyValuePattern.FindAllStringSubmatchIndex(val, -1)
	if len(matches) == 0 {
		return "", false
	}

	var b strings.Builder
	last := 0
	changed := false

	for _, match := range matches {
		if len(match) < 8 {
			continue
		}
		keyStart, keyEnd := match[4], match[5]
		valueStart, valueEnd := match[6], match[7]
		if keyStart < 0 || valueStart < 0 {
			continue
		}

		key := val[keyStart:keyEnd]
		if !isSensitiveKeyLike(key) {
			continue
		}

		rawValue := val[valueStart:valueEnd]
		maskedValue, ok := maskLogValueToken(rawValue)
		if !ok || maskedValue == rawValue {
			continue
		}

		b.WriteString(val[last:valueStart])
		b.WriteString(maskedValue)
		last = valueEnd
		changed = true
	}

	if !changed {
		return "", false
	}

	b.WriteString(val[last:])
	return b.String(), true
}

func maskStringValue(val string) string {
	if masked, ok := tryMaskJSON(val); ok {
		return masked
	}
	if masked, ok := tryMaskURL(val); ok {
		return masked
	}
	normalized := val
	if strings.Contains(val, `\n`) || strings.Contains(val, `\t`) || strings.Contains(val, `\r`) {
		replacer := strings.NewReplacer(`\n`, "\n", `\t`, "\t", `\r`, "\r")
		normalized = replacer.Replace(val)
	}
	if masked, ok := tryMaskYAML(normalized); ok {
		if strings.Contains(masked, "=") {
			if kvMasked, kvOk := maskKeyValuePairs(masked); kvOk {
				return kvMasked
			}
		}
		return masked
	}
	if masked, ok := maskXMLTags(normalized); ok {
		return masked
	}
	masked := normalized
	if kvMasked, ok := maskKeyValuePairs(masked); ok {
		masked = kvMasked
	}
	if logMasked, ok := tryMaskLog(masked); ok {
		return logMasked
	}
	if masked != normalized {
		return masked
	}
	return val
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
	case string:
		return maskStringValue(val)
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, vv := range val {
			if isSensitiveKeyLike(k) {
				if sv, ok := vv.(string); ok {
					out[k] = MaskString(sv)
					continue
				}
			}
			out[k] = MaskNestedValue(vv, k)
		}
		return out
	case map[string]string:
		out := make(map[string]string, len(val))
		for k, vv := range val {
			if isSensitiveKeyLike(k) {
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
