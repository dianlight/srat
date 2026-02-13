/**
 * Utility functions for censoring sensitive data in various formats
 * Uses maskify-ts for advanced data masking and auto-detection
 */

import { Maskify } from "maskify-ts";

// Keywords that indicate sensitive data (used by maskify-ts sensitiveKeys)
export const SENSITIVE_KEYWORDS = [
	"password",
	"pass",
	"pwd",
	"secret",
	"token",
	"key",
	"auth",
	"credential",
	"private",
	"confidential",
	"secure",
	"api_key",
	"apikey",
	"access_token",
	"refresh_token",
	"jwt",
	"bearer",
	"authorization",
	"salt",
	"hash",
];

/**
 * Check if a field name indicates sensitive data
 */
export function isSensitiveField(label: string): boolean {
	const lowerLabel = label.toLowerCase();
	return SENSITIVE_KEYWORDS.some((keyword) => lowerLabel.includes(keyword));
}

/**
 * Censor a value with lock emoji using maskify-ts transform function
 */
export function censorValue(value: unknown): string {
	const strValue = String(value);
	// Use maskify-ts with custom transform to replace with lock emoji
	return Maskify.mask(strValue, {
		transform: (val: string) => "🔒".repeat(Math.min(val.length, 8)),
	});
}

/**
 * Censor sensitive data in plain text (e.g., INI, config files, JSON)
 * Searches for patterns like "key = value" and censors the value if key is sensitive
 * Supports various separators (=, :, ;, >, ->), quoted keys/values, escaped quotes, backticks, and encoded strings
 * Also handles JSON key-value pairs like "password":"value"
 */
