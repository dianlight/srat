export function decodeEscapeSequence(source: unknown): string {
	if (typeof source !== "string") return "";
	return source.replace(/\\x([0-9A-Fa-f]{2})/g, (_match, group1) =>
		String.fromCharCode(parseInt(String(group1), 16)),
	);
}
