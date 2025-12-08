import ComputerIcon from "@mui/icons-material/Computer";
import CreditScoreIcon from "@mui/icons-material/CreditScore";
import EjectIcon from "@mui/icons-material/Eject";
import SdStorageIcon from "@mui/icons-material/SdStorage";
import SettingsSuggestIcon from "@mui/icons-material/SettingsSuggest";
import StorageIcon from "@mui/icons-material/Storage";
import UsbIcon from "@mui/icons-material/Usb";
import {
    Box,
    Chip,
    Tooltip,
    Typography,
    useTheme,
} from "@mui/material";
import { SimpleTreeView } from "@mui/x-tree-view/SimpleTreeView";
import { TreeItem } from "@mui/x-tree-view/TreeItem";
import { filesize } from "filesize";
import { useMemo } from "react";
import { type Disk, type Partition } from "../../../store/sratApi";
import { decodeEscapeSequence } from "../utils";
import { PartitionActions } from "./PartitionActions";

interface VolumesTreeViewProps {
    disks?: Disk[];
    // Selected item id can be either a disk id or a partition id
    selectedItemId?: string;
    // Backward-compat for older callers/tests
    selectedPartitionId?: string;
    hideSystemPartitions?: boolean;
    // Controlled expanded items and change callback (required)
    expandedItems: string[];
    onExpandedItemsChange: (items: string[]) => void;
    // Selection handlers
    onDiskSelect?: (disk: Disk) => void;
    onPartitionSelect: (disk: Disk, partition: Partition) => void;
    onToggleAutomount: (partition: Partition) => void;
    onMount: (partition: Partition) => void;
    onUnmount: (partition: Partition, force: boolean) => void;
    onCreateShare: (partition: Partition) => void;
    onGoToShare: (partition: Partition) => void;
    protectedMode?: boolean;
    readOnly?: boolean;
}

