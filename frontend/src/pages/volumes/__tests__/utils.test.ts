import { describe, expect, it } from "vitest";

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

	it("getFilesystemLabelValidation validates labels against the provided regex", async () => {
		const { getFilesystemLabelValidation } = await import("../utils");

		const invalidResult = getFilesystemLabelValidation(
			"bad-label!",
			"^[A-Z0-9]{1,5}$",
		);
		expect(invalidResult.isValid).toBe(false);
		expect(invalidResult.helperText).toContain(
			"Accepted format: ^[A-Z0-9]{1,5}$",
		);

		const validResult = getFilesystemLabelValidation(
			"DATA",
			"^[A-Z0-9]{1,5}$",
		);
		expect(validResult.isValid).toBe(true);
		expect(validResult.helperText).toContain(
			"Accepted format: ^[A-Z0-9]{1,5}$",
		);
	});

	it("getFilesystemLabelValidation allows an empty optional label", async () => {
		const { getFilesystemLabelValidation } = await import("../utils");

		const result = getFilesystemLabelValidation(
			"",
			"^[A-Z0-9]{1,5}$",
			true,
		);
		expect(result.isValid).toBe(true);
		expect(result.helperText).toContain(
			"Accepted format: ^[A-Z0-9]{1,5}$",
		);
	});

	it("isSynthesizedWholeDiskPartition matches a whole-disk entry with a detected filesystem", async () => {
		const { isSynthesizedWholeDiskPartition } = await import("../utils");

		const disk = { legacy_device_name: "sdd" } as any;
		const wholeDiskWithFs = {
			legacy_device_name: "sdd",
			fs_type: "vfat",
		} as any;
		expect(isSynthesizedWholeDiskPartition(disk, wholeDiskWithFs)).toBe(true);
	});

	it("isSynthesizedWholeDiskPartition does not match a real partition with a numeric suffix", async () => {
		const { isSynthesizedWholeDiskPartition } = await import("../utils");

		const disk = { legacy_device_name: "sdd" } as any;
		const realPartition = {
			legacy_device_name: "sdd1",
			fs_type: "vfat",
		} as any;
		expect(isSynthesizedWholeDiskPartition(disk, realPartition)).toBe(false);
	});

	it("getRealPartitions excludes whole-disk entries even when a filesystem is detected", async () => {
		const { getRealPartitions } = await import("../utils");

		const disk = {
			legacy_device_name: "sdd",
			partitions: {
				"usb-General_UDisk-0:0": {
					id: "usb-General_UDisk-0:0",
					legacy_device_name: "sdd",
					fs_type: "vfat",
				},
				"usb-General_UDisk-0:1": {
					id: "usb-General_UDisk-0:1",
					legacy_device_name: "sdd1",
					fs_type: "ext4",
				},
			},
		} as any;

		const real = getRealPartitions(disk);
		expect(real).toHaveLength(1);
		expect(real[0].legacy_device_name).toBe("sdd1");
	});
});

