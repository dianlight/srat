import { describe, expect, it } from "vitest";
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
		expect(sanitizeAndUppercaseShareName("500G")).toBe("500G");
		expect(sanitizeAndUppercaseShareName("1TB_Backup")).toBe("1TB_BACKUP");
		expect(sanitizeAndUppercaseShareName("")).toBe("");
		// Hyphens are valid in share names (issue #1400) and must be preserved
		expect(sanitizeAndUppercaseShareName("parity-check-1")).toBe("PARITY-CHECK-1");
		expect(sanitizeAndUppercaseShareName("my-disk")).toBe("MY-DISK");
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

	it("normalizes share subfolders (issue #184)", async () => {
		const { normalizeSubfolder } = await import("../utils");

		expect(normalizeSubfolder(undefined)).toBe("");
		expect(normalizeSubfolder(null)).toBe("");
		expect(normalizeSubfolder("photos/2024/")).toBe("photos/2024");
		// Names are never trimmed: a space can be part of a folder name.
		expect(normalizeSubfolder("Italie 2025")).toBe("Italie 2025");
	});

	it("explains unacceptable share subfolders like the backend", async () => {
		const { subfolderProblem } = await import("../utils");

		for (const ok of ["", "emperor", "photos/2024/", "..hidden/v1.2", "Rodina/Dovolená", "Italie 2025", "a".repeat(255)]) {
			expect(subfolderProblem(ok)).toBeUndefined();
		}
		expect(subfolderProblem("/etc")).toMatch(/leading \//);
		expect(subfolderProblem("/")).toMatch(/leading \//);
		expect(subfolderProblem("photos\\2024")).toMatch(/separate/);
		expect(subfolderProblem("photos//2024")).toMatch(/empty/);
		expect(subfolderProblem("../etc")).toMatch(/"\.\."/);
		expect(subfolderProblem("./photos")).toMatch(/"\."/);
		expect(subfolderProblem("a".repeat(256))).toMatch(/255 bytes/);
		// 128 two-byte characters are 256 bytes, over the limit like on the server.
		expect(subfolderProblem("é".repeat(128))).toMatch(/255 bytes/);
		expect(subfolderProblem("pho\u0007tos")).toMatch(/control/);
		expect(subfolderProblem(" photos")).toMatch(/space/);
		expect(subfolderProblem("photos /2024")).toMatch(/space/);
		expect(subfolderProblem("   ")).toMatch(/space/);
	});

	it("lists the folders of a volume already used by other shares", async () => {
		const { foldersInUse } = await import("../utils");
		const shares = [
			{ name: "DATA1", mount_point_data: { path: "/mnt/DATA1" } },
			{ name: "EMPEROR", subfolder: "emperor/", mount_point_data: { path: "/mnt/DATA1" } },
			{ name: "OTHER", subfolder: "x", mount_point_data: { path: "/mnt/OTHER" } },
		];

		expect(foldersInUse(shares, "/mnt/DATA1")).toEqual(new Set(["", "emperor"]));
		expect(foldersInUse(shares, "/mnt/DATA1", "EMPEROR")).toEqual(new Set([""]));
		expect(foldersInUse(shares, "/mnt/NONE")).toEqual(new Set());
	});
});
