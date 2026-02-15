import type { MountPointData, Partition } from "../../../store/sratApi";

export type PartitionActionKey =
    | "mount"
    | "enable-automount"
    | "disable-automount"
    | "unmount"
    | "force-unmount"
    | "create-share"
    | "go-to-share";

export interface PartitionActionItem {
    key: PartitionActionKey;
    title: string;
    onClick: () => void;
}

export interface PartitionActionOptions {
    partition: Partition;
    protectedMode: boolean;
    onToggleAutomount: (partition: Partition) => void;
    onMount: (partition: Partition) => void;
    onUnmount: (partition: Partition, force: boolean) => void;
    onCreateShare: (partition: Partition) => void;
    onGoToShare: (partition: Partition) => void;
}

export function getPartitionActionItems({
    partition,
    protectedMode,
    onToggleAutomount,
    onMount,
    onUnmount,
    onCreateShare,
    onGoToShare,
}: PartitionActionOptions): PartitionActionItem[] | null {
    if (
        protectedMode ||
        partition.name?.startsWith("hassos-") ||
        Object.values(partition.host_mount_point_data || {}).length > 0
    ) {
        return null;
    }

    const actionItems: PartitionActionItem[] = [];
    const mountPointData = partition.mount_point_data || {};
    const keys = Object.keys(mountPointData);

    if (!partition.mount_point_data || keys.length === 0) {
        actionItems.push({
            key: "mount",
            title: "Mount Partition",
            onClick: () => onMount(partition),
        });
        return actionItems;
    }

    if (keys.length === 1 && keys[0] && mountPointData[keys[0]]) {
        const mpd = mountPointData[keys[0]] as MountPointData;
        const isMounted = mpd?.is_mounted;
        const hasEnabledShare = mpd?.share && mpd?.share.disabled === false;
        const hasShare = mpd?.share !== null && mpd?.share !== undefined;
        const hadNoShareOrIsDisabled =
            !hasShare || (mpd?.share && mpd?.share.disabled === true);

        const canShowAutomount = !(isMounted && hasEnabledShare);
        if (canShowAutomount) {
            if (mpd?.is_to_mount_at_startup) {
                actionItems.push({
                    key: "disable-automount",
                    title: "Disable automatic mount",
                    onClick: () => onToggleAutomount(partition),
                });
            } else {
                actionItems.push({
                    key: "enable-automount",
                    title: "Enable automatic mount",
                    onClick: () => onToggleAutomount(partition),
                });
            }
        }

        if (!isMounted) {
            actionItems.push({
                key: "mount",
                title: "Mount Partition",
                onClick: () => onMount(partition),
            });
        } else {
            if (hasShare) {
                actionItems.push({
                    key: "go-to-share",
                    title: "Go to Share",
                    onClick: () => onGoToShare(partition),
                });
            }

            if (hadNoShareOrIsDisabled && !mpd?.is_to_mount_at_startup) {
                actionItems.push({
                    key: "unmount",
                    title: "Unmount Partition",
                    onClick: () => onUnmount(partition, false),
                });
                actionItems.push({
                    key: "force-unmount",
                    title: "Force Unmount Partition",
                    onClick: () => onUnmount(partition, true),
                });
            }

            if (!hasShare && mpd.path?.startsWith("/mnt/")) {
                actionItems.push({
                    key: "create-share",
                    title: "Create Share",
                    onClick: () => onCreateShare(partition),
                });
            }
        }

        return actionItems;
    }

    console.warn("Partition has no mount_point_data:", partition);
    return null;
}
