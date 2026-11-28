/* eslint-disable */
import type { MountPointData, Partition } from "../../../store/sratApi";

export type PartitionActionKey =
  | "mount"
  | "enable-automount"
  | "disable-automount"
  | "unmount"
  | "force-unmount"
  | "create-share"
  | "go-to-share"
  | "check-filesystem"
  | "set-label"
  | "format";

export interface PartitionActionItem {
  key: PartitionActionKey;
  title: string;
  color:
    | "primary"
    | "secondary"
    | "warning"
    | "error"
    | "info"
    | "success"
    | undefined;
  onClick: () => void;
}

export interface PartitionActionOptions {
  partition: Partition;
  protectedMode: boolean;
  // When set, actions are scoped to one mount point of a multi-mount
  // partition (tree leaf / details-panel mount-point row).
  mountPointDataSelection?: MountPointData;
  onToggleAutomount: (
    partition: Partition,
    mountPointData?: MountPointData,
  ) => void;
  onMount: (partition: Partition, mountPointData?: MountPointData) => void;
  onUnmount: (
    partition: Partition,
    force: boolean,
    mountPointData?: MountPointData,
  ) => void;
  onCreateShare: (
    partition: Partition,
    mountPointData?: MountPointData,
  ) => void;
  onGoToShare: (partition: Partition, mountPointData?: MountPointData) => void;
  onCheckFilesystem?: (partition: Partition) => void;
  onSetFilesystemLabel?: (partition: Partition) => void;
  onFormatPartition?: (partition: Partition) => void;
}

