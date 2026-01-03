import BackupIcon from "@mui/icons-material/Backup";
import EditIcon from "@mui/icons-material/Edit";
import VisibilityIcon from "@mui/icons-material/Visibility";
import FolderSpecialIcon from "@mui/icons-material/FolderSpecial";
import StorageIcon from "@mui/icons-material/Storage";
import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import ExpandLessIcon from "@mui/icons-material/ExpandLess";
import {
    Box,
    Card,
    CardContent,
    CardHeader,
    Chip,
    Collapse,
    Divider,
    Grid,
    IconButton,
    Stack,
    Tooltip,
    Typography,
} from "@mui/material";
import { filesize } from "filesize";
import { useState } from "react";
import { type SharedResource, Time_machine_support, Usage } from "../../../store/sratApi";
import type { ShareEditProps } from "../types";
import { PreviewDialog } from "../../../components/PreviewDialog";

interface ShareDetailsPanelProps {
    share?: SharedResource;
    shareKey?: string;
    onEdit?: (data: ShareEditProps) => void;
    onDelete?: (shareName: string, shareData: SharedResource) => void;
    onEditClick?: () => void; // New prop for edit button click
    onCancelEdit?: () => void; // New prop for cancel edit button click
    isEditing?: boolean; // New prop to indicate if we're in edit mode
    children?: React.ReactNode; // For embedding the ShareEditForm
}

