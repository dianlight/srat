import "../../../../test/setup";
import { describe, expect, it } from "bun:test";
import type { CasingStyle } from "../types";

describe("shares utils", () => {
	it("extracts base name from path variations", async () => {
		const { getPathBaseName } = await import("../utils");

		expect(getPathBaseName("/mnt/data/share")).toBe("share");
		expect(getPathBaseName("/mnt/data/share/")).toBe("share");
		expect(getPathBaseName("/")).toBe("");
		expect(getPathBaseName("")).toBe("");
	});

	it("sanitizes share names and enforces uppercase", async () => {
		const { sanitizeAndUppercaseShareName } = await import("../utils");

		expect(sanitizeAndUppercaseShareName("My Share")).toBe("MY_SHARE");
		expect(sanitizeAndUppercaseShareName("home/media")).toBe("HOME_MEDIA");
		expect(sanitizeAndUppercaseShareName("")).toBe("");
	});

	it("validates veto file entries", async () => {
		const { isValidVetoFileEntry } = await import("../utils");

		expect(isValidVetoFileEntry("Thumbs.db")).toBe(true);
		expect(isValidVetoFileEntry("foo/bar")).toBe(false);
		expect(isValidVetoFileEntry("bad\0entry")).toBe(false);
	});

	it("converts casing styles using helpers", async () => {
		const { toCamelCase, toKebabCase } = await import("../utils");

		expect(toCamelCase("My new Share")).toBe("myNewShare");
		expect(toCamelCase("MY_NEW_SHARE")).toBe("myNewShare");
		expect(toCamelCase("")).toBe("");

		expect(toKebabCase("My new Share")).toBe("my_new_share");
		expect(toKebabCase("already_kebab")).toBe("already_kebab");
		expect(toKebabCase("")).toBe("");
	});

	it("returns icons for casing styles", async () => {
		const { getCasingIcon, casingCycleOrder } = await import("../utils");

		for (const style of casingCycleOrder) {
			const Icon = getCasingIcon(style);
			expect(typeof Icon === "function" || typeof Icon === "object").toBe(true);
		}

		// Test fallback for unknown style
		const fallbackIcon = getCasingIcon("unknown" as CasingStyle);
		expect(
			typeof fallbackIcon === "function" || typeof fallbackIcon === "object",
		).toBe(true);
	});
});
