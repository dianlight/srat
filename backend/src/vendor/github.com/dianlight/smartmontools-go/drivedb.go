// Package smartmontools provides Go bindings for interfacing with smartmontools
// to monitor and manage storage device health using S.M.A.R.T. data.
//
// This file contains functions for parsing and managing the embedded drivedb.h
// database from smartmontools, which includes USB bridge device mappings.
package smartmontools

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"

	"github.com/dianlight/tlog"
)

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

// extractUSBIDs extracts USB vendor:product IDs from a modelregexp pattern.
// Returns a slice of IDs in format "0xVVVV:0xPPPP"
// Handles both exact matches and expands common regex patterns.
func extractUSBIDs(modelregexp string) []string {
	var ids []string

	// Pattern to match USB IDs with exact hex: 0xVVVV:0xPPPP
	exactPattern := regexp.MustCompile(`(0x[0-9a-fA-F]{4}:0x[0-9a-fA-F]{4})`)
	matches := exactPattern.FindAllString(modelregexp, -1)
	ids = append(ids, matches...)

	// Handle common regex patterns in product ID
	// Pattern like "0x152d:0x05(7[789]|80)" -> expand to 0x0577, 0x0578, 0x0579, 0x0580
	regexPattern := regexp.MustCompile(`(0x[0-9a-fA-F]{4}):0x([0-9a-fA-F]{2})\(([^\)]+)\)`)
	regexMatches := regexPattern.FindAllStringSubmatch(modelregexp, -1)
	for _, match := range regexMatches {
		if len(match) >= 4 {
			vendor := match[1]
			prefix := match[2]
			alternatives := match[3]

			// Handle alternatives like "7[789]|80"
			// Split by |
			parts := strings.Split(alternatives, "|")
			for _, part := range parts {
				expandedIDs := expandProductIDPattern(vendor, prefix, part)
				ids = append(ids, expandedIDs...)
			}
		}
	}

	// Handle patterns like "0x0480:0x...." (wildcard) - these are too broad to expand
	// We'll skip these for now as they would create too many entries

	return ids
}

// expandProductIDPattern expands a product ID pattern like "7[789]" to actual hex values
func expandProductIDPattern(vendor, prefix, pattern string) []string {
	var ids []string

	// Handle character class patterns like "7[789]"
	charClassPattern := regexp.MustCompile(`^(\w)\[([^\]]+)\]$`)
	if match := charClassPattern.FindStringSubmatch(pattern); len(match) >= 3 {
		firstChar := match[1]
		chars := match[2]
		for _, c := range chars {
			productID := fmt.Sprintf("0x%s%s%c", prefix, firstChar, c)
			ids = append(ids, fmt.Sprintf("%s:%s", vendor, productID))
		}
		return ids
	}

	// Handle simple hex values like "80"
	if len(pattern) == 2 {
		productID := fmt.Sprintf("0x%s%s", prefix, pattern)
		ids = append(ids, fmt.Sprintf("%s:%s", vendor, productID))
		return ids
	}

	// Handle full 4-digit hex like "0562"
	if len(pattern) == 4 {
		productID := fmt.Sprintf("0x%s", pattern)
		ids = append(ids, fmt.Sprintf("%s:%s", vendor, productID))
		return ids
	}

	// For other complex patterns, skip for now
	return ids
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
