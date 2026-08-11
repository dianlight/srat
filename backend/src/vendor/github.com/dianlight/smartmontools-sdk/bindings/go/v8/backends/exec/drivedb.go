// SPDX-License-Identifier: GPL-2.0-or-later

// Package smartmontools provides Go bindings for interfacing with smartmontools
// to monitor and manage storage device health using S.M.A.R.T. data.
//
// This file contains functions for parsing and managing the embedded drivedb.h
// database from smartmontools, which includes USB bridge device mappings.
package exec

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"

	"github.com/dianlight/tlog"
)

// drivedbCache holds the parsed drivedb entries to avoid reparsing on each access.
var drivedbCache map[string]string

// cloneDeviceTypeCache returns a copy of the global drivedb cache.
// This prevents per-client mutations from affecting other clients.
func cloneDeviceTypeCache() map[string]string {
	if drivedbCache == nil {
		return make(map[string]string)
	}
	copyCache := make(map[string]string, len(drivedbCache))
	for key, value := range drivedbCache {
		copyCache[key] = value
	}
	return copyCache
}

func init() {
	drivedbCache = loadDrivedbAddendum()
}

//go:embed drivedb.h
var drivedbH string

// loadDrivedbAddendum parses the embedded drivedb.h file from smartmontools
// and returns a map of USB device identifiers to device types.
//
// The drivedb.h file contains C-style struct entries. USB entries have:
//   - modelfamily starting with "USB:"
//   - modelregexp containing USB vendor:product ID (e.g., "0x152d:0x0578")
//   - presets containing device type after "-d " (e.g., "-d sat")
//
// Returns a map with keys in format "usb:0x152d:0x0578" -> device type "sat"
func loadDrivedbAddendum() map[string]string {
	cache := make(map[string]string)

	// Regular expressions to parse drivedb.h entries
	// Match entries starting with { "USB:
	usbEntryPattern := regexp.MustCompile(`\{\s*"USB:`)
	// Match quoted strings (for modelfamily, modelregexp, firmwareregexp, warningmsg, presets)
	quotedStringPattern := regexp.MustCompile(`"([^"]*)"`)
	// Match device type in presets: -d <type> (may have options like "sat,12")
	deviceTypePattern := regexp.MustCompile(`-d\s+(\S+)`)

	// Split into lines and process
	lines := strings.Split(drivedbH, "\n")
	var inUSBEntry bool
	var currentFields []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check if this is the start of a USB entry
		if usbEntryPattern.MatchString(line) {
			inUSBEntry = true
			currentFields = []string{}
		}

		if inUSBEntry {
			// Extract all quoted strings from this line
			matches := quotedStringPattern.FindAllStringSubmatch(line, -1)
			for _, match := range matches {
				if len(match) > 1 {
					currentFields = append(currentFields, match[1])
				}
			}

			// Check if we've reached the end of this entry (closing brace)
			if strings.Contains(line, "},") || (strings.Contains(line, "}") && !strings.Contains(line, "{")) {
				inUSBEntry = false

				// Process the complete entry
				// Expected fields: [modelfamily, modelregexp, firmwareregexp, warningmsg, presets]
				if len(currentFields) >= 5 {
					modelfamily := currentFields[0]
					modelregexp := currentFields[1]
					presets := currentFields[4]

					// Only process USB entries
					if strings.HasPrefix(modelfamily, "USB:") {
						// Extract device type from presets
						deviceTypeMatch := deviceTypePattern.FindStringSubmatch(presets)
						if len(deviceTypeMatch) > 1 {
							deviceType := deviceTypeMatch[1]
							// Remove any options after comma (e.g., "sat,12" -> "sat")
							if commaIdx := strings.Index(deviceType, ","); commaIdx != -1 {
								deviceType = deviceType[:commaIdx]
							}

							// Parse USB vendor:product IDs from modelregexp
							// The modelregexp can contain simple IDs like "0x152d:0x0578"
							// or regex patterns like "0x152d:0x05(7[789]|80)"
							// For simplicity, we'll extract exact matches and simple patterns
							usbIDs := extractUSBIDs(modelregexp)
							for _, usbID := range usbIDs {
								key := "usb:" + strings.ToLower(usbID)
								cache[key] = deviceType
							}
						}
					}
				}
				currentFields = []string{}
			}
		}
	}

	tlog.Debug("Loaded drivedb from smartmontools drivedb.h", "entries", len(cache))
	return cache
}

