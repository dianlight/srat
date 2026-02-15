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
    color: "primary" | "secondary" | "warning" | "error" | "info" | "success" | undefined;
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
            color: undefined,
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
                    color: "primary",
                    onClick: () => onToggleAutomount(partition),
                });
            } else {
                actionItems.push({
                    key: "enable-automount",
                    title: "Enable automatic mount",
                    color: "primary",
                    onClick: () => onToggleAutomount(partition),
                });
            }
        }

        if (!isMounted) {
            actionItems.push({
                key: "mount",
                title: "Mount Partition",
                color: undefined,
                onClick: () => onMount(partition),
            });
        } else {
            if (hasShare) {
                actionItems.push({
                    key: "go-to-share",
                    title: "Go to Share",
                    color: undefined,
                    onClick: () => onGoToShare(partition),
                });
            }

            if (hadNoShareOrIsDisabled && !mpd?.is_to_mount_at_startup) {
                actionItems.push({
                    key: "unmount",
                    title: "Unmount Partition",
                    color: undefined,
                    onClick: () => onUnmount(partition, false),
                });
                actionItems.push({
                    key: "force-unmount",
                    title: "Force Unmount Partition",
                    color: "warning",
                    onClick: () => onUnmount(partition, true),
                });
            }

            if (!hasShare && mpd.path?.startsWith("/mnt/")) {
                actionItems.push({
                    key: "create-share",
                    title: "Create Share",
                    color: "success",
                    onClick: () => onCreateShare(partition),
                });
            }
        }

        // Additional Action on supported filesystems 
        if (partition.filesystem_info?.Support?.canCheck && !isMounted) {
            actionItems.push({
                key: "check-filesystem",
                title: "Check Filesystem",
                color: "info",
                onClick: () => {
                    // Implement filesystem check action here
                    console.log("Checking filesystem for partition:", partition.name);
                },
            });
        }
        if (partition.filesystem_info?.Support?.canSetLabel && !isMounted) {
            actionItems.push({
                key: "set-label",
                title: "Set Label",
                color: "info",
                onClick: () => {
                    // Implement set label action here
                    console.log("Setting label for partition:", partition.name);
                },
            });
        }
        if (partition.filesystem_info?.Support?.canFormat && !isMounted) {
            actionItems.push({
                key: "format",
                title: "Format Partition",
                color: "error",
                onClick: () => {
                    // Implement format action here
                    console.log("Formatting partition:", partition.name);
                },
            });
        }
        return actionItems;
    }

    console.warn("Partition has no mount_point_data:", partition);
    return null;
}
