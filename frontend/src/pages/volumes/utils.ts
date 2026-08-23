import type { Disk, Partition } from "../../store/sratApi";

export function decodeEscapeSequence(source: unknown): string {
  // Basic check to avoid errors if source is not a string
  if (typeof source !== "string") return "";
  return source.replace(/\\x([0-9A-Fa-f]{2})/g, (_match, group1) => {
    // Ensure group1 is treated as a string before parseInt
    return String.fromCharCode(parseInt(String(group1), 16));
  });
}

export function getFilesystemLabelValidation(
  label: string,
  labelRule: unknown,
  optional = false,
): {
  isValid: boolean;
  helperText?: string;
} {
  const normalizedRule = typeof labelRule === "string" ? labelRule.trim() : "";
  const normalizedLabel = label.trim();

  if (!normalizedRule) {
    return {
      isValid: optional || normalizedLabel.length > 0,
    };
  }

  const acceptedFormatHint = `Accepted format: ${normalizedRule}`;

  if (normalizedLabel.length === 0) {
    return {
      isValid: optional,
      helperText: acceptedFormatHint,
    };
  }

  try {
    const isValid = new RegExp(normalizedRule).test(normalizedLabel);
    return {
      isValid,
      helperText: isValid
        ? acceptedFormatHint
        : `Invalid label. ${acceptedFormatHint}`,
    };
  } catch {
    return {
      isValid: true,
      helperText: acceptedFormatHint,
    };
  }
}

export function getDiskIdentifier(disk: Disk, fallbackIndex: number): string {
  const stableId =
    disk.id || disk.legacy_device_name || disk.device_path || disk.serial;
  if (stableId) return stableId;
  // Backend defect indicator: a disk with none of id/legacy_device_name/
  // device_path/serial would re-point persisted selection/expansion state
  // whenever the snapshot order changes, so make the fallback visible.
  console.warn(
    `Disk #${fallbackIndex} has no stable identifier (id/legacy_device_name/device_path/serial); using index-based key "disk-${fallbackIndex}" which is not stable across reordering.`,
  );
  return `disk-${fallbackIndex}`;
}

export function getPartitionIdentifier(
  diskIdentifier: string,
  partition: Partition,
  partitionKey: string | undefined,
  fallbackIndex: number,
): string {
  const stableBase =
    partition.id ||
    partition.uuid ||
    partition.device_path ||
    partition.legacy_device_name ||
    partition.legacy_device_path ||
    partitionKey;
  if (stableBase) return `${diskIdentifier}::${stableBase}`;
  // Same backend-defect indicator as getDiskIdentifier: warn when only the
  // index-based fallback is available.
  console.warn(
    `Partition #${fallbackIndex} has no stable identifier (id/uuid/device_path/legacy names/key); using index-based key "part-${fallbackIndex}" which is not stable across reordering.`,
  );
  return `${diskIdentifier}::part-${fallbackIndex}`;
}

export function getMountpointIdentifier(
  partitionIdentifier: string,
  mountpointKey: string,
): string {
  return `${partitionIdentifier}::mp::${mountpointKey}`;
}

// Task 044: the backend synthesizes a whole-disk partition entry for raw disks
// (no partition table) so they stay actionable (mount/unmount/format). The
// backend may legitimately detect a filesystem written directly to the whole
// disk (e.g. vfat superfloppy). Such an entry always carries the disk's own
// legacy device name (real partitions have a numeric suffix, like sdc1), so
// name equality is the discriminator. It is not a real partition and must not
// be counted or labeled as one.
export function isSynthesizedWholeDiskPartition(
  disk: Disk,
  partition: Partition,
): boolean {
  return (
    partition.legacy_device_name != null &&
    partition.legacy_device_name === disk.legacy_device_name
  );
}

export function getRealPartitions(disk: Disk): Partition[] {
  return Object.values(disk.partitions || {}).filter(
    (partition) => !isSynthesizedWholeDiskPartition(disk, partition),
  );
}
