import "../../../../../test/setup";
import { describe, expect, it } from "bun:test";

describe("metrics utils", () => {
	it("decodes escape sequences correctly", async () => {
		const { decodeEscapeSequence } = await import("../utils");

		expect(decodeEscapeSequence("\\x48\\x65\\x6c\\x6c\\x6f")).toBe("Hello");
		expect(decodeEscapeSequence("\\x57\\x6f\\x72\\x6c\\x64")).toBe("World");
	});

	it("handles strings without escape sequences", async () => {
		const { decodeEscapeSequence } = await import("../utils");

		expect(decodeEscapeSequence("Plain text")).toBe("Plain text");
		expect(decodeEscapeSequence("Hello World")).toBe("Hello World");
	});

	it("handles empty strings", async () => {
		const { decodeEscapeSequence } = await import("../utils");

		expect(decodeEscapeSequence("")).toBe("");
	});

	it("handles mixed content with and without escapes", async () => {
		const { decodeEscapeSequence } = await import("../utils");

		expect(decodeEscapeSequence("Hello \\x57\\x6f\\x72\\x6c\\x64!")).toBe(
			"Hello World!",
		);
		expect(decodeEscapeSequence("\\x48i there")).toBe("Hi there");
	});

	it("handles non-string inputs", async () => {
		const { decodeEscapeSequence } = await import("../utils");

		expect(decodeEscapeSequence(null as any)).toBe("");
		expect(decodeEscapeSequence(undefined as any)).toBe("");
		expect(decodeEscapeSequence(123 as any)).toBe("");
	});

	it("handles invalid escape sequences", async () => {
		const { decodeEscapeSequence } = await import("../utils");

		// Invalid sequences should be left as-is
		expect(decodeEscapeSequence("\\xGG")).toBe("\\xGG");
		expect(decodeEscapeSequence("\\x")).toBe("\\x");
	});

	it("handles multiple escape sequences in a row", async () => {
		const { decodeEscapeSequence } = await import("../utils");

		expect(decodeEscapeSequence("\\x41\\x42\\x43")).toBe("ABC");
	});

	it("handles special characters", async () => {
		const { decodeEscapeSequence } = await import("../utils");

		expect(decodeEscapeSequence("\\x20")).toBe(" "); // space
		expect(decodeEscapeSequence("\\x0a")).toBe("\n"); // newline
		expect(decodeEscapeSequence("\\x09")).toBe("\t"); // tab
	});

	it("preserves case sensitivity in hex values", async () => {
		const { decodeEscapeSequence } = await import("../utils");

		expect(decodeEscapeSequence("\\x41")).toBe("A");
		expect(decodeEscapeSequence("\\x61")).toBe("a");
	});

	it("handles both uppercase and lowercase hex notation", async () => {
		const { decodeEscapeSequence } = await import("../utils");

		expect(decodeEscapeSequence("\\x41")).toBe("A");
		expect(decodeEscapeSequence("\\X41")).toBe("\\X41"); // uppercase X is not valid
		expect(decodeEscapeSequence("\\x4A")).toBe("J");
		expect(decodeEscapeSequence("\\x4a")).toBe("J");
	});
});
