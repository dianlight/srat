import FolderSharedIcon from "@mui/icons-material/FolderShared";
import FolderSpecialIcon from "@mui/icons-material/FolderSpecial";
import StorageIcon from "@mui/icons-material/Storage";
import VisibilityIcon from "@mui/icons-material/Visibility";
import EditIcon from "@mui/icons-material/Edit";
import CheckCircleIcon from "@mui/icons-material/CheckCircle";
import BlockIcon from "@mui/icons-material/Block";
import {
    Box,
    Chip,
    IconButton,
    Tooltip,
    Typography,
    useTheme,
} from "@mui/material";
import { SimpleTreeView } from "@mui/x-tree-view/SimpleTreeView";
import { TreeItem } from "@mui/x-tree-view/TreeItem";
import { useConfirm } from "material-ui-confirm";
import { Fragment, useMemo } from "react";
import { toast } from "react-toastify";
import { addMessage } from "../../../store/errorSlice";
import {
    type SharedResource,
    Usage,
    usePutApiShareByShareNameDisableMutation,
    usePutApiShareByShareNameEnableMutation,
} from "../../../store/sratApi";
import { useAppDispatch } from "../../../store/store";
import type { Shared } from "react-redux";

interface SharesTreeViewTestOverrides {
    dispatch?: (action: unknown) => void;
    confirm?: (options: Parameters<ReturnType<typeof useConfirm>>[0]) => Promise<any>;
    enableShare?: (params: { shareName: string }) => Promise<any>;
    disableShare?: (params: { shareName: string }) => Promise<any>;
}

interface SharesTreeViewProps {
    shares?: Record<string, SharedResource> | SharedResource[];
    selectedShareKey?: string;
    onShareSelect: (shareKey: string, share: SharedResource) => void;
    protectedMode?: boolean;
    readOnly?: boolean;
    // Controlled expanded items and change callback (required)
    expandedItems: string[];
    onExpandedItemsChange: (items: string[]) => void;
    testOverrides?: SharesTreeViewTestOverrides;
}

