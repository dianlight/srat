import { sha1 } from "js-sha1";

export function decodeEscapeSequence(source: string) {
	// Basic check to avoid errors if source is not a string
	if (typeof source !== "string") return "";
	return source.replace(/\\x([0-9A-Fa-f]{2})/g, (_match, group1) => {
		// Ensure group1 is treated as a string before parseInt
		return String.fromCharCode(parseInt(String(group1), 16));
	});
}

// Helper function to generate SHA-1 hash with fallback
export async function generateSHA1Hash(input: string): Promise<string> {
	// Try to use crypto.subtle if available
	if (typeof crypto !== "undefined" && crypto.subtle) {
		try {
			const hashBuffer = await crypto.subtle.digest(
				"SHA-1",
				new TextEncoder().encode(input),
			);
			return Array.from(new Uint8Array(hashBuffer))
				.map((b) => b.toString(16).padStart(2, "0"))
				.join("");
		} catch (error) {
			console.warn("crypto.subtle failed, falling back to js-sha1:", error);
		}
	}

	// Fallback to js-sha1
	return sha1(input);
}
