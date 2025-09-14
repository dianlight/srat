import BackupIcon from "@mui/icons-material/Backup";
import EditIcon from "@mui/icons-material/Edit";
import VisibilityIcon from "@mui/icons-material/Visibility";
import FolderSpecialIcon from "@mui/icons-material/FolderSpecial";
import StorageIcon from "@mui/icons-material/Storage";
import ComputerIcon from "@mui/icons-material/Computer";
import UsbIcon from "@mui/icons-material/Usb";
import SdStorageIcon from "@mui/icons-material/SdStorage";
import EjectIcon from "@mui/icons-material/Eject";
import {
    Box,
    Card,
    CardContent,
    CardHeader,
    Chip,
    Grid,
    Stack,
    Typography,
} from "@mui/material";
import { filesize } from "filesize";
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
    const isMounted = mountData?.is_mounted;

    return (
        <Box sx={{ height: "100%", overflow: "auto", p: 2 }}>
            <Stack spacing={3}>
                {/* Disk Information Card */}
                <Card>
                    <CardHeader
                        title="Disk Information"
                        avatar={renderDiskIcon(disk)}
                    />
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
                </Card>

                {/* Partition Information Card */}
                <Card>
                    <CardHeader
                        title="Partition Information"
                        avatar={<StorageIcon color="primary" />}
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
                            {mountData?.fstype && (
                                <Grid size={{ xs: 12, sm: 6 }}>
                                    <Typography variant="subtitle2" color="text.secondary">
                                        File System
                                    </Typography>
                                    <Typography variant="body2">
                                        {mountData.fstype}
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
                                    <Grid size={{ xs: 12, sm: 6 }}>
                                        <Typography variant="subtitle2" color="text.secondary">
                                            Mount Path
                                        </Typography>
                                        <Typography variant="body2" sx={{ fontFamily: "monospace" }}>
                                            {mountData.path || "N/A"}
                                        </Typography>
                                    </Grid>
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
                                        Host Mount Points
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

                {/* Share Information Card */}
                {mountData?.shares && mountData.shares.length > 0 ? (
                    <Card>
                        <CardHeader
                            title="Related Share"
                            avatar={<FolderSpecialIcon color="primary" />}
                        />
                        <CardContent>
                            {mountData.shares.map((share, index) => (
                                <Box key={index}>
                                    <Grid container spacing={2}>
                                        <Grid size={{ xs: 12 }}>
                                            <Typography variant="subtitle2" color="text.secondary">
                                                Share Name
                                            </Typography>
                                            <Typography variant="h6">
                                                {share.name}
                                            </Typography>
                                        </Grid>

                                        {/* Share Properties */}
                                        <Grid size={{ xs: 12 }}>
                                            <Stack direction="row" spacing={1} flexWrap="wrap" sx={{ gap: 1 }}>
                                                {share.usage && share.usage !== Usage.Internal && (
                                                    <Chip
                                                        icon={<FolderSpecialIcon />}
                                                        label={`Usage: ${share.usage}`}
                                                        variant="outlined"
                                                        color="primary"
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
                                            <Grid size={{ xs: 12 }}>
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
                                            <Grid size={{ xs: 12 }}>
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
                                    </Grid>
                                    {index < (mountData.shares?.length || 0) - 1 && <Box sx={{ my: 2 }} />}
                                </Box>
                            ))}
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
        </Box>
    );
}