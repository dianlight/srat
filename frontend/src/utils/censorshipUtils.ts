/**
 * Utility functions for censoring sensitive data in various formats
 */

// Keywords that indicate sensitive data
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
 * Censor a value with lock emoji
 */
export function censorValue(value: any): string {
	// Use lock emoji to censor sensitive data
	if (typeof value === 'string') {
		return '🔒'.repeat(Math.min(value.length, 8));
	}
	return '🔒'.repeat(8);
}

/**
 * Censor sensitive data in plain text (e.g., INI, config files)
 * Searches for patterns like "key = value" and censors the value if key is sensitive
 */
export function censorPlainText(text: string): string {
	const lines = text.split('\n');
	const censoredLines = lines.map(line => {
		// Match patterns like "key = value" or "key=value" or "key: value"
		const keyValueMatch = line.match(/^(\s*)([^=:\s]+)\s*([=:])\s*(.*)$/);
		
		if (keyValueMatch) {
			const [, indent, key, separator, value] = keyValueMatch;
			
			// Check if the key is sensitive
			if (isSensitiveField(key)) {
				// Preserve the structure but censor the value
				const censoredValue = value.trim() ? censorValue(value) : value;
				return `${indent}${key}${separator} ${censoredValue}`;
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
