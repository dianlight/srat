import type { Disk, Partition } from "../../store/sratApi";

export function decodeEscapeSequence(source: unknown): string {
	// Basic check to avoid errors if source is not a string
	if (typeof source !== "string") return "";
	return source.replace(/\\x([0-9A-Fa-f]{2})/g, (_match, group1) => {
		// Ensure group1 is treated as a string before parseInt
		return String.fromCharCode(parseInt(String(group1), 16));
	});
}

export function getDiskIdentifier(disk: Disk, fallbackIndex: number): string {
	return (
		disk.id ||
		disk.legacy_device_name ||
		disk.device_path ||
		disk.serial ||
		`disk-${fallbackIndex}`
	);
}

export function getPartitionIdentifier(
	diskIdentifier: string,
	partition: Partition,
	partitionKey: string | undefined,
	fallbackIndex: number,
): string {
	const partitionBase =
		partition.id ||
		partition.uuid ||
		partition.device_path ||
		partition.legacy_device_name ||
		partition.legacy_device_path ||
		partitionKey ||
		`part-${fallbackIndex}`;

	return `${diskIdentifier}::${partitionBase}`;
}

export function getMountpointIdentifier(partitionIdentifier: string, mountpointKey: string): string {
	return `${partitionIdentifier}::mp::${mountpointKey}`;
}
