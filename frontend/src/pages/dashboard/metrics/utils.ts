export function decodeEscapeSequence(source: string) {
	if (typeof source !== "string") return "";
	return source.replace(/\\x([0-9A-Fa-f]{2})/g, (_match, group1) =>
		String.fromCharCode(parseInt(String(group1), 16)),
	);
}

export function formatUptime(millis: number): string {
	let seconds = Math.floor((Date.now() - millis) / 1000);
	if (seconds <= 0) return "0 seconds";

	const days = Math.floor(seconds / (24 * 3600));
	seconds %= 24 * 3600;
	const hours = Math.floor(seconds / 3600);
	seconds %= 3600;
	const minutes = Math.floor(seconds / 60);
	const remainingSeconds = Math.floor(seconds % 60);

	const parts = [];
	if (days > 0) parts.push(`${days}d`);
	if (hours > 0) parts.push(`${hours}h`);
	if (minutes > 0) parts.push(`${minutes}m`);
	if (remainingSeconds > 0 || parts.length === 0)
		parts.push(`${remainingSeconds}s`);

	return parts.join(" ");
}
