/* eslint-disable */
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
  Stack,
  Typography,
} from "@mui/material";
import { filesize } from "filesize";
import { useMemo, useState } from "react";
import { PreviewDialog } from "../../../components/PreviewDialog";
import {
  type Disk,
  type Partition,
  type Settings,
  useGetApiSettingsQuery,
} from "../../../store/sratApi";
import { DiskIcon } from "../../../utils/diskIcon";
import { getRealPartitions, isSynthesizedWholeDiskPartition } from "../utils";
import { FilesystemCheckDialog } from "./FilesystemCheckDialog";
import { FilesystemFormatDialog } from "./FilesystemFormatDialog";
import { FilesystemLabelDialog } from "./FilesystemLabelDialog";
import { HDIdleDiskSettings } from "./HDIdleDiskSettings";
import { PartitionInformationCard } from "./PartitionInformationCard";
import { SmartStatusPanel } from "./SmartStatusPanel";

export interface DetailsActions {
  onToggleAutomount?: (partition: Partition) => void;
  onMount?: (partition: Partition) => void;
  onUnmount?: (partition: Partition, force: boolean) => void;
  onCreateShare?: (partition: Partition) => void;
  onGoToShare?: (partition: Partition) => void;
  onLabelUpdated?: (partitionId: string, label: string) => void;
  protectedMode?: boolean;
  readOnly?: boolean;
}

interface VolumeDetailsPanelProps {
  disk?: Disk;
  partition?: Partition;
  actions?: DetailsActions;
  // Legacy flat props (deprecated): kept so existing tests and older callers
  // keep working. New code must use disk/partition/actions.
  protectedMode?: boolean;
  readOnly?: boolean;
  onToggleAutomount?: (partition: Partition) => void;
  onMount?: (partition: Partition) => void;
  onUnmount?: (partition: Partition, force: boolean) => void;
  onCreateShare?: (partition: Partition) => void;
  onGoToShare?: (partition: Partition) => void;
  onLabelUpdated?: (partitionId: string, label: string) => void;
  // share?: SharedResource;
}

