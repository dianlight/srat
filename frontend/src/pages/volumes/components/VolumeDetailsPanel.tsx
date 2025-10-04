import BackupIcon from "@mui/icons-material/Backup";
import EditIcon from "@mui/icons-material/Edit";
import VisibilityIcon from "@mui/icons-material/Visibility";
import FolderSpecialIcon from "@mui/icons-material/FolderSpecial";
import StorageIcon from "@mui/icons-material/Storage";
import ComputerIcon from "@mui/icons-material/Computer";
import UsbIcon from "@mui/icons-material/Usb";
import SdStorageIcon from "@mui/icons-material/SdStorage";
import EjectIcon from "@mui/icons-material/Eject";
import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import SettingsIcon from "@mui/icons-material/Settings";
import {
    Box,
    Card,
    CardContent,
    CardHeader,
    Chip,
    Collapse,
    Grid,
    IconButton,
    Paper,
    Stack,
    Typography,
} from "@mui/material";
import { filesize } from "filesize";
import { useState } from "react";
import { useNavigate } from "react-router";
import { PreviewDialog } from "../../../components/PreviewDialog";
import { type LocationState, TabIDs } from "../../../store/locationState";
import { type Disk, type Partition, type SharedResource, Usage, Time_machine_support } from "../../../store/sratApi";
import { decodeEscapeSequence } from "../utils";

interface VolumeDetailsPanelProps {
    disk?: Disk;
    partition?: Partition;
    share?: SharedResource;
}