// usbIDPairPattern matches a USB vendor:product ID pair where either side
// may be a plain hex literal or a limited regex (nested parenthesised
// alternation and/or bracket character classes), e.g.:
//
//	0x152d:0x0578               plain
//	0x(0bda|0dd8):0x9210        vendor alternatives
//	0x152d:0x05(7[789]|80)      product alternatives with a nested class
//	0x045b:0x022[9a]            product class with no enclosing parens
//	0x1e91:0x(a([23]a5|4a[7e])|de46)   arbitrarily nested
//
// Wildcard product IDs like "0x0480:0x...." contain no chars from this set
// after "0x", so they never match and are silently skipped, same as before.
var usbIDPairPattern = regexp.MustCompile(`0x([0-9a-fA-F()|\[\]]+):0x([0-9a-fA-F()|\[\]]+)`)

// extractUSBIDs extracts USB vendor:product IDs from a modelregexp pattern.
// Returns a slice of IDs in format "0xVVVV:0xPPPP", expanding any
// parenthesised alternation or bracket character class on either side of
// the pair into every concrete hex value it can produce.
func extractUSBIDs(modelregexp string) []string {
	var ids []string

	for _, match := range usbIDPairPattern.FindAllStringSubmatch(modelregexp, -1) {
		vendors := expandHexPattern(match[1])
		products := expandHexPattern(match[2])
		for _, vendor := range vendors {
			for _, product := range products {
				ids = append(ids, "0x"+vendor+":0x"+product)
			}
		}
	}

	return ids
}

// expandHexPattern expands a limited regex over hex digits — literal
// characters, bracket character classes ("[9a]"), and arbitrarily nested
// parenthesised alternation ("(a|b|c)") — into every concrete string it can
// produce. It has no notion of hex specifically; any character other than
// '(', ')', '|', '[', ']' is treated as a literal to concatenate.
func expandHexPattern(pattern string) []string {
	p := &hexPatternParser{s: pattern}
	return p.parseAlternation()
}

type hexPatternParser struct {
	s   string
	pos int
}

// parseAlternation parses "concat1|concat2|...", stopping at ')' or end of
// input, and returns the union of each alternative's expansions.
func (p *hexPatternParser) parseAlternation() []string {
	results := p.parseConcat()
	for p.pos < len(p.s) && p.s[p.pos] == '|' {
		p.pos++
		results = append(results, p.parseConcat()...)
	}
	return results
}

// parseConcat parses a sequence of atoms until '|', ')', or end of input,
// and returns the cartesian product of their expansions.
func (p *hexPatternParser) parseConcat() []string {
	results := []string{""}
	for p.pos < len(p.s) {
		switch p.s[p.pos] {
		case '|', ')':
			return results
		case '(':
			p.pos++
			inner := p.parseAlternation()
			if p.pos < len(p.s) && p.s[p.pos] == ')' {
				p.pos++
			}
			results = cartesianConcat(results, inner)
		case '[':
			end := strings.IndexByte(p.s[p.pos:], ']')
			if end == -1 {
				results = cartesianConcat(results, []string{p.s[p.pos:]})
				p.pos = len(p.s)
				continue
			}
			chars := p.s[p.pos+1 : p.pos+end]
			alternatives := make([]string, 0, len(chars))
			for _, c := range chars {
				alternatives = append(alternatives, string(c))
			}
			results = cartesianConcat(results, alternatives)
			p.pos += end + 1
		default:
			results = cartesianConcat(results, []string{string(p.s[p.pos])})
			p.pos++
		}
	}
	return results
}

func cartesianConcat(prefixes, suffixes []string) []string {
	out := make([]string, 0, len(prefixes)*len(suffixes))
	for _, prefix := range prefixes {
		for _, suffix := range suffixes {
			out = append(out, prefix+suffix)
		}
	}
	return out
}

// isUnknownUSBBridge checks if the smartctl messages contain an "Unknown USB bridge" error
func isUnknownUSBBridge(smartInfo *SMARTInfo) bool {
	if smartInfo == nil || smartInfo.Smartctl == nil {
		return false
	}
	for _, msg := range smartInfo.Smartctl.Messages {
		if strings.Contains(msg.String, "Unknown USB bridge") {
			return true
		}
	}
	return false
}

// extractUSBBridgeID extracts the USB vendor:product ID from an "Unknown USB bridge" error message.
// Returns the ID in the format "usb:0xVVVV:0xPPPP" or an empty string if not found.
func extractUSBBridgeID(smartInfo *SMARTInfo) string {
	if smartInfo == nil || smartInfo.Smartctl == nil {
		return ""
	}

	// Pattern to match: "Unknown USB bridge [0x152d:0x578e ..."
	re := regexp.MustCompile(`Unknown USB bridge \[(0x[0-9a-fA-F]+):(0x[0-9a-fA-F]+)`)

	for _, msg := range smartInfo.Smartctl.Messages {
		if matches := re.FindStringSubmatch(msg.String); len(matches) >= 3 {
			vendorID := strings.ToLower(matches[1])
			productID := strings.ToLower(matches[2])
			return fmt.Sprintf("usb:%s:%s", vendorID, productID)
		}
	}
	return ""
}
