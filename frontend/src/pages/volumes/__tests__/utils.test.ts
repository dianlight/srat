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










});
