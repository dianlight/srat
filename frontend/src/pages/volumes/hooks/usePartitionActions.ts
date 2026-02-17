import { useMemo } from "react";
import { type Partition } from "../../../store/sratApi";
import { getPartitionActionItems } from "../components/partition-action-items";

export interface UsePartitionActionsParams {
    partition?: Partition;
    protectedMode: boolean;
    onToggleAutomount?: (partition: Partition) => void;
    onMount?: (partition: Partition) => void;
    onUnmount?: (partition: Partition, force: boolean) => void;
    onCreateShare?: (partition: Partition) => void;
    onGoToShare?: (partition: Partition) => void;
    onCheckFilesystem?: () => void;
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
}: UsePartitionActionsParams) {
    return useMemo(() => {
        if (!partition) return null;
        if (!onToggleAutomount || !onMount || !onUnmount || !onCreateShare || !onGoToShare) {
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
            onCheckFilesystem: onCheckFilesystem || (() => {}),
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
    ]);
}