export function SharesTreeView({
    shares,
    selectedShareKey,
    onShareSelect,
    protectedMode = false,
    readOnly = false,
    expandedItems,
    onExpandedItemsChange,
    testOverrides,
}: SharesTreeViewProps) {
    const theme = useTheme();
    const dispatch = testOverrides?.dispatch ?? useAppDispatch();
    const confirm = testOverrides?.confirm ?? useConfirm();
    const [enableShareMutation] = usePutApiShareByShareNameEnableMutation();
    const [disableShareMutation] = usePutApiShareByShareNameDisableMutation();
    const enableShare =
        testOverrides?.enableShare ??
        ((params: { shareName: string }) => enableShareMutation(params).unwrap());
    const disableShare =
        testOverrides?.disableShare ??
        ((params: { shareName: string }) => disableShareMutation(params).unwrap());

    const groupedAndSortedShares = useMemo(() => {
        if (!shares) {
            return [];
        }

        const groups: Record<string, Array<[string, SharedResource]>> = {};

        Object.entries(shares).forEach(([shareKey, shareProps]) => {
            const usageGroup = shareProps.usage || Usage.None;

            // In protected mode, only show internal shares
            if (protectedMode && usageGroup !== Usage.Internal) {
                return;
            }

            if (!groups[usageGroup]) {
                groups[usageGroup] = [];
            }
            groups[usageGroup].push([shareKey, shareProps]);
        });

        // Sort shares within each group by name
        for (const usageGroup in groups) {
            groups[usageGroup].sort((a, b) =>
                (a[1].name || "").localeCompare(b[1].name || ""),
            );
        }

        // Sort the groups by usage type (key)
        return Object.entries(groups).sort((a, b) => a[0].localeCompare(b[0]));
    }, [shares, protectedMode]);

    const handleToggleShare = async (
        event: React.MouseEvent<HTMLButtonElement>,
        shareKey: string,
        shareProps: SharedResource,
    ) => {
        event.stopPropagation();
        const isEnabled = !shareProps.disabled;
        const shareName = shareProps.name || "";

        if (isEnabled) {
            // Currently enabled, disable it
            try {
                await confirm({
                    title: `Disable ${shareName}?`,
                    description:
                        "If you disable this share, all of its configurations will be retained.",
                    acknowledgement:
                        "I understand that disabling the share will retain its configurations but prevent access to it.",
                });

                await disableShare({ shareName });
            } catch (error: any) {
                if (error.confirmed === false) {
                    return; // User cancelled
                }
                dispatch(addMessage(JSON.stringify(error)));
            }
        } else {
            // Currently disabled, enable it
            try {
                await enableShare({ shareName });
            } catch (error) {
                dispatch(addMessage(JSON.stringify(error)));
            }
        }
    };

    const renderShareItem = (shareKey: string, shareProps: SharedResource) => {
        const isSelected = selectedShareKey === shareKey;
        const isDisabled = shareProps.disabled;

        return (
            <TreeItem
                key={shareKey}
                itemId={shareKey}
                label={
                    <Box
                        sx={{
                            display: "flex",
                            alignItems: "center",
                            py: 0.5,
                            px: 1,
                            opacity: isDisabled ? 0.5 : 1,
                            backgroundColor: isSelected ? theme.palette.action.selected : "transparent",
                            borderRadius: 1,
                            "&:hover": {
                                backgroundColor: theme.palette.action.hover,
                            },
                        }}
                        onClick={(e) => {
                            e.stopPropagation();
                            onShareSelect(shareKey, shareProps);
                        }}
                    >
                        {shareProps.mount_point_data?.invalid ? (
                            <Tooltip title={shareProps.mount_point_data?.invalid_error} arrow>
                                <FolderSharedIcon color="error" sx={{ mr: 1 }} />
                            </Tooltip>
                        ) : (
                            <Tooltip title={shareProps.mount_point_data?.warnings} arrow>
                                <FolderSharedIcon sx={{ mr: 1 }} />
                            </Tooltip>
                        )}

                        <Box sx={{ flexGrow: 1, mr: 1 }}>
                            <Typography variant="body2" fontWeight={isSelected ? 600 : 400}>
                                {shareProps.name}
                            </Typography>
                            <Box sx={{ display: "flex", flexWrap: "wrap", gap: 0.5, mt: 0.5 }}>
                                {shareProps.mount_point_data?.disk_label && (
                                    <Chip
                                        size="small"
                                        icon={<StorageIcon />}
                                        variant="outlined"
                                        label={shareProps.mount_point_data.disk_label}
                                        sx={{ fontSize: "0.7rem", height: 16 }}
                                    />
                                )}
                                {!shareProps.mount_point_data?.is_write_supported && (
                                    <Chip
                                        label="Read-Only"
                                        size="small"
                                        variant="outlined"
                                        color="secondary"
                                        sx={{ fontSize: "0.7rem", height: 16 }}
                                    />
                                )}
                                {shareProps.users && shareProps.users.length > 0 && (
                                    <Chip
                                        size="small"
                                        icon={<EditIcon />}
                                        variant="outlined"
                                        label={`Users: ${shareProps.users.length}`}
                                        sx={{ fontSize: "0.7rem", height: 16 }}
                                    />
                                )}
                                {shareProps.ro_users && shareProps.ro_users.length > 0 && (
                                    <Chip
                                        size="small"
                                        icon={<VisibilityIcon />}
                                        variant="outlined"
                                        label={`RO: ${shareProps.ro_users.length}`}
                                        sx={{ fontSize: "0.7rem", height: 16 }}
                                    />
                                )}
                                {shareProps.usage && shareProps.usage !== Usage.Internal && (
                                    <Chip
                                        size="small"
                                        icon={<FolderSpecialIcon />}
                                        variant="outlined"
                                        label={shareProps.usage}
                                        sx={{ fontSize: "0.7rem", height: 16 }}
                                    />
                                )}
                            </Box>
                        </Box>

                        {!readOnly && shareProps.mount_point_data && (
                            <Tooltip title={isDisabled ? "Enable share" : "Disable share"}>
                                <IconButton
                                    aria-label={isDisabled ? "enable share" : "disable share"}
                                    data-testid={`share-toggle-${shareKey}`}
                                    size="small"
                                    onClick={(e) => handleToggleShare(e, shareKey, shareProps)}
                                    sx={{
                                        color: isDisabled ? "action.disabled" : "success.main",
                                        "&:hover": {
                                            backgroundColor: isDisabled ? "success.light" : "error.light",
                                            color: isDisabled ? "success.main" : "error.main",
                                        },
                                    }}
                                >
                                    {isDisabled ? <BlockIcon /> : <CheckCircleIcon />}
                                </IconButton>
                            </Tooltip>
                        )}
                    </Box>
                }
            />
        );
    };

    return (
        <Box sx={{ height: "100%", overflow: "auto" }}>
            <SimpleTreeView
                selectedItems={selectedShareKey || ""}
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
                {groupedAndSortedShares.map(([usageGroup, sharesInGroup]) => (
                    <TreeItem
                        key={`group-${usageGroup}`}
                        itemId={`group-${usageGroup}`}
                        label={
                            <Typography
                                variant="subtitle2"
                                color="text.primary"
                                sx={{
                                    textTransform: "capitalize",
                                    py: 1,
                                    fontWeight: 600
                                }}
                            >
                                {usageGroup} Shares ({sharesInGroup.length})
                            </Typography>
                        }
                    >
                        {sharesInGroup.map(([shareKey, shareProps]) =>
                            renderShareItem(shareKey, shareProps),
                        )}
                    </TreeItem>
                ))}
            </SimpleTreeView>
        </Box>
    );
}