export function VolumesTreeView({
    disks,
    selectedItemId,
    selectedPartitionId,
    hideSystemPartitions = true,
    expandedItems,
    onExpandedItemsChange,
    onDiskSelect,
    onPartitionSelect,
    onToggleAutomount,
    onMount,
    onUnmount,
    onCreateShare,
    onGoToShare,
    protectedMode = false,
    readOnly = false,
}: VolumesTreeViewProps) {
    const theme = useTheme();
    // Normalize selected id to support both the new and legacy prop name
    const normalizedSelectedId = selectedItemId ?? selectedPartitionId;

    const filteredDisks = useMemo(() => {
        if (!disks) return [];

        return disks.filter((disk) => {
            const partitions = Object.values(disk.partitions || {});
            if (partitions.length === 0) return false;

            const visiblePartitions = partitions.filter(
                (partition) =>
                    !(
                        hideSystemPartitions &&
                        (partition.system &&
                            (partition.name?.startsWith("hassos-") ||
                                (Object.values(partition.host_mount_point_data || {}).length > 0)))
                    ),
            );

            return visiblePartitions.length > 0;
        });
    }, [disks, hideSystemPartitions]);

    // Helper function to render disk icon
    const renderDiskIcon = (disk: Disk) => {
        switch (disk.connection_bus?.toLowerCase()) {
            case "usb":
                return <UsbIcon />;
            case "sdio":
            case "mmc":
                return <SdStorageIcon />;
        }
        if (disk.removable) {
            return <EjectIcon />;
        }
        return <ComputerIcon />;
    };

    // Helper function to render partition icon
    const renderPartitionIcon = (partition: Partition) => {
        const isToMountAtStartup =
            partition.mount_point_data?.[0]?.is_to_mount_at_startup === true;
        const iconColorProp = isToMountAtStartup
            ? { color: "primary" as const }
            : {};

        if (partition.name === "hassos-data") {
            return <CreditScoreIcon fontSize="small" {...iconColorProp} />;
        }
        if (
            partition.system ||
            partition.name?.startsWith("hassos-") ||
            (Object.values(partition.host_mount_point_data || {}).length > 0)
        ) {
            return <SettingsSuggestIcon fontSize="small" {...iconColorProp} />;
        }
        return <StorageIcon fontSize="small" {...iconColorProp} />;
    };

    const renderPartitionItem = (disk: Disk, partition: Partition, diskIdx: number, partIdx: number) => {
        const partitionIdentifier = partition.id || `${disk.id || `disk-${diskIdx}`}-part-${partIdx}`;
        const isSelected = normalizedSelectedId === partitionIdentifier;
        const partitionNameDecoded = decodeEscapeSequence(
            partition.name || partition.id || "Unnamed Partition",
        );
        const mpds = Object.values(partition.mount_point_data || {});
        const isMounted = mpds.some((mpd) => mpd.is_mounted);

        return (
            <TreeItem
                key={partitionIdentifier}
                itemId={partitionIdentifier}
                label={
                    <Box
                        sx={{
                            display: "flex",
                            alignItems: "center",
                            py: 0.5,
                            px: 1,
                            backgroundColor: isSelected ? theme.palette.action.selected : "transparent",
                            borderRadius: 1,
                            "&:hover": {
                                backgroundColor: theme.palette.action.hover,
                            },
                        }}
                        onClick={(e) => {
                            e.stopPropagation();
                            onPartitionSelect(disk, partition);
                        }}
                    >
                        {renderPartitionIcon(partition)}

                        <Box sx={{ flexGrow: 1, ml: 1, mr: 1, minWidth: 0 }}>
                            <Tooltip title={partitionNameDecoded} placement="top">
                                <Typography
                                    variant="body2"
                                    fontWeight={isSelected ? 600 : 400}
                                    sx={{
                                        overflow: "hidden",
                                        textOverflow: "ellipsis",
                                        whiteSpace: "nowrap",
                                    }}
                                >
                                    {partitionNameDecoded}
                                </Typography>
                            </Tooltip>
                            <Box sx={{ display: "flex", flexWrap: "wrap", gap: 0.5, mt: 0.5 }}>
                                {partition.size != null && (
                                    <Chip
                                        label={filesize(partition.size, { round: 0 })}
                                        size="small"
                                        variant="outlined"
                                        sx={{ fontSize: "0.7rem", height: 16 }}
                                    />
                                )}
                                {mpds[0]?.fstype && (
                                    <Chip
                                        label={mpds[0]?.fstype}
                                        size="small"
                                        variant="outlined"
                                        sx={{ fontSize: "0.7rem", height: 16 }}
                                    />
                                )}
                                {isMounted && (
                                    <Chip
                                        label={mpds.length > 0 && mpds.every(mp => !mp.is_write_supported) ? "Mounted (Read-Only)" : "Mounted"}
                                        size="small"
                                        variant="outlined"
                                        color={mpds.length > 0 && mpds.every(mp => !mp.is_write_supported) ? "secondary" : "success"}
                                        sx={{ fontSize: "0.7rem", height: 16 }}
                                    />
                                )}
                            </Box>
                        </Box>

                        {!readOnly && (
                            <Box sx={{ flexShrink: 0 }}>
                                <PartitionActions
                                    partition={partition}
                                    protected_mode={protectedMode}
                                    onToggleAutomount={onToggleAutomount}
                                    onMount={onMount}
                                    onUnmount={onUnmount}
                                    onCreateShare={onCreateShare}
                                    onGoToShare={onGoToShare}
                                />
                            </Box>
                        )}
                    </Box>
                }
            />
        );
    };

    const renderDiskItem = (disk: Disk, diskIdx: number) => {
        const diskIdentifier = disk.id || `disk-${diskIdx}`;
        const partitions = Object.values(disk.partitions || {});
        const filteredPartitions = partitions.filter(
            (partition) =>
                !(
                    hideSystemPartitions &&
                    (partition.system &&
                        (partition.name?.startsWith("hassos-") ||
                            (Object.values(partition.host_mount_point_data || {}).length > 0)))
                ),
        ) || [];

        if (filteredPartitions.length === 0) return null;

        const isSelected = normalizedSelectedId === diskIdentifier;

        return (
            <TreeItem
                key={diskIdentifier}
                itemId={diskIdentifier}
                label={
                    <Box
                        sx={{
                            display: "flex",
                            alignItems: "center",
                            py: 1,
                            px: 1,
                            backgroundColor: isSelected ? theme.palette.action.selected : "transparent",
                            borderRadius: 1,
                            "&:hover": {
                                backgroundColor: theme.palette.action.hover,
                            },
                        }}
                        onClick={(e) => {
                            e.stopPropagation();
                            onDiskSelect?.(disk);
                        }}
                    >
                        {renderDiskIcon(disk)}

                        <Box sx={{ flexGrow: 1, ml: 1 }}>
                            <Typography variant="subtitle2" fontWeight={600}>
                                {disk.model?.toUpperCase() || `Disk ${diskIdx + 1}`}
                            </Typography>
                            <Box sx={{ display: "flex", flexWrap: "wrap", gap: 0.5, mt: 0.5 }}>
                                <Chip
                                    label={`${filteredPartitions.length} partition(s)`}
                                    size="small"
                                    variant="outlined"
                                    sx={{ fontSize: "0.7rem", height: 16 }}
                                />
                                {disk.size != null && (
                                    <Chip
                                        label={filesize(disk.size, { round: 1 })}
                                        size="small"
                                        variant="outlined"
                                        sx={{ fontSize: "0.7rem", height: 16 }}
                                    />
                                )}
                                {disk.connection_bus && (
                                    <Chip
                                        label={disk.connection_bus}
                                        size="small"
                                        variant="outlined"
                                        sx={{ fontSize: "0.7rem", height: 16 }}
                                    />
                                )}
                            </Box>
                        </Box>
                    </Box>
                }
            >
                {filteredPartitions.map((partition, partIdx) =>
                    renderPartitionItem(disk, partition, diskIdx, partIdx),
                )}
            </TreeItem>
        );
    };

    return (
        <Box sx={{ height: "100%", overflow: "auto" }}>
            <SimpleTreeView
                selectedItems={normalizedSelectedId || ""}
                expandedItems={expandedItems}
                onExpandedItemsChange={(_, items) => {
                    // SimpleTreeView may emit different shapes; normalize to string[] when possible
                    if (!items) return;
                    if (Array.isArray(items)) {
                        onExpandedItemsChange(items as string[]);
                    } else if ((items as any)?.items && Array.isArray((items as any).items)) {
                        onExpandedItemsChange((items as any).items as string[]);
                    }
                }}
            >
                {filteredDisks.map((disk, diskIdx) => renderDiskItem(disk, diskIdx))}
            </SimpleTreeView>
        </Box>
    );
} 