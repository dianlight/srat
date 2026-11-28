import { useMemo } from "react";

import type { MountPointData, Partition } from "../../../store/sratApi";
import { getPartitionActionItems } from "../components/PartitionActionItems";

export interface UsePartitionActionsParams {
  partition?: Partition;
  protectedMode: boolean;
  // Selected mount point for a multi-mount partition (tree leaf / details row).
  mountPointDataSelection?: MountPointData;
  onToggleAutomount?: (
    partition: Partition,
    mountPointData?: MountPointData,
  ) => void;
  onMount?: (partition: Partition, mountPointData?: MountPointData) => void;
  onUnmount?: (
    partition: Partition,
    force: boolean,
    mountPointData?: MountPointData,
  ) => void;
  onCreateShare?: (
    partition: Partition,
    mountPointData?: MountPointData,
  ) => void;
  onGoToShare?: (partition: Partition, mountPointData?: MountPointData) => void;
  onCheckFilesystem?: (partition: Partition) => void;
  onSetFilesystemLabel?: (partition: Partition) => void;
  onFormatPartition?: (partition: Partition) => void;
}

export function usePartitionActions({
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
}: UsePartitionActionsParams) {
  return useMemo(() => {
    if (!partition) return null;
    if (
      !onToggleAutomount ||
      !onMount ||
      !onUnmount ||
      !onCreateShare ||
      !onGoToShare
    ) {
      return null;
    }
    return getPartitionActionItems({
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
    });
  }, [
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
  ]);
}
