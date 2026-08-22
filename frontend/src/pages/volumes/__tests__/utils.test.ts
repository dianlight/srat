import { afterEach, describe, expect, it, vi } from "vitest";

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

	afterEach(() => {
		vi.restoreAllMocks();
	});

	it("getDiskIdentifier prefers stable ids over the fallback index", async () => {
		const { getDiskIdentifier } = await import("../utils");

		const disk = {
			id: "sda-1234",
			legacy_device_name: "sda",
			device_path: "/dev/sda",
			serial: "SERIAL",
		} as any;
		expect(getDiskIdentifier(disk, 0)).toBe("sda-1234");
	});

	it("getDiskIdentifier falls back through legacy name, device path, serial", async () => {
		const { getDiskIdentifier } = await import("../utils");

		expect(
			getDiskIdentifier({ legacy_device_name: "sdb" } as any, 0),
		).toBe("sdb");
		expect(
			getDiskIdentifier({ device_path: "/dev/sdc" } as any, 0),
		).toBe("/dev/sdc");
		expect(getDiskIdentifier({ serial: "SN-X" } as any, 0)).toBe("SN-X");
	});

	it("getDiskIdentifier warns and uses the index key when no stable id exists", async () => {
		const { getDiskIdentifier } = await import("../utils");
		const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

		expect(getDiskIdentifier({} as any, 3)).toBe("disk-3");
		expect(warnSpy).toHaveBeenCalledTimes(1);
		expect(warnSpy.mock.calls[0][0]).toContain('"disk-3"');
	});

	it("getPartitionIdentifier prefers stable ids and the map key", async () => {
		const { getPartitionIdentifier } = await import("../utils");

		const partition = {
			id: "part-abc",
			uuid: "uuid-1",
			device_path: "/dev/sda1",
		} as any;
		expect(getPartitionIdentifier("disk-1", partition, "key1", 0)).toBe(
			"disk-1::part-abc",
		);
		// Falls back to the object key before the index.
		expect(
			getPartitionIdentifier("disk-1", {} as any, "key1", 0),
		).toBe("disk-1::key1");
	});

	it("getPartitionIdentifier warns and uses the index key when nothing stable exists", async () => {
		const { getPartitionIdentifier } = await import("../utils");
		const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

		expect(getPartitionIdentifier("disk-1", {} as any, undefined, 2)).toBe(
			"disk-1::part-2",
		);
		expect(warnSpy).toHaveBeenCalledTimes(1);
		expect(warnSpy.mock.calls[0][0]).toContain('"part-2"');
	});

	it("identifiers stay stable across snapshot reordering", async () => {
		const { getDiskIdentifier, getPartitionIdentifier } = await import(
			"../utils"
		);

		const disks = [
			{
				id: "disk-a",
				legacy_device_name: "sda",
				partitions: {
					p1: { id: "part-1", legacy_device_name: "sda1" },
					p2: { id: "part-2", legacy_device_name: "sda2" },
				},
			},
			{
				id: "disk-b",
				legacy_device_name: "sdb",
				partitions: {
					p3: { id: "part-3", legacy_device_name: "sdb1" },
				},
			},
		] as any;

		const before = disks.map((disk, diskIndex) => ({
			diskId: getDiskIdentifier(disk, diskIndex),
			partitionIds: Object.entries(disk.partitions).map(
				([key, partition], partIndex) =>
					getPartitionIdentifier(
						getDiskIdentifier(disk, diskIndex),
						partition as any,
						key,
						partIndex,
					),
			),
		}));

		// Reverse the snapshot order — identifiers must not change because
		// they are derived from stable ids, not array positions.
		const reordered = [...disks].reverse();
		const after = reordered.map((disk, diskIndex) => ({
			diskId: getDiskIdentifier(disk, diskIndex),
			partitionIds: Object.entries(disk.partitions).map(
				([key, partition], partIndex) =>
					getPartitionIdentifier(
						getDiskIdentifier(disk, diskIndex),
						partition as any,
						key,
						partIndex,
					),
			),
		}));

		// Key by disk id: each disk must keep its identifiers no matter where
		// it appears in the snapshot.
		const idsByDisk = (list: typeof before) =>
			Object.fromEntries(list.map((entry) => [entry.diskId, entry.partitionIds]));
		expect(idsByDisk(after)).toEqual(idsByDisk(before));
	});
});