export function getPartitionActionItems({
  partition,
  protectedMode,
  mountPointDataSelection,
  onToggleAutomount,
  onMount,
  onUnmount,
  onCreateShare,
  onGoToShare,
  onCheckFilesystem,
  onSetFilesystemLabel,
  onFormatPartition,
}: PartitionActionOptions): PartitionActionItem[] | null {
  // #1073: stale orphan rows (invalid + unmounted, e.g. a legacy dashes path
  // left behind after the wizard suggestion switched to underscores) must not
  // count toward the multi-mount guard nor shadow the live entry below. Only
  // non-orphan entries gate visibility; the primary entry prefers mounted.
  const mountEntries = Object.values(partition.mount_point_data || {});
  const liveEntries = mountEntries.filter(
    (entry) =>
      !(
        (entry as MountPointData)?.invalid &&
        !(entry as MountPointData)?.is_mounted
      ),
  );

  // A tree leaf or a details-panel mount-point row passes its own entry so
  // actions stay scoped to it. Without a selection, keep the legacy behavior
  // of hiding partition-level actions for ambiguous multi-mount partitions.
  const selectedMpd = mountPointDataSelection as MountPointData | undefined;
  const hasHAMountedEntry = liveEntries.some(
    (entry) => (entry as MountPointData)?.is_ha_mounted,
  );
  const scopedAddonMpd =
    selectedMpd && hasHAMountedEntry && !selectedMpd.is_ha_mounted
      ? selectedMpd
      : undefined;

  // Home Assistant owns the mount lifecycle for its HA-mounted entries
  // (2026.11+ storage): no mount/unmount/automount/format/label actions there.
  // A share on the volume can still be created or opened from here.
  //
  // This branch MUST run before the host_mount_point_data guard below: an
  // HA-managed disk always surfaces its host-namespace mount under
  // host_mount_point_data (e.g. /mnt/data/supervisor/media/<name>), and that
  // guard would otherwise return null first and hide the only supported
  // action (Create Share / Go to Share). When an addon entry is explicitly
  // selected on a mixed-ownership partition, it keeps its own actions.
  const haMpd = (
    selectedMpd
      ? selectedMpd.is_ha_mounted
        ? selectedMpd
        : undefined
      : liveEntries.find((entry) => (entry as MountPointData)?.is_ha_mounted)
  ) as MountPointData | undefined;

  if (protectedMode || partition.name?.startsWith("hassos-")) {
    return null;
  }

  if (haMpd) {
    if (haMpd.share) {
      return [
        {
          key: "go-to-share",
          title: "Go to Share",
          color: undefined,
          onClick: () => onGoToShare(partition, haMpd),
        },
      ];
    }
    // Sharing an HA-mounted disk is the primary supported use case.
    return [
      {
        key: "create-share",
        title: "Create Share",
        color: "success",
        onClick: () => onCreateShare(partition, haMpd),
      },
    ];
  }

  const hasHostMount =
    Object.values(partition.host_mount_point_data || {}).length > 0;
  if (
    (hasHostMount && !scopedAddonMpd) ||
    (liveEntries.length > 1 && !selectedMpd)
  ) {
    return null;
  }

  const actionItems: PartitionActionItem[] = [];
  const mountPointData = partition.mount_point_data || {};
  const keys = Object.keys(mountPointData);

  const liveMpd = (liveEntries[0] ?? mountEntries[0]) as MountPointData;
  // Fall back to the legacy first-key lookup when the map is non-empty but
  // the live pick is missing (keeps array-shaped test fixtures working).
  const mpd =
    selectedMpd ?? liveMpd ?? (mountPointData[keys[0]] as MountPointData);
  const isMounted = mpd?.is_mounted;
  const hasEnabledShare = mpd?.share && mpd?.share.disabled === false;
  const hasShare = mpd?.share !== null && mpd?.share !== undefined;

  const canShowAutomount = !(isMounted && hasEnabledShare) && mpd;
  if (canShowAutomount) {
    if (mpd?.is_to_mount_at_startup) {
      actionItems.push({
        key: "disable-automount",
        title: "Disable automatic mount",
        color: "primary",
        onClick: () => onToggleAutomount(partition, mpd),
      });
    } else {
      actionItems.push({
        key: "enable-automount",
        title: "Enable automatic mount",
        color: "primary",
        onClick: () => onToggleAutomount(partition, mpd),
      });
    }
  }

  if (!isMounted) {
    actionItems.push({
      key: "mount",
      title: "Mount Partition",
      color: undefined,
      onClick: () => onMount(partition, mpd),
    });
  } else {
    if (hasShare) {
      actionItems.push({
        key: "go-to-share",
        title: "Go to Share",
        color: undefined,
        onClick: () => onGoToShare(partition, mpd),
      });
    }

    // Always offer unmount controls for mounted addon partitions (#971):
    // gating on share/automount state previously left users with no UI
    // path to unmount a partition mounted with a share or automount on.
    actionItems.push({
      key: "unmount",
      title: "Unmount Partition",
      color: undefined,
      onClick: () => onUnmount(partition, false, mpd),
    });
    actionItems.push({
      key: "force-unmount",
      title: "Force Unmount Partition",
      color: "warning",
      onClick: () => onUnmount(partition, true, mpd),
    });

    if (!hasShare && mpd.path?.startsWith("/mnt/")) {
      actionItems.push({
        key: "create-share",
        title: "Create Share",
        color: "success",
        onClick: () => onCreateShare(partition, mpd),
      });
    }
  }

  // Additional Action on supported filesystems
  if (
    onCheckFilesystem &&
    partition.filesystem_info?.support?.canCheck &&
    !isMounted
  ) {
    actionItems.push({
      key: "check-filesystem",
      title: "Check Filesystem",
      color: "info",
      onClick: () => onCheckFilesystem?.(partition),
    });
  }
  if (
    onSetFilesystemLabel &&
    partition.filesystem_info?.support?.canSetLabel &&
    !isMounted
  ) {
    actionItems.push({
      key: "set-label",
      title: "Set Label",
      color: "info",
      onClick: () => onSetFilesystemLabel?.(partition),
    });
  }
  if (
    onFormatPartition &&
    partition.filesystem_info?.support?.canFormat &&
    !isMounted
  ) {
    actionItems.push({
      key: "format",
      title: "Format Partition",
      color: "error",
      onClick: () => onFormatPartition?.(partition),
    });
  }
  return actionItems;
}
