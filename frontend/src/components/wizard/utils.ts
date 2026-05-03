import type { Disk, Partition, Settings, User } from "../../store/sratApi";

export interface WizardPartitionOption {
  partitionId: string;
  displayName: string;
  suggestedShareName: string;
  partition: Partition;
}

export const isValidSettings = (data: unknown): data is Settings =>
  data !== null && typeof data === "object" && "hostname" in data;

export const isValidUsers = (data: unknown): data is User[] =>
  Array.isArray(data) &&
  data.every((u) => typeof u === "object" && u !== null && "username" in u);

export const sanitizeWizardShareName = (value: string): string => {
  const sanitized = value
    .trim()
    .replace(/^[^a-zA-Z0-9]+/, "")
    .replace(/[^a-zA-Z0-9]+/g, "_")
    .replace(/_+/g, "_");
  return sanitized.length > 0 ? sanitized : "Share";
};

export const getWizardAvailablePartitions = (
  disks?: Disk[] | null,
): WizardPartitionOption[] => {
  if (!Array.isArray(disks) || disks.length === 0) {
    return [];
  }

  const options: WizardPartitionOption[] = [];

  for (const disk of disks) {
    const diskLabel =
      disk.model ||
      disk.serial ||
      disk.legacy_device_name ||
      disk.device_path ||
      "Disk";

    for (const partition of Object.values(disk.partitions || {})) {
      if (!partition?.id || partition.system) {
        continue;
      }

      const mountEntries = Object.values(partition.mount_point_data || {});
      const isAlreadyMounted = mountEntries.some(
        (entry) => entry?.is_mounted || Boolean(entry?.path),
      );
      if (isAlreadyMounted) {
        continue;
      }

      const baseName =
        partition.name ||
        partition.legacy_device_name ||
        partition.legacy_device_path ||
        partition.device_path ||
        partition.id;

      options.push({
        partitionId: partition.id,
        displayName: `${baseName} (${diskLabel})`,
        suggestedShareName: sanitizeWizardShareName(baseName),
        partition,
      });
    }
  }

  return options;
};

export const findPartitionById = (
  disks: Disk[] | null | undefined,
  partitionId: string,
): Partition | undefined => {
  if (!Array.isArray(disks) || !partitionId) {
    return undefined;
  }

  for (const disk of disks) {
    for (const partition of Object.values(disk.partitions || {})) {
      if (partition?.id === partitionId) {
        return partition;
      }
    }
  }

  return undefined;
};
