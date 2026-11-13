import { useMemo } from "react";

import type { MountPointData, Partition } from "../../../store/sratApi";
import { getPartitionActionItems } from "../components/PartitionActionItems";

export interface UsePartitionActionsParams {
  partition?: Partition;
  protectedMode: boolean;
  onToggleAutomount?: (partition: Partition) => void;
  onMount?: (partition: Partition) => void;
  onUnmount?: (partition: Partition, force: boolean) => void;
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