export function ShareDetailsPanel({
    share,
    shareKey,
    onEdit,
    onDelete,
    onEditClick,
    onCancelEdit,
    isEditing = false,
    children,
}: ShareDetailsPanelProps) {
    const [mountPointExpanded, setMountPointExpanded] = useState(false);
    const [showMountPointPreview, setShowMountPointPreview] = useState(false);
    if (!share || !shareKey) {
        return (
            <Box
                sx={{
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                    height: "100%",
                    color: "text.secondary",
                }}
            >
                <Typography variant="h6">
                    Select a share to view details
                </Typography>
            </Box>
        );
    }

    const mountData = share.mount_point_data;

    return (
        <Box
            sx={{
                height: "100%",
                overflow: "auto",
                p: 2,
                opacity: share.mount_point_data?.invalid ? 0.6 : 1,
                pointerEvents: share.mount_point_data?.invalid ? "none" : "auto",
                filter: share.mount_point_data?.invalid ? "grayscale(50%)" : "none",
                transition: "opacity 0.3s, filter 0.3s"
            }}
        >
            <Stack spacing={3}>
                {/* Mount Point Information Card */}
                <Card sx={{ position: "relative" }}>
                    {share.mount_point_data?.invalid && (
                        <Box
                            sx={{
                                position: "absolute",
                                top: 0,
                                left: 0,
                                right: 0,
                                bottom: 0,
                                backgroundColor: "rgba(0, 0, 0, 0.05)",
                                zIndex: 1,
                                pointerEvents: "none",
                                display: "flex",
                                alignItems: "center",
                                justifyContent: "center",
                            }}
                        >
                            <Chip
                                label="Share Disabled"
                                color="error"
                                variant="filled"
                                size="small"
                                sx={{
                                    position: "absolute",
                                    top: 8,
                                    right: 8,
                                    fontWeight: "bold",
                                }}
                            />
                        </Box>
                    )}
                    <CardHeader
                        title={
                            <Box sx={{ display: "flex", alignItems: "center", justifyContent: "space-between", width: "100%" }}>
                                <Typography variant="h6">Mount Point Information</Typography>
                                {!mountPointExpanded && (
                                    <Stack direction="row" spacing={1}>
                                        {mountData?.disk_label && (
                                            <Chip
                                                size="small"
                                                icon={<StorageIcon />}
                                                variant="outlined"
                                                label={mountData.disk_label}
                                                sx={{ fontSize: "0.75rem", height: 24 }}
                                            />
                                        )}
                                        {mountData?.disk_size && (
                                            <Chip
                                                size="small"
                                                variant="outlined"
                                                label={filesize(mountData.disk_size, { round: 1 })}
                                                sx={{ fontSize: "0.75rem", height: 24 }}
                                            />
                                        )}
                                    </Stack>
                                )}
                            </Box>
                        }
                        avatar={
                            <Tooltip title="View mount point details">
                                <IconButton
                                    onClick={() => setShowMountPointPreview(true)}
                                    size="small"
                                >
                                    <StorageIcon color="primary" />
                                </IconButton>
                            </Tooltip>
                        }
                        action={
                            <IconButton
                                onClick={() => setMountPointExpanded(!mountPointExpanded)}
                                aria-expanded={mountPointExpanded}
                                aria-label="show more"
                            >
                                {mountPointExpanded ? <ExpandLessIcon /> : <ExpandMoreIcon />}
                            </IconButton>
                        }
                    />
                    <Collapse in={mountPointExpanded} timeout="auto">
                        <CardContent>
                            <Grid container spacing={2}>
                                <Grid size={{ xs: 12, sm: 6 }}>
                                    <Typography variant="subtitle2" color="text.secondary">
                                        Path
                                    </Typography>
                                    <Typography variant="body2" sx={{ fontFamily: "monospace" }}>
                                        {mountData?.path || "N/A"}
                                    </Typography>
                                </Grid>
                                <Grid size={{ xs: 12, sm: 6 }}>
                                    <Typography variant="subtitle2" color="text.secondary">
                                        Device
                                    </Typography>
                                    <Typography variant="body2">
                                        {mountData?.device_id || "N/A"}
                                    </Typography>
                                </Grid>
                                <Grid size={{ xs: 12, sm: 6 }}>
                                    <Typography variant="subtitle2" color="text.secondary">
                                        Disk Label
                                    </Typography>
                                    <Typography variant="body2">
                                        {mountData?.disk_label || "N/A"}
                                    </Typography>
                                </Grid>
                                <Grid size={{ xs: 12, sm: 6 }}>
                                    <Typography variant="subtitle2" color="text.secondary">
                                        File System
                                    </Typography>
                                    <Typography variant="body2">
                                        {mountData?.fstype || "N/A"}
                                    </Typography>
                                </Grid>
                                {mountData?.disk_size && (
                                    <Grid size={{ xs: 12, sm: 6 }}>
                                        <Typography variant="subtitle2" color="text.secondary">
                                            Size
                                        </Typography>
                                        <Typography variant="body2">
                                            {filesize(mountData.disk_size, { round: 1 })}
                                        </Typography>
                                    </Grid>
                                )}
                                <Grid size={{ xs: 12, sm: 6 }}>
                                    <Typography variant="subtitle2" color="text.secondary">
                                        Write Support
                                    </Typography>
                                    <Chip
                                        label={mountData?.is_write_supported ? "Yes" : "Read-Only"}
                                        color={mountData?.is_write_supported ? "success" : "warning"}
                                        size="small"
                                    />
                                </Grid>
                                <Grid size={{ xs: 12, sm: 6 }}>
                                    <Typography variant="subtitle2" color="text.secondary">
                                        Mounted
                                    </Typography>
                                    <Chip
                                        label={mountData?.is_mounted ? "Yes" : "No"}
                                        color={mountData?.is_mounted ? "success" : "error"}
                                        size="small"
                                    />
                                </Grid>
                                {mountData?.time_machine_support && (
                                    <Grid size={{ xs: 12, sm: 6 }}>
                                        <Typography variant="subtitle2" color="text.secondary">
                                            Time Machine Support
                                        </Typography>
                                        <Chip
                                            label={mountData.time_machine_support}
                                            color={
                                                mountData.time_machine_support === Time_machine_support.Supported
                                                    ? "success"
                                                    : mountData.time_machine_support === Time_machine_support.Experimental
                                                        ? "warning"
                                                        : "error"
                                            }
                                            size="small"
                                        />
                                    </Grid>
                                )}
                            </Grid>

                            {/* Warnings and Errors */}
                            {mountData?.warnings && share.usage !== Usage.Internal && (
                                <Box sx={{ mt: 2 }}>
                                    <Typography variant="subtitle2" color="warning.main">
                                        Warnings
                                    </Typography>
                                    <Typography variant="body2" color="warning.main">
                                        {mountData.warnings}
                                    </Typography>
                                </Box>
                            )}

                            {mountData?.invalid && mountData?.invalid_error && (
                                <Box sx={{ mt: 2 }}>
                                    <Typography variant="subtitle2" color="error.main">
                                        Error
                                    </Typography>
                                    <Typography variant="body2" color="error.main">
                                        {mountData.invalid_error}
                                    </Typography>
                                </Box>
                            )}
                        </CardContent>
                    </Collapse>
                </Card>

                {/* Share Information Card */}
                <Card sx={{ position: "relative" }}>
                    {share.disabled && (
                        <Box
                            sx={{
                                position: "absolute",
                                top: 0,
                                left: 0,
                                right: 0,
                                bottom: 0,
                                backgroundColor: "rgba(0, 0, 0, 0.05)",
                                zIndex: 1,
                                pointerEvents: "none",
                            }}
                        />
                    )}
                    <CardHeader
                        title="Share Configuration"
                        avatar={
                            <Tooltip title={isEditing ? "View Share" : "Edit Share"}>
                                <span>
                                    <IconButton
                                        color="primary"
                                        size="small"
                                        onClick={isEditing ? onCancelEdit : onEditClick}
                                        disabled={isEditing ? !onCancelEdit : !onEditClick}
                                    >
                                        {isEditing ? <VisibilityIcon /> : <EditIcon />}
                                    </IconButton>
                                </span>
                            </Tooltip>
                        }
                    />
                    {isEditing && children ? (
                        // Render the ShareEditForm directly in place of CardContent
                        children
                    ) : (
                        <CardContent>
                            <Grid container spacing={2}>
                                <Grid size={{ md: 6, sm: 12 }}>
                                    <Typography variant="subtitle2" color="text.secondary">
                                        Share Name
                                    </Typography>
                                    <Typography variant="h6">
                                        {share.name}
                                    </Typography>
                                </Grid>

                                {/* Share Properties */}
                                <Grid size={{ xs: 6, sm: 12 }}>
                                    <Stack direction="row" spacing={1} flexWrap="wrap" sx={{ gap: 1 }}>
                                        {share.usage && share.usage !== Usage.Internal && (
                                            <Chip
                                                icon={<FolderSpecialIcon />}
                                                label={`Usage: ${share.usage}`}
                                                variant="outlined"
                                                color={share.is_ha_mounted ? "primary" : "default"}
                                            />
                                        )}
                                        {share.timemachine && (
                                            <Chip
                                                icon={<BackupIcon />}
                                                label="Time Machine"
                                                variant="outlined"
                                                color="secondary"
                                            />
                                        )}
                                        {share.recycle_bin_enabled && (
                                            <Chip
                                                label="Recycle Bin"
                                                variant="outlined"
                                                color="info"
                                            />
                                        )}
                                        {share.guest_ok && (
                                            <Chip
                                                label="Guest Access"
                                                variant="outlined"
                                                color="warning"
                                            />
                                        )}
                                        {share.disabled && (
                                            <Chip
                                                label="Disabled"
                                                variant="outlined"
                                                color="error"
                                            />
                                        )}
                                    </Stack>
                                </Grid>

                                {/* Users */}
                                {share.users && share.users.length > 0 && (
                                    <Grid size={{ md: 6, sm: 12 }}>
                                        <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 1 }}>
                                            Read/Write Users
                                        </Typography>
                                        <Stack direction="row" spacing={1} flexWrap="wrap" sx={{ gap: 1 }}>
                                            {share.users.map((user) => (
                                                <Chip
                                                    key={user.username}
                                                    icon={<EditIcon />}
                                                    label={user.username}
                                                    variant="outlined"
                                                    color={user.is_admin ? "warning" : "default"}
                                                    size="small"
                                                />
                                            ))}
                                        </Stack>
                                    </Grid>
                                )}

                                {/* Read-Only Users */}
                                {share.ro_users && share.ro_users.length > 0 && (
                                    <Grid size={{ md: 6, sm: 12 }}>
                                        <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 1 }}>
                                            Read-Only Users
                                        </Typography>
                                        <Stack direction="row" spacing={1} flexWrap="wrap" sx={{ gap: 1 }}>
                                            {share.ro_users.map((user) => (
                                                <Chip
                                                    key={user.username}
                                                    icon={<VisibilityIcon />}
                                                    label={user.username}
                                                    variant="outlined"
                                                    color={user.is_admin ? "warning" : "default"}
                                                    size="small"
                                                />
                                            ))}
                                        </Stack>
                                    </Grid>
                                )}

                                {/* Veto Files */}
                                {share.veto_files && share.veto_files.length > 0 && (
                                    <Grid size={{ md: 6, sm: 12 }}>
                                        <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 1 }}>
                                            Veto Files
                                        </Typography>
                                        <Stack direction="row" spacing={1} flexWrap="wrap" sx={{ gap: 1 }}>
                                            {share.veto_files.map((vetoFile, index) => (
                                                <Chip
                                                    key={index}
                                                    label={vetoFile}
                                                    variant="outlined"
                                                    size="small"
                                                />
                                            ))}
                                        </Stack>
                                    </Grid>
                                )}

                                {/* Time Machine Settings */}
                                {share.timemachine && share.timemachine_max_size && (
                                    <Grid size={{ md: 6, sm: 12 }}>
                                        <Typography variant="subtitle2" color="text.secondary">
                                            Time Machine Max Size
                                        </Typography>
                                        <Typography variant="body2">
                                            {share.timemachine_max_size}
                                        </Typography>
                                    </Grid>
                                )}
                            </Grid>
                        </CardContent>
                    )}
                </Card>
            </Stack>

            {/* Mount Point Preview Dialog */}
            <PreviewDialog
                title={`Mount Point: ${mountData?.path || 'N/A'}`}
                objectToDisplay={mountData}
                open={showMountPointPreview}
                onClose={() => setShowMountPointPreview(false)}
            />
        </Box>
    );
} 