export function VolumeDetailsPanel(props: VolumeDetailsPanelProps) {
  const {
    disk,
    partition,
    actions,
    protectedMode: legacyProtectedMode,
    readOnly: legacyReadOnly,
    onToggleAutomount: legacyOnToggleAutomount,
    onMount: legacyOnMount,
    onUnmount: legacyOnUnmount,
    onCreateShare: legacyOnCreateShare,
    onGoToShare: legacyOnGoToShare,
    onLabelUpdated: legacyOnLabelUpdated,
  } = props;
  const protectedMode = actions?.protectedMode ?? legacyProtectedMode ?? false;
  const readOnly = actions?.readOnly ?? legacyReadOnly ?? false;
  const onToggleAutomount =
    actions?.onToggleAutomount ?? legacyOnToggleAutomount;
  const onMount = actions?.onMount ?? legacyOnMount;
  const onUnmount = actions?.onUnmount ?? legacyOnUnmount;
  const onCreateShare = actions?.onCreateShare ?? legacyOnCreateShare;
  const onGoToShare = actions?.onGoToShare ?? legacyOnGoToShare;
  const onLabelUpdated = actions?.onLabelUpdated ?? legacyOnLabelUpdated;
  const [diskInfoExpanded, setDiskInfoExpanded] = useState(!partition);
  const [smartExpanded, setSmartExpanded] = useState(true);
  const [previewOpen, setPreviewOpen] = useState(false);
  const [previewObject, setPreviewObject] = useState<object | null>(null);
  const [previewTitle, setPreviewTitle] = useState<string>("Preview");
  const [checkDialogOpen, setCheckDialogOpen] = useState(false);
  const [formatDialogOpen, setFormatDialogOpen] = useState(false);
  const [labelDialogOpen, setLabelDialogOpen] = useState(false);

  const openDialog = (setDialogOpen: (open: boolean) => void) => {
    const activeElement = document.activeElement;
    if (activeElement instanceof HTMLElement) {
      activeElement.blur();
    }
    setDialogOpen(true);
  };

  // Fetch settings for disable_smart
  const { data: settings, isLoading: settingsLoading } =
    useGetApiSettingsQuery();

  const openPreviewFor = (obj: object, title?: string) => {
    setPreviewObject(obj);
    setPreviewTitle(title ?? "Preview");
    setPreviewOpen(true);
  };
  const closePreview = () => {
    setPreviewOpen(false);
    setPreviewObject(null);
  };

  const mpds = Object.values(partition?.mount_point_data || {});
  const mountData = mpds[0];
  const isMounted = mpds.some((mpd) => mpd.is_mounted);

  // Task 044: detect a whole-disk synthesized partition (a raw disk with a
  // filesystem written directly to it, with no partition table). The backend
  // reports it as a single partition whose device name equals the disk's.
  const wholeDiskPartition = useMemo(() => {
    if (!disk || partition) {
      return undefined;
    }
    const parts = Object.values(disk.partitions || {});
    if (parts.length !== 1) {
      return undefined;
    }
    const candidate = parts[0];
    if (!candidate || !isSynthesizedWholeDiskPartition(disk, candidate)) {
      return undefined;
    }
    return candidate;
  }, [disk, partition]);

  // When nothing is selected, show placeholder
  if (!disk && !partition) {
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

  return (
    <Box sx={{ height: "100%", overflow: "auto", p: 2 }}>
      <Stack spacing={3}>
        {/* Disk Information and disk-only panels */}
        {disk && (
          <Card>
            <CardHeader
              title="Disk Information"
              avatar={
                <IconButton
                  onClick={() =>
                    openPreviewFor(
                      disk,
                      `Disk: ${disk.model || disk.serial || disk.id || "Unknown"}`,
                    )
                  }
                  aria-label="disk preview"
                  size="small"
                >
                  <DiskIcon disk={disk} color="primary" />
                </IconButton>
              }
              action={
                <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                  {!diskInfoExpanded && (
                    <Stack direction="row" spacing={1} sx={{ mr: 1 }}>
                      <Typography
                        variant="caption"
                        sx={{
                          color: "text.secondary",
                        }}
                      >
                        {disk.model || "Unknown"}
                      </Typography>
                      {disk.size != null && (
                        <Typography
                          variant="caption"
                          sx={{
                            color: "text.secondary",
                          }}
                        >
                          • {filesize(disk.size, { round: 1 })}
                        </Typography>
                      )}
                      {disk.connection_bus && (
                        <Typography
                          variant="caption"
                          sx={{
                            color: "text.secondary",
                          }}
                        >
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
                      transform: diskInfoExpanded
                        ? "rotate(180deg)"
                        : "rotate(0deg)",
                      transition:
                        "transform 150ms cubic-bezier(0.4, 0, 0.2, 1)",
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
                    <Typography
                      variant="subtitle2"
                      sx={{
                        color: "text.secondary",
                      }}
                    >
                      Model
                    </Typography>
                    <Typography variant="body2">
                      {disk.model || "Unknown"}
                    </Typography>
                  </Grid>
                  <Grid size={{ xs: 12, sm: 6 }}>
                    <Typography
                      variant="subtitle2"
                      sx={{
                        color: "text.secondary",
                      }}
                    >
                      Vendor
                    </Typography>
                    <Typography variant="body2">
                      {disk.vendor || "N/A"}
                    </Typography>
                  </Grid>
                  {disk.serial && (
                    <Grid size={{ xs: 12, sm: 6 }}>
                      <Typography
                        variant="subtitle2"
                        sx={{
                          color: "text.secondary",
                        }}
                      >
                        Serial
                      </Typography>
                      <Typography variant="body2">{disk.serial}</Typography>
                    </Grid>
                  )}
                  <Grid size={{ xs: 12, sm: 6 }}>
                    <Typography
                      variant="subtitle2"
                      sx={{
                        color: "text.secondary",
                      }}
                    >
                      Disk Type
                    </Typography>
                    <Typography variant="body2">
                      {disk.is_rotational == null
                        ? "Unknown"
                        : disk.is_rotational
                          ? "HDD (Rotational)"
                          : "SSD (Solid State)"}
                    </Typography>
                  </Grid>
                  {disk.smart_info?.rotation_rate != null && (
                    <Grid size={{ xs: 12, sm: 6 }}>
                      <Typography
                        variant="subtitle2"
                        sx={{
                          color: "text.secondary",
                        }}
                      >
                        Rotation Rate
                      </Typography>
                      <Typography variant="body2">
                        {disk.smart_info.rotation_rate} RPM
                      </Typography>
                    </Grid>
                  )}
                  {disk.size != null && (
                    <Grid size={{ xs: 12, sm: 6 }}>
                      <Typography
                        variant="subtitle2"
                        sx={{
                          color: "text.secondary",
                        }}
                      >
                        Size
                      </Typography>
                      <Typography variant="body2">
                        {filesize(disk.size, { round: 1 })}
                      </Typography>
                    </Grid>
                  )}
                  <Grid size={{ xs: 12, sm: 6 }}>
                    <Typography
                      variant="subtitle2"
                      sx={{
                        color: "text.secondary",
                      }}
                    >
                      Connection
                    </Typography>
                    <Typography variant="body2">
                      {disk.connection_bus || "N/A"}
                    </Typography>
                  </Grid>
                  <Grid size={{ xs: 12 }}>
                    <Typography
                      variant="subtitle2"
                      sx={{
                        color: "text.secondary",
                      }}
                    >
                      Properties
                    </Typography>
                    <Stack
                      direction="row"
                      spacing={1}
                      sx={{
                        flexWrap: "wrap",
                        gap: 1,
                        mt: 0.5,
                      }}
                    >
                      {disk.removable && (
                        <Chip
                          label="Removable"
                          size="small"
                          variant="outlined"
                        />
                      )}
                      {/* Task 044: disks with no partition table (raw disks) */}
                      {getRealPartitions(disk).length === 0 ? (
                        <Chip
                          label="Raw disk (no partition table)"
                          size="small"
                          color="warning"
                          variant="outlined"
                        />
                      ) : (
                        <Chip
                          label={`${getRealPartitions(disk).length} Partition(s)`}
                          size="small"
                          variant="outlined"
                        />
                      )}
                    </Stack>
                  </Grid>
                </Grid>
              </CardContent>
            </Collapse>
          </Card>
        )}

        {/* Disk-only panels: visible only when a disk is selected without a partition */}
        {disk &&
          !partition &&
          disk.hdidle_device?.supported &&
          disk.is_rotational === true && (
            <HDIdleDiskSettings disk={disk} readOnly={readOnly} />
          )}
        {/* Only render SmartStatusPanel if settings are loaded and SMART is on */}
        {disk &&
          !partition &&
          !settingsLoading &&
          settings &&
          (settings as Settings).smart_on !== false &&
          disk.smart_info?.supported && (
            <SmartStatusPanel
              smartInfo={disk.smart_info}
              diskId={disk.id}
              bus={disk.connection_bus}
              isReadOnlyMode={readOnly}
              isExpanded={smartExpanded}
              onSetExpanded={setSmartExpanded}
            />
          )}
        {/* Partition Information Card (shown only when a partition is selected) */}
        {partition && (
          <PartitionInformationCard
            partition={partition}
            title="Partition Information"
            protectedMode={protectedMode}
            readOnly={readOnly}
            onToggleAutomount={onToggleAutomount}
            onMount={onMount}
            onUnmount={onUnmount}
            onCreateShare={onCreateShare}
            onGoToShare={onGoToShare}
            onCheckFilesystem={() => openDialog(setCheckDialogOpen)}
            onSetFilesystemLabel={() => openDialog(setLabelDialogOpen)}
            onFormatPartition={() => openDialog(setFormatDialogOpen)}
            onPreview={openPreviewFor}
          />
        )}

        {/* Raw Disk Filesystem Information Card (shown for a raw disk with a
            whole-disk filesystem, when no partition is selected) */}
        {wholeDiskPartition && !partition && (
          <PartitionInformationCard
            partition={wholeDiskPartition}
            title="Raw Disk Filesystem Information"
            protectedMode={protectedMode}
            readOnly={readOnly}
            onToggleAutomount={onToggleAutomount}
            onMount={onMount}
            onUnmount={onUnmount}
            onCreateShare={onCreateShare}
            onGoToShare={onGoToShare}
            onCheckFilesystem={() => openDialog(setCheckDialogOpen)}
            onSetFilesystemLabel={() => openDialog(setLabelDialogOpen)}
            onFormatPartition={() => openDialog(setFormatDialogOpen)}
            onPreview={openPreviewFor}
          />
        )}

        {/* Mount Settings Card */}
        {partition &&
          isMounted &&
          mountData &&
          Object.values(partition.mount_point_data || {}).length === 1 && (
            <Card>
              <CardHeader
                title="Mount Settings"
                avatar={
                  <IconButton
                    onClick={() =>
                      openPreviewFor(
                        mountData,
                        `Mount Settings: ${mountData.path || ""}`,
                      )
                    }
                    aria-label="mount settings preview"
                    size="small"
                  >
                    <SettingsIcon color="primary" />
                  </IconButton>
                }
              />
              <CardContent>
                <Grid container spacing={2}>
                  {/* File System Type */}
                  {(mountData.fstype || partition.fs_type) && (
                    <Grid size={{ xs: 12, sm: 6 }}>
                      <Typography
                        variant="subtitle2"
                        sx={{
                          color: "text.secondary",
                        }}
                      >
                        File System Type
                      </Typography>
                      <Typography variant="body2">
                        {mountData.fstype ?? partition.fs_type}
                      </Typography>
                    </Grid>
                  )}

                  {/* Automatic Mount */}
                  <Grid size={{ xs: 12, sm: 6 }}>
                    <Typography
                      variant="subtitle2"
                      sx={{
                        color: "text.secondary",
                      }}
                    >
                      Automatic Mount
                    </Typography>
                    <Chip
                      label={
                        mountData.is_to_mount_at_startup
                          ? "Enabled"
                          : "Disabled"
                      }
                      color={
                        mountData.is_to_mount_at_startup ? "success" : "default"
                      }
                      size="small"
                    />
                  </Grid>

                  {/* Mount Flags */}
                  {mountData.flags && mountData.flags.length > 0 && (
                    <Grid size={{ xs: 12 }}>
                      <Typography
                        variant="subtitle2"
                        sx={{
                          color: "text.secondary",
                          mb: 1,
                        }}
                      >
                        Mount Flags
                      </Typography>
                      <Stack
                        direction="row"
                        spacing={1}
                        sx={{
                          flexWrap: "wrap",
                          gap: 1,
                        }}
                      >
                        {mountData.flags.map((flag) => (
                          <Chip
                            key={flag.name}
                            label={
                              flag.value
                                ? `${flag.name}=${flag.value}`
                                : flag.name
                            }
                            size="small"
                            variant="outlined"
                            color="primary"
                          />
                        ))}
                      </Stack>
                    </Grid>
                  )}

                  {/* Custom/Filesystem-specific Mount Flags */}
                  {mountData.custom_flags &&
                    mountData.custom_flags.length > 0 && (
                      <Grid size={{ xs: 12 }}>
                        <Typography
                          variant="subtitle2"
                          sx={{
                            color: "text.secondary",
                            mb: 1,
                          }}
                        >
                          Filesystem-specific Mount Flags
                        </Typography>
                        <Stack
                          direction="row"
                          spacing={1}
                          sx={{
                            flexWrap: "wrap",
                            gap: 1,
                          }}
                        >
                          {mountData.custom_flags.map((flag) => (
                            <Chip
                              key={flag.name}
                              label={
                                flag.value
                                  ? `${flag.name}=${flag.value}`
                                  : flag.name
                              }
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
                    <Typography
                      variant="subtitle2"
                      sx={{
                        color: "text.secondary",
                      }}
                    >
                      Write Support
                    </Typography>
                    <Chip
                      label={
                        mountData.is_write_supported
                          ? "Read/Write"
                          : "Read-Only"
                      }
                      color={
                        mountData.is_write_supported ? "success" : "warning"
                      }
                      size="small"
                    />
                  </Grid>
                </Grid>
              </CardContent>
            </Card>
          )}

        {/* Preview Button for Partition or Disk */}
      </Stack>
      {/* Preview dialog for disk object */}
      <PreviewDialog
        open={previewOpen}
        onClose={closePreview}
        title={previewTitle}
        objectToDisplay={previewObject}
      />
      <FilesystemCheckDialog
        open={checkDialogOpen}
        partition={partition ?? wholeDiskPartition}
        onClose={() => setCheckDialogOpen(false)}
      />
      <FilesystemLabelDialog
        open={labelDialogOpen}
        partition={partition ?? wholeDiskPartition}
        onClose={() => setLabelDialogOpen(false)}
        onLabelUpdated={onLabelUpdated}
      />
      <FilesystemFormatDialog
        open={formatDialogOpen}
        partition={partition ?? wholeDiskPartition}
        onClose={() => setFormatDialogOpen(false)}
      />
    </Box>
  );
}