export function VolumeDetailsPanel({
    disk,
    partition,
    share,
}: VolumeDetailsPanelProps) {
    const navigate = useNavigate();
    const [diskInfoExpanded, setDiskInfoExpanded] = useState(false);
    const [previewOpen, setPreviewOpen] = useState(false);
    const [previewObject, setPreviewObject] = useState<any | null>(null);
    const [previewTitle, setPreviewTitle] = useState<string>("Preview");

    const openPreviewFor = (obj: any, title?: string) => {
        setPreviewObject(obj);
        setPreviewTitle(title ?? "Preview");
        setPreviewOpen(true);
    };
    const closePreview = () => {
        setPreviewOpen(false);
        setPreviewObject(null);
    };

    const navigateToShare = (share: SharedResource) => {
        if (share?.name) {
            // Navigate to the shares page and pass the share name as state
            navigate("/", {
                state: { tabId: TabIDs.SHARES, shareName: share.name } as LocationState,
            });
        }
    };

    if (!disk || !partition) {
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
                    Select a partition from the tree to view details
                </Typography>
            </Box>
        );
    }

    // Helper function to render disk icon
    const renderDiskIcon = (disk: Disk) => {
        switch (disk.connection_bus?.toLowerCase()) {
            case "usb":
                return <UsbIcon color="primary" />;
            case "sdio":
            case "mmc":
                return <SdStorageIcon color="primary" />;
        }
        if (disk.removable) {
            return <EjectIcon color="primary" />;
        }
        return <ComputerIcon color="primary" />;
    };

    const mountData = partition.mount_point_data?.[0];
    const allShares = partition.mount_point_data?.flatMap((mpd) => mpd.shares).filter(Boolean) || [];
    const isMounted = mountData?.is_mounted;

    return (
        <Box sx={{ height: "100%", overflow: "auto", p: 2 }}>
            <Stack spacing={3}>
                {/* Disk Information Card */}
                <Card>
                    <CardHeader
                        title="Disk Information"
                        avatar={
                            <IconButton onClick={() => openPreviewFor(disk, `Disk: ${disk.model || disk.serial || disk.id || "Unknown"}`)} aria-label="disk preview" size="small">
                                {renderDiskIcon(disk)}
                            </IconButton>
                        }
                        action={
                            <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                                {!diskInfoExpanded && (
                                    <Stack direction="row" spacing={1} sx={{ mr: 1 }}>
                                        <Typography variant="caption" color="text.secondary">
                                            {disk.model || "Unknown"}
                                        </Typography>
                                        {disk.size != null && (
                                            <Typography variant="caption" color="text.secondary">
                                                • {filesize(disk.size, { round: 1 })}
                                            </Typography>
                                        )}
                                        {disk.connection_bus && (
                                            <Typography variant="caption" color="text.secondary">
                                                • {disk.connection_bus}
                                            </Typography>
                                        )}
                                    </Stack>
                                )}
                                <IconButton
                                    onClick={() => setDiskInfoExpanded(!diskInfoExpanded)}
                                    aria-expanded={diskInfoExpanded}
                                    aria-label="show more"
                                    sx={{
                                        transform: diskInfoExpanded ? "rotate(180deg)" : "rotate(0deg)",
                                        transition: "transform 150ms cubic-bezier(0.4, 0, 0.2, 1)",
                                    }}
                                >
                                    <ExpandMoreIcon />
                                </IconButton>
                            </Box>
                        }
                    />
                    <Collapse in={diskInfoExpanded} timeout="auto" unmountOnExit>
                        <CardContent>
                            <Grid container spacing={2}>
                                <Grid size={{ xs: 12, sm: 6 }}>
                                    <Typography variant="subtitle2" color="text.secondary">
                                        Model
                                    </Typography>
                                    <Typography variant="body2">
                                        {disk.model || "Unknown"}
                                    </Typography>
                                </Grid>
                                <Grid size={{ xs: 12, sm: 6 }}>
                                    <Typography variant="subtitle2" color="text.secondary">
                                        Vendor
                                    </Typography>
                                    <Typography variant="body2">
                                        {disk.vendor || "N/A"}
                                    </Typography>
                                </Grid>
                                {disk.size != null && (
                                    <Grid size={{ xs: 12, sm: 6 }}>
                                        <Typography variant="subtitle2" color="text.secondary">
                                            Size
                                        </Typography>
                                        <Typography variant="body2">
                                            {filesize(disk.size, { round: 1 })}
                                        </Typography>
                                    </Grid>
                                )}
                                <Grid size={{ xs: 12, sm: 6 }}>
                                    <Typography variant="subtitle2" color="text.secondary">
                                        Connection
                                    </Typography>
                                    <Typography variant="body2">
                                        {disk.connection_bus || "N/A"}
                                    </Typography>
                                </Grid>
                                {disk.serial && (
                                    <Grid size={{ xs: 12, sm: 6 }}>
                                        <Typography variant="subtitle2" color="text.secondary">
                                            Serial
                                        </Typography>
                                        <Typography variant="body2" sx={{ fontFamily: "monospace" }}>
                                            {disk.serial}
                                        </Typography>
                                    </Grid>
                                )}
                                {disk.revision && (
                                    <Grid size={{ xs: 12, sm: 6 }}>
                                        <Typography variant="subtitle2" color="text.secondary">
                                            Revision
                                        </Typography>
                                        <Typography variant="body2">
                                            {disk.revision}
                                        </Typography>
                                    </Grid>
                                )}
                                <Grid size={{ xs: 12 }}>
                                    <Typography variant="subtitle2" color="text.secondary">
                                        Properties
                                    </Typography>
                                    <Stack direction="row" spacing={1} flexWrap="wrap" sx={{ gap: 1, mt: 0.5 }}>
                                        {disk.removable && (
                                            <Chip label="Removable" size="small" variant="outlined" />
                                        )}
                                        <Chip
                                            label={`${disk.partitions?.length || 0} Partition(s)`}
                                            size="small"
                                            variant="outlined"
                                        />
                                    </Stack>
                                </Grid>
                            </Grid>
                        </CardContent>
                    </Collapse>
                </Card>

                {/* Partition Information Card */}
                <Card>
                    <CardHeader
                        title="Partition Information"
                        avatar={
                            <IconButton onClick={() => openPreviewFor(partition, `Partition: ${decodeEscapeSequence(partition.name || partition.id || "Unnamed")}`)} aria-label="partition preview" size="small">
                                <StorageIcon color="primary" />
                            </IconButton>
                        }
                    />
                    <CardContent>
                        <Grid container spacing={2}>
                            <Grid size={{ xs: 12 }}>
                                <Typography variant="subtitle2" color="text.secondary">
                                    Name
                                </Typography>
                                <Typography variant="h6">
                                    {decodeEscapeSequence(partition.name || partition.id || "Unnamed Partition")}
                                </Typography>
                            </Grid>
                            {partition.size != null && (
                                <Grid size={{ xs: 12, sm: 6 }}>
                                    <Typography variant="subtitle2" color="text.secondary">
                                        Size
                                    </Typography>
                                    <Typography variant="body2">
                                        {filesize(partition.size, { round: 1 })}
                                    </Typography>
                                </Grid>
                            )}
                            {(mountData?.fstype || partition.fs_type) && (
                                <Grid size={{ xs: 12, sm: 6 }}>
                                    <Typography variant="subtitle2" color="text.secondary">
                                        File System
                                    </Typography>
                                    <Typography variant="body2">
                                        {mountData?.fstype ?? partition.fs_type}
                                    </Typography>
                                </Grid>
                            )}
                            {partition.id && (
                                <Grid size={{ xs: 12, sm: 6 }}>
                                    <Typography variant="subtitle2" color="text.secondary">
                                        Partition ID
                                    </Typography>
                                    <Typography variant="body2" sx={{ fontFamily: "monospace" }}>
                                        {partition.id}
                                    </Typography>
                                </Grid>
                            )}
                            {partition.legacy_device_name && (
                                <Grid size={{ xs: 12, sm: 6 }}>
                                    <Typography variant="subtitle2" color="text.secondary">
                                        Device
                                    </Typography>
                                    <Typography variant="body2" sx={{ fontFamily: "monospace" }}>
                                        {partition.legacy_device_name}
                                    </Typography>
                                </Grid>
                            )}

                            {/* Mount Status */}
                            <Grid size={{ xs: 12 }}>
                                <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 1 }}>
                                    Status
                                </Typography>
                                <Stack direction="row" spacing={1} flexWrap="wrap" sx={{ gap: 1 }}>
                                    <Chip
                                        label={isMounted ? "Mounted" : "Not Mounted"}
                                        color={isMounted ? "success" : "default"}
                                        size="small"
                                    />
                                    {partition.system && (
                                        <Chip label="System" size="small" variant="outlined" />
                                    )}
                                    {mountData?.is_to_mount_at_startup && (
                                        <Chip label="Auto-mount" size="small" variant="outlined" color="primary" />
                                    )}
                                    {mountData && !mountData.is_write_supported && (
                                        <Chip label="Read-Only" size="small" variant="outlined" color="secondary" />
                                    )}
                                </Stack>
                            </Grid>

                            {/* Mount Information */}
                            {isMounted && mountData && (
                                <>
                                    {/* Host Mount Information */}
                                    {partition.mount_point_data && partition.mount_point_data.length > 0 && (
                                        <Grid size={{ xs: 12, sm: 6 }}>
                                            <Typography variant="subtitle2" color="text.secondary">
                                                Mount Point{partition.mount_point_data.length > 1 ? "s" : ""}
                                            </Typography>
                                            <Stack direction="row" spacing={1} flexWrap="wrap" sx={{ gap: 1, mt: 0.5 }}>
                                                {partition.mount_point_data.map((mpd, index) => (
                                                    <Chip
                                                        key={index}
                                                        label={mpd.path}
                                                        size="small"
                                                        variant="outlined"
                                                        sx={{ fontFamily: "monospace" }}
                                                    />
                                                ))}
                                            </Stack>
                                        </Grid>
                                    )}
                                    {mountData.disk_label && (
                                        <Grid size={{ xs: 12, sm: 6 }}>
                                            <Typography variant="subtitle2" color="text.secondary">
                                                Disk Label
                                            </Typography>
                                            <Typography variant="body2">
                                                {mountData.disk_label}
                                            </Typography>
                                        </Grid>
                                    )}
                                    {mountData.time_machine_support && (
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
                                    {/* Warnings and Errors */}
                                    {mountData.warnings && (
                                        <Grid size={{ xs: 12 }}>
                                            <Typography variant="subtitle2" color="warning.main">
                                                Warnings
                                            </Typography>
                                            <Typography variant="body2" color="warning.main">
                                                {mountData.warnings}
                                            </Typography>
                                        </Grid>
                                    )}
                                    {mountData.invalid && mountData.invalid_error && (
                                        <Grid size={{ xs: 12 }}>
                                            <Typography variant="subtitle2" color="error.main">
                                                Error
                                            </Typography>
                                            <Typography variant="body2" color="error.main">
                                                {mountData.invalid_error}
                                            </Typography>
                                        </Grid>
                                    )}
                                </>
                            )}

                            {/* Host Mount Information */}
                            {partition.host_mount_point_data && partition.host_mount_point_data.length > 0 && (
                                <Grid size={{ xs: 12 }}>
                                    <Typography variant="subtitle2" color="text.secondary">
                                        Host Mount Point{partition.host_mount_point_data.length > 1 ? "s" : ""}
                                    </Typography>
                                    <Stack direction="row" spacing={1} flexWrap="wrap" sx={{ gap: 1, mt: 0.5 }}>
                                        {partition.host_mount_point_data.map((hmpd, index) => (
                                            <Chip
                                                key={index}
                                                label={hmpd.path}
                                                size="small"
                                                variant="outlined"
                                                sx={{ fontFamily: "monospace" }}
                                            />
                                        ))}
                                    </Stack>
                                </Grid>
                            )}
                        </Grid>
                    </CardContent>
                </Card>

                {/* Mount Settings Card */}
                {isMounted && mountData && partition.mount_point_data?.length == 1 && (
                    <Card>
                        <CardHeader
                            title="Mount Settings"
                            avatar={
                                <IconButton onClick={() => openPreviewFor(mountData, `Mount Settings: ${mountData.path || ""}`)} aria-label="mount settings preview" size="small">
                                    <SettingsIcon color="primary" />
                                </IconButton>
                            }
                        />
                        <CardContent>
                            <Grid container spacing={2}>
                                {/* File System Type */}
                                {(mountData.fstype || partition.fs_type) && (
                                    <Grid size={{ xs: 12, sm: 6 }}>
                                        <Typography variant="subtitle2" color="text.secondary">
                                            File System Type
                                        </Typography>
                                        <Typography variant="body2">
                                            {mountData.fstype ?? partition.fs_type}
                                        </Typography>
                                    </Grid>
                                )}

                                {/* Mount at Startup */}
                                <Grid size={{ xs: 12, sm: 6 }}>
                                    <Typography variant="subtitle2" color="text.secondary">
                                        Mount at Startup
                                    </Typography>
                                    <Chip
                                        label={mountData.is_to_mount_at_startup ? "Enabled" : "Disabled"}
                                        color={mountData.is_to_mount_at_startup ? "success" : "default"}
                                        size="small"
                                    />
                                </Grid>

                                {/* Mount Flags */}
                                {mountData.flags && mountData.flags.length > 0 && (
                                    <Grid size={{ xs: 12 }}>
                                        <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 1 }}>
                                            Mount Flags
                                        </Typography>
                                        <Stack direction="row" spacing={1} flexWrap="wrap" sx={{ gap: 1 }}>
                                            {mountData.flags.map((flag, index) => (
                                                <Chip
                                                    key={index}
                                                    label={flag.value ? `${flag.name}=${flag.value}` : flag.name}
                                                    size="small"
                                                    variant="outlined"
                                                    color="primary"
                                                />
                                            ))}
                                        </Stack>
                                    </Grid>
                                )}

                                {/* Custom/Filesystem-specific Mount Flags */}
                                {mountData.custom_flags && mountData.custom_flags.length > 0 && (
                                    <Grid size={{ xs: 12 }}>
                                        <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 1 }}>
                                            Filesystem-specific Mount Flags
                                        </Typography>
                                        <Stack direction="row" spacing={1} flexWrap="wrap" sx={{ gap: 1 }}>
                                            {mountData.custom_flags.map((flag, index) => (
                                                <Chip
                                                    key={index}
                                                    label={flag.value ? `${flag.name}=${flag.value}` : flag.name}
                                                    size="small"
                                                    variant="outlined"
                                                    color="secondary"
                                                />
                                            ))}
                                        </Stack>
                                    </Grid>
                                )}

                                {/* Write Support Status */}
                                <Grid size={{ xs: 12, sm: 6 }}>
                                    <Typography variant="subtitle2" color="text.secondary">
                                        Write Support
                                    </Typography>
                                    <Chip
                                        label={mountData.is_write_supported ? "Read/Write" : "Read-Only"}
                                        color={mountData.is_write_supported ? "success" : "warning"}
                                        size="small"
                                    />
                                </Grid>
                            </Grid>
                        </CardContent>
                    </Card>
                )}

                {/* Share Information Card */}
                {mountData?.shares && mountData.shares.length > 0 ? (
                    <Card>
                        <CardHeader
                            title={`Related Share${allShares?.length === 1 ? "" : "s"} (${allShares?.length})`}
                            avatar={
                                <IconButton onClick={() => openPreviewFor(allShares, `Related Shares (${allShares.length})`)} aria-label="shares preview" size="small">
                                    <FolderSpecialIcon color="primary" />
                                </IconButton>
                            }
                        />
                        <CardContent>
                            <Grid container spacing={2}>
                                {partition.mount_point_data?.flatMap((mpd) => mpd.shares).filter(Boolean).map((share, index) => (
                                    <Grid size={{ xs: 12, sm: 6, md: 4 }} key={index}>
                                        <Card variant="outlined" sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
                                            <CardHeader
                                                title={share?.name || "Unnamed Share"}
                                                avatar={
                                                    <IconButton onClick={() => openPreviewFor(share, `Share: ${share?.name}`)} aria-label="share preview" size="small">
                                                        <FolderSpecialIcon color="primary" />
                                                    </IconButton>
                                                }
                                                action={
                                                    <IconButton onClick={() => share && navigateToShare(share)} size="small" aria-label={`go to share ${share?.name}`}>
                                                        <VisibilityIcon />
                                                    </IconButton>
                                                }
                                            />
                                            <CardContent sx={{ flex: 1 }}>
                                                <Stack spacing={2}>
                                                    {/* Share Properties */}
                                                    <Stack direction="row" spacing={1} flexWrap="wrap" sx={{ gap: 1 }}>
                                                        {share?.usage && share?.usage !== Usage.Internal && (
                                                            <Chip
                                                                icon={<FolderSpecialIcon />}
                                                                label={`Usage: ${share.usage}`}
                                                                variant="outlined"
                                                                color="primary"
                                                                size="small"
                                                            />
                                                        )}
                                                        {share?.timemachine && (
                                                            <Chip
                                                                icon={<BackupIcon />}
                                                                label="Time Machine"
                                                                variant="outlined"
                                                                color="secondary"
                                                                size="small"
                                                            />
                                                        )}
                                                        {share?.recycle_bin_enabled && (
                                                            <Chip
                                                                label="Recycle Bin"
                                                                variant="outlined"
                                                                color="info"
                                                                size="small"
                                                            />
                                                        )}
                                                        {share?.guest_ok && (
                                                            <Chip
                                                                label="Guest Access"
                                                                variant="outlined"
                                                                color="warning"
                                                                size="small"
                                                            />
                                                        )}
                                                        {share?.disabled && (
                                                            <Chip
                                                                label="Disabled"
                                                                variant="outlined"
                                                                color="error"
                                                                size="small"
                                                            />
                                                        )}
                                                    </Stack>

                                                    {/* Users */}
                                                    {share?.users && share?.users.length > 0 && (
                                                        <Box>
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
                                                        </Box>
                                                    )}

                                                    {/* Read-Only Users */}
                                                    {share?.ro_users && share?.ro_users.length > 0 && (
                                                        <Box>
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
                                                        </Box>
                                                    )}
                                                </Stack>
                                            </CardContent>
                                        </Card>
                                        {index < (mountData.shares?.length || 0) - 1 && <Box sx={{ my: 2 }} />}
                                    </Grid>
                                ))}
                            </Grid>
                        </CardContent>
                    </Card>
                ) : isMounted && mountData?.path?.startsWith("/mnt/") ? (
                    <Card>
                        <CardHeader
                            title="Share Information"
                            avatar={<FolderSpecialIcon color="disabled" />}
                        />
                        <CardContent>
                            <Box
                                sx={{
                                    display: "flex",
                                    alignItems: "center",
                                    justifyContent: "center",
                                    py: 4,
                                    color: "text.secondary",
                                }}
                            >
                                <Typography variant="body2">
                                    No shares configured for this partition
                                </Typography>
                            </Box>
                        </CardContent>
                    </Card>
                ) : null}
            </Stack>

            {/* Preview dialog for disk object */}
            <PreviewDialog
                open={previewOpen}
                onClose={closePreview}
                title={previewTitle}
                objectToDisplay={previewObject}
            />
        </Box>
    );
}