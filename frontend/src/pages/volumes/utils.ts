export function decodeEscapeSequence(source: unknown): string {
	// Basic check to avoid errors if source is not a string
	if (typeof source !== "string") return "";
	return source.replace(/\\x([0-9A-Fa-f]{2})/g, (_match, group1) => {
		// Ensure group1 is treated as a string before parseInt
		return String.fromCharCode(parseInt(String(group1), 16));
	});
}
