import type { Disk, Partition } from "../../store/sratApi";

export type TourVolumeSelection = {
  disk: Disk;
  partition?: Partition;
};

export function getTourVolumeSelection(
  disks?: Disk[],
  hideSystemPartitions = false,
): TourVolumeSelection | undefined {
  if (!Array.isArray(disks) || disks.length === 0) return undefined;

  for (const disk of disks) {
    const partitions = Object.values(disk.partitions || {});
    const visiblePartition = partitions.find((partition) => {
      if (!partition) return false;
      if (!hideSystemPartitions) return true;

      return !(
        partition.name?.startsWith("hassos-") || partition.system === true
      );
    });

    if (visiblePartition) {
      return { disk, partition: visiblePartition };
    }
  }

  return { disk: disks[0] as Disk };
}
