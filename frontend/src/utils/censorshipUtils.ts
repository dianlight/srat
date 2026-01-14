/**
 * Utility functions for censoring sensitive data in various formats
 * Uses maskify-ts for advanced data masking and auto-detection
 */

import { Maskify, type AutoMaskOptions } from 'maskify-ts';

// Keywords that indicate sensitive data (used by maskify-ts sensitiveKeys)
export const SENSITIVE_KEYWORDS = [
	'password', 'pass', 'pwd', 'secret', 'token', 'key', 'auth', 'credential',
	'private', 'confidential', 'secure', 'api_key', 'apikey', 'access_token',
	'refresh_token', 'jwt', 'bearer', 'authorization', 'salt', 'hash'
];

/**
 * Check if a field name indicates sensitive data
 */
export function isSensitiveField(label: string): boolean {
	const lowerLabel = label.toLowerCase();
	return SENSITIVE_KEYWORDS.some(keyword =>
		lowerLabel.includes(keyword)
	);
}

/**
 * Censor a value with lock emoji using maskify-ts transform function
 */
export function censorValue(value: any): string {
	const strValue = String(value);
	// Use maskify-ts with custom transform to replace with lock emoji
	return Maskify.mask(strValue, {
		transform: (val: string) => '🔒'.repeat(Math.min(val.length, 8))
	});
}

/**
 * Censor sensitive data in plain text (e.g., INI, config files)
 * Searches for patterns like "key = value" and censors the value if key is sensitive
 * Supports various separators (=, :, ;, >, ->), quoted keys/values, and encoded strings
 */
export function censorPlainText(text: string): string {
	const lines = text.split('\n');
	const censoredLines = lines.map(line => {
		// Enhanced regex to match various key-value patterns:
		// - Supports separators: = : ; > ->
		// - Handles quoted keys/values: "key", 'key', <key>
		// - Handles HTML/URL encoded strings
		// - Preserves indentation and spacing
		// Pattern breakdown:
		// ^(\s*) - capture leading whitespace
		// (['"]?)([^=:;>\s'"]+)\2 - optional quote, key (unquoted or quoted), matching close quote
		// OR <([^>]+)> - key wrapped in angle brackets
		// \s* - optional space before separator
		// (=|:|;|>|->|=>|::) - various separators
		// \s* - optional space after separator
		// (.*) - value (everything remaining on the line)
		const keyValueMatch = line.match(/^(\s*)(?:(['"])([^=:;>\s'"]+)\2|<([^>]+)>|([^=:;>\s'"]+))\s*(=|:|;|>|->|=>|::)\s*(.*)$/);
		
		if (keyValueMatch) {
			const [, indent, openQuote, quotedKey, angleKey, unquotedKey, separator, value] = keyValueMatch;
			
			// Extract the actual key (could be quoted, in angle brackets, or unquoted)
			const key = quotedKey || angleKey || unquotedKey;
			
			// Check if the key is sensitive
			if (key && value !== undefined && isSensitiveField(key)) {
				// Extract value (may be quoted)
				let actualValue = value;
				let valueQuoteOpen = '';
				let valueQuoteClose = '';
				
				// Check if value is quoted
				const valueQuoteMatch = value.match(/^(['"]|&quot;|&apos;|%22|%27)(.*)(\1)$/);
				if (valueQuoteMatch) {
					valueQuoteOpen = valueQuoteMatch[1];
					actualValue = valueQuoteMatch[2];
					valueQuoteClose = valueQuoteMatch[3];
				} else {
					// Check for angle bracket wrapping
					const angleBracketMatch = value.match(/^<(.*)>$/);
					if (angleBracketMatch) {
						valueQuoteOpen = '<';
						actualValue = angleBracketMatch[1];
						valueQuoteClose = '>';
					}
				}
				
				// Preserve the structure but censor the value
				const censoredValue = actualValue.trim() ? censorValue(actualValue) : actualValue;
				
				// Reconstruct the line with original formatting
				const keyPart = openQuote 
					? `${openQuote}${key}${openQuote}` 
					: angleKey 
						? `<${key}>` 
						: key;
				
				return `${indent}${keyPart} ${separator} ${valueQuoteOpen}${censoredValue}${valueQuoteClose}`;
			}
		}
		
		return line;
	});
	
	return censoredLines.join('\n');
}

/**
 * Convert an object to plain text with censorship
 */
export function objectToPlainText(obj: any, indent = 0, label = 'Root'): string {
	if (obj === undefined || obj === null || obj === "") {
		return '';
	}

	const indentStr = '  '.repeat(indent);
	const isSensitive = isSensitiveField(label);
	let result = '';

	if (typeof obj === "string" || typeof obj === "number") {
		const displayValue = isSensitive ? censorValue(obj) : obj;
		result += `${indentStr}${label}: ${displayValue} (${typeof obj}${isSensitive ? ', censored' : ''})\n`;
	} else if (typeof obj === "boolean") {
		result += `${indentStr}${label}: ${obj ? "Yes" : "No"} (boolean)\n`;
	} else if (Array.isArray(obj)) {
		result += `${indentStr}${label} (array[${obj.length}]):\n`;
		obj.forEach((item, index) => {
			result += objectToPlainText(item, indent + 1, `[${index}]`);
		});
	} else if (typeof obj === "object") {
		const keys = Object.getOwnPropertyNames(obj);
		if (label !== 'Root') {
			result += `${indentStr}${label} (object):\n`;
		}
		keys.forEach(key => {
			const value = Object.getOwnPropertyDescriptor(obj, key)?.value;
			result += objectToPlainText(value, indent + (label !== 'Root' ? 1 : 0), key);
		});
	} else {
		result += `${indentStr}${label}: Unknown type (${typeof obj})\n`;
	}

	return result;
}

/**
 * Convert an object to markdown with censorship
 */
export function objectToMarkdown(obj: any, indent = 0, label = 'Root'): string {
	if (obj === undefined || obj === null || obj === "") {
		return '';
	}

	const indentStr = '  '.repeat(indent);
	const isSensitive = isSensitiveField(label);
	let result = '';

	if (typeof obj === "string" || typeof obj === "number") {
		const displayValue = isSensitive ? censorValue(obj) : obj;
		result += `${indentStr}- **${label}**: \`${displayValue}\` _(${typeof obj}${isSensitive ? ', censored' : ''})_\n`;
	} else if (typeof obj === "boolean") {
		result += `${indentStr}- **${label}**: ${obj ? "Yes" : "No"} _(boolean)_\n`;
	} else if (Array.isArray(obj)) {
		result += `${indentStr}- **${label}** _(array[${obj.length}])_:\n`;
		obj.forEach((item, index) => {
			result += objectToMarkdown(item, indent + 1, `[${index}]`);
		});
	} else if (typeof obj === "object") {
		const keys = Object.getOwnPropertyNames(obj);
		if (label !== 'Root') {
			result += `${indentStr}- **${label}** _(object)_:\n`;
		}
		keys.forEach(key => {
			const value = Object.getOwnPropertyDescriptor(obj, key)?.value;
			result += objectToMarkdown(value, indent + (label !== 'Root' ? 1 : 0), key);
		});
	} else {
		result += `${indentStr}- **${label}**: Unknown type (${typeof obj})\n`;
	}

	return result;
}