export function censorPlainText(text: string): string {
	const lines = text.split("\n");
	const censoredLines = lines.map((line) => {
		let result = line;

		// Handle JSON-style key-value pairs:
		// Match quoted keys followed by colon, then handle values in any quote style or unquoted
		// Three patterns:
		// 1. "key":"value" or 'key':'value' - both quoted
		// 2. "key":value - key quoted, value unquoted (can be string, number, bool, null, object, array)
		//
		// We use lookahead to find the proper value boundary without consuming it
		result = result.replace(
			/"([^"]+)"\s*:\s*"([^"]*)"|"([^"]+)"\s*:\s*'([^']*)'|'([^']+)'\s*:\s*"([^"]*)"|'([^']+)'\s*:\s*'([^']*)'|"([^"]+)"\s*:\s*([^,}\]"\s][^,}\]]*?(?:[^,}\]"\s]|(?=\s*[,}\]]))|[^,}\]":\s])/g,
			(match, key1, val1, key2, val2, key3, val3, key4, val4, key5, val5) => {
				const key = key1 || key2 || key3 || key4 || key5;
				const value = val1 || val2 || val3 || val4 || val5;

				if (
					key &&
					isSensitiveField(key) &&
					value !== undefined &&
					value.trim()
				) {
					const censoredVal = censorValue(value.trim());

					// Reconstruct with appropriate quoting
					if (val1 !== undefined) {
						// Case: "key":"value"
						return `"${key}":"${censoredVal}"`;
					} else if (val2 !== undefined) {
						// Case: "key":'value'
						return `"${key}":"${censoredVal}"`;
					} else if (val3 !== undefined) {
						// Case: 'key':"value"
						return `"${key}":"${censoredVal}"`;
					} else if (val4 !== undefined) {
						// Case: 'key':'value'
						return `"${key}":"${censoredVal}"`;
					} else {
						// Case: "key":unquotedvalue
						return `"${key}":"${censoredVal}"`;
					}
				}
				return match;
			},
		);

		// Then, handle INI-style key-value patterns
		// Pattern: key = value or key: value at the beginning of line (after optional whitespace)
		const keyValueMatch = result.match(
			/^(\s*)(?:(?:\\(['"`])([^=:;>\s'"`\\]+)\\\2)|(?:(['"`])([^=:;>\s'"`\\]+)\4)|<([^>]+)>|([^=:;>\s'"`\\]+))\s*(=|:|;|>|->|=>|::)\s*(.*)$/,
		);

		if (keyValueMatch) {
			const [
				,
				indent,
				escapedQuote,
				escapedKey,
				regularQuote,
				regularKey,
				angleKey,
				unquotedKey,
				separator,
				value,
			] = keyValueMatch;

			// Extract the actual key and quote style
			let key: string;
			let keyQuotePrefix = "";
			let keyQuoteSuffix = "";

			if (escapedKey) {
				key = escapedKey;
				keyQuotePrefix = `\\${escapedQuote}`;
				keyQuoteSuffix = `\\${escapedQuote}`;
			} else if (regularKey) {
				key = regularKey;
				keyQuotePrefix = regularQuote || "";
				keyQuoteSuffix = regularQuote || "";
			} else if (angleKey) {
				key = angleKey;
				keyQuotePrefix = "<";
				keyQuoteSuffix = ">";
			} else {
				key = unquotedKey || "";
			}

			// Check if the key is sensitive
			if (key && value !== undefined && isSensitiveField(key)) {
				// Extract value (may be quoted with various quote styles)
				let actualValue = value;
				let valueQuoteOpen = "";
				let valueQuoteClose = "";

				// Check for escaped quotes in value
				const escapedValueMatch = value.match(/^\\(['"`])(.*)\\(\1)$/);
				if (escapedValueMatch) {
					valueQuoteOpen = `\\${escapedValueMatch[1]}`;
					actualValue = escapedValueMatch[2] || "";
					valueQuoteClose = `\\${escapedValueMatch[3]}`;
				} else {
					// Check for regular quotes (including backticks)
					const valueQuoteMatch = value.match(
						/^(['"`]|&quot;|&apos;|%22|%27)(.*)(\1)$/,
					);
					if (valueQuoteMatch) {
						valueQuoteOpen = valueQuoteMatch[1] || "";
						actualValue = valueQuoteMatch[2] || "";
						valueQuoteClose = valueQuoteMatch[3] || "";
					} else {
						// Check for angle bracket wrapping
						const angleBracketMatch = value.match(/^<(.*)>$/);
						if (angleBracketMatch) {
							valueQuoteOpen = "<";
							actualValue = angleBracketMatch[1] || "";
							valueQuoteClose = ">";
						}
					}
				}

				// Preserve the structure but censor the value
				const censoredValue = actualValue.trim()
					? censorValue(actualValue)
					: actualValue;

				// Reconstruct the line with original formatting
				const keyPart = keyQuotePrefix + key + keyQuoteSuffix;

				return `${indent}${keyPart} ${separator} ${valueQuoteOpen}${censoredValue}${valueQuoteClose}`;
			}
		}

		return result;
	});

	return censoredLines.join("\n");
}

/**
 * Convert an object to plain text with censorship
 */
export function objectToPlainText(
	obj: unknown,
	indent = 0,
	label = "Root",
): string {
	if (obj === undefined || obj === null || obj === "") {
		return "";
	}

	const indentStr = "  ".repeat(indent);
	const isSensitive = isSensitiveField(label);
	let result = "";

	if (typeof obj === "string" || typeof obj === "number") {
		const displayValue = isSensitive ? censorValue(obj) : obj;
		result += `${indentStr}${label}: ${displayValue} (${typeof obj}${isSensitive ? ", censored" : ""})\n`;
	} else if (typeof obj === "boolean") {
		result += `${indentStr}${label}: ${obj ? "Yes" : "No"} (boolean)\n`;
	} else if (Array.isArray(obj)) {
		result += `${indentStr}${label} (array[${obj.length}]):\n`;
		obj.forEach((item, index) => {
			result += objectToPlainText(item, indent + 1, `[${index}]`);
		});
	} else if (typeof obj === "object") {
		const keys = Object.getOwnPropertyNames(obj);
		if (label !== "Root") {
			result += `${indentStr}${label} (object):\n`;
		}
		keys.forEach((key) => {
			const value = Object.getOwnPropertyDescriptor(obj, key)?.value;
			result += objectToPlainText(
				value,
				indent + (label !== "Root" ? 1 : 0),
				key,
			);
		});
	} else {
		result += `${indentStr}${label}: Unknown type (${typeof obj})\n`;
	}

	return result;
}

/**
 * Convert an object to markdown with censorship
 */
export function objectToMarkdown(
	obj: unknown,
	indent = 0,
	label = "Root",
): string {
	if (obj === undefined || obj === null || obj === "") {
		return "";
	}

	const indentStr = "  ".repeat(indent);
	const isSensitive = isSensitiveField(label);
	let result = "";

	if (typeof obj === "string" || typeof obj === "number") {
		const displayValue = isSensitive ? censorValue(obj) : obj;
		result += `${indentStr}- **${label}**: \`${displayValue}\` _(${typeof obj}${isSensitive ? ", censored" : ""})_\n`;
	} else if (typeof obj === "boolean") {
		result += `${indentStr}- **${label}**: ${obj ? "Yes" : "No"} _(boolean)_\n`;
	} else if (Array.isArray(obj)) {
		result += `${indentStr}- **${label}** _(array[${obj.length}])_:\n`;
		obj.forEach((item, index) => {
			result += objectToMarkdown(item, indent + 1, `[${index}]`);
		});
	} else if (typeof obj === "object") {
		const keys = Object.getOwnPropertyNames(obj);
		if (label !== "Root") {
			result += `${indentStr}- **${label}** _(object)_:\n`;
		}
		keys.forEach((key) => {
			const value = Object.getOwnPropertyDescriptor(obj, key)?.value;
			result += objectToMarkdown(
				value,
				indent + (label !== "Root" ? 1 : 0),
				key,
			);
		});
	} else {
		result += `${indentStr}- **${label}**: Unknown type (${typeof obj})\n`;
	}

	return result;
}
