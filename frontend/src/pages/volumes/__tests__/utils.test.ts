import "../../../../test/setup";
import { describe, expect, it } from "bun:test";

describe("volumes utils", () => {
	it("decodeEscapeSequence decodes hex sequences", async () => {
		const { decodeEscapeSequence } = await import("../utils");

		// Test hex sequence
		const result = decodeEscapeSequence("test\\x20value");
		expect(result).toBe("test value"); // \x20 is space
	});

	it("decodeEscapeSequence handles regular strings", async () => {
		const { decodeEscapeSequence } = await import("../utils");

		const result = decodeEscapeSequence("regular_string");
		expect(result).toBe("regular_string");
	});

	it("decodeEscapeSequence handles empty string", async () => {
		const { decodeEscapeSequence } = await import("../utils");

		const result = decodeEscapeSequence("");
		expect(result).toBe("");
	});

	it("decodeEscapeSequence handles multiple escape sequences", async () => {
		const { decodeEscapeSequence } = await import("../utils");

		const result = decodeEscapeSequence("test\\x20value\\x20here");
		expect(result).toContain("test");
		expect(result).toContain("value");
	});

	it("decodeEscapeSequence handles non-string input", async () => {
		const { decodeEscapeSequence } = await import("../utils");

		const result = decodeEscapeSequence(null);
		expect(result).toBe("");
	});

	it("decodeEscapeSequence handles hex escape with uppercase", async () => {
		const { decodeEscapeSequence } = await import("../utils");

		const result = decodeEscapeSequence("test\\x41"); // A
		expect(result).toBe("testA");
	});

	it("generateSHA1Hash generates hash from string", async () => {
		const { generateSHA1Hash } = await import("../utils");

		const hash = await generateSHA1Hash("test-string");
		expect(hash).toBeTruthy();
		expect(typeof hash).toBe("string");
		expect(hash.length).toBeGreaterThan(0);
	});

	it("generateSHA1Hash generates consistent hashes", async () => {
		const { generateSHA1Hash } = await import("../utils");

		const hash1 = await generateSHA1Hash("test");
		const hash2 = await generateSHA1Hash("test");
		expect(hash1).toBe(hash2);
	});

	it("generateSHA1Hash generates different hashes for different inputs", async () => {
		const { generateSHA1Hash } = await import("../utils");

		const hash1 = await generateSHA1Hash("test1");
		const hash2 = await generateSHA1Hash("test2");
		expect(hash1).not.toBe(hash2);
	});

	it("generateSHA1Hash handles empty string", async () => {
		const { generateSHA1Hash } = await import("../utils");

		const hash = await generateSHA1Hash("");
		expect(hash).toBeTruthy();
		expect(typeof hash).toBe("string");
	});

	it("generateSHA1Hash handles special characters", async () => {
		const { generateSHA1Hash } = await import("../utils");

		const hash = await generateSHA1Hash("test@#$%^&*()");
		expect(hash).toBeTruthy();
		expect(typeof hash).toBe("string");
	});

	it("generateSHA1Hash handles unicode characters", async () => {
		const { generateSHA1Hash } = await import("../utils");

		const hash = await generateSHA1Hash("test-ñ-unicode-测试");
		expect(hash).toBeTruthy();
		expect(typeof hash).toBe("string");
	});

	it("generateSHA1Hash produces valid SHA1 format", async () => {
		const { generateSHA1Hash } = await import("../utils");

		const hash = await generateSHA1Hash("test");
		// SHA1 hash should be 40 characters long (160 bits in hex)
		expect(hash.length).toBe(40);
		// Should only contain hex characters
		expect(/^[a-f0-9]+$/.test(hash)).toBe(true);
	});
});
