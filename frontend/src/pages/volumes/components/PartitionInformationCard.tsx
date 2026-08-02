import CheckCircleOutlinedIcon from "@mui/icons-material/CheckCircleOutlined";
import ErrorOutlinedIcon from "@mui/icons-material/ErrorOutlined";
import HelpOutlinedIcon from "@mui/icons-material/HelpOutlined";
import StorageIcon from "@mui/icons-material/Storage";
import {
  Box,
  Button,
  Card,
  CardContent,
  CardHeader,
  Chip,
  Grid,
  IconButton,
  Stack,
  Tooltip,
  Typography,
} from "@mui/material";
import { filesize } from "filesize";
import { useMemo } from "react";
import {
  type FilesystemState,
  type Partition,
  Time_machine_support,
  useGetApiFilesystemStateQuery,
} from "../../../store/sratApi";
import { usePartitionActions } from "../hooks/usePartitionActions";
import { decodeEscapeSequence } from "../utils";

interface PartitionInformationCardProps {
  partition: Partition;
  title?: string;
  protectedMode?: boolean;
  readOnly?: boolean;
  onToggleAutomount?: (partition: Partition) => void;
  onMount?: (partition: Partition) => void;
  onUnmount?: (partition: Partition, force: boolean) => void;
  onCreateShare?: (partition: Partition) => void;
  onGoToShare?: (partition: Partition) => void;
  onCheckFilesystem?: () => void;
  onSetFilesystemLabel?: () => void;
  onFormatPartition?: () => void;
  onPreview: (obj: object, title?: string) => void;
}

export function PartitionInformationCard({
  partition,
  title = "Partition Information",
  protectedMode = false,
  readOnly = false,
  onToggleAutomount,
  onMount,
  onUnmount,
  onCreateShare,
  onGoToShare,
  onCheckFilesystem,
  onSetFilesystemLabel,
  onFormatPartition,
  onPreview,
}: PartitionInformationCardProps) {
  const mpds = Object.values(partition.mount_point_data || {});
  const isMounted = mpds.some((mpd) => mpd.is_mounted);
  const partitionId = partition.id;
  const {
    data: filesystemStateResponse,
    currentData: filesystemStateCurrentData,
    isLoading: filesystemStateLoading,
  } = useGetApiFilesystemStateQuery({ partitionId }, { skip: !partitionId });

  const filesystemStatePayload =
    filesystemStateResponse ?? filesystemStateCurrentData;

  const filesystemState = useMemo<FilesystemState | null>(() => {
    if (!filesystemStatePayload) {
      return null;
    }
    if ("hasErrors" in filesystemStatePayload) {
      return filesystemStatePayload;
    }
    return null;
  }, [filesystemStatePayload]);

  const filesystemStatus = useMemo(() => {
    if (!filesystemState) {
      return "no_status" as const;
    }
    if (filesystemState.hasErrors) {
      return "has_error" as const;
    }
    if (filesystemState.isClean) {
      return "clean" as const;
    }
    return "no_status" as const;
  }, [filesystemState]);

  const filesystemStatusIcon = useMemo(() => {
    if (filesystemStatus === "clean") {
      return <CheckCircleOutlinedIcon color="success" fontSize="small" />;
    }
    if (filesystemStatus === "has_error") {
      return <ErrorOutlinedIcon color="error" fontSize="small" />;
    }
    return <HelpOutlinedIcon color="disabled" fontSize="small" />;
  }, [filesystemStatus]);

  const filesystemStatusTooltip = useMemo(() => {
    if (filesystemStateLoading) {
      return "Loading filesystem status...";
    }
    if (!filesystemState) {
      return "No filesystem status available";
    }
    const description = filesystemState.stateDescription || "Filesystem status";
    const additionalInfoEntries = Object.entries(
      filesystemState.additionalInfo || {},
    );
    if (additionalInfoEntries.length === 0) {
      return description;
    }
    return (
      <Box>
        <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
          {description}
        </Typography>
        {additionalInfoEntries.map(([key, value]) => (
          <Typography key={key} variant="body2">
            {key}:{" "}
            {(typeof value === "string" ? value : JSON.stringify(value))
              .split("\n")
              .map((line, index) => (
                // biome-ignore lint/suspicious/noArrayIndexKey: text lines have no unique id
                <span key={index}>
                  {line}
                  <br />
                </span>
              ))}
          </Typography>
        ))}
      </Box>
    );
  }, [filesystemState, filesystemStateLoading]);

  const partitionActionItems = usePartitionActions({
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

  const readOnlyActionTooltip = "Read-only mode enabled. Actions are disabled.";

  return (
    <Card>
      <CardHeader
        title={title}
        avatar={
          <IconButton
            onClick={() =>
              onPreview(
                partition,
                `${title}: ${decodeEscapeSequence(partition.name || partition.id || "Unnamed")}`,
              )
            }
            aria-label="partition preview"
            size="small"
          >
            <StorageIcon color="primary" />
          </IconButton>
        }
      />
      <CardContent>
        <Grid container spacing={2}>
          <Grid size={{ xs: 12 }}>
            <Typography
              variant="subtitle2"
              sx={{
                color: "text.secondary",
              }}
            >
              Name
            </Typography>
            <Typography variant="h6">
              {decodeEscapeSequence(
                partition.name || partition.id || "Unnamed Partition",
              )}
            </Typography>
          </Grid>
          {partition.size != null && (
            <Grid size={{ xs: 12, sm: 6, md: 4 }}>
              <Typography
                variant="subtitle2"
                sx={{
                  color: "text.secondary",
                }}
              >
                Size
              </Typography>
              <Typography variant="body2">
                {filesize(partition.size, { round: 1 })}
              </Typography>
            </Grid>
          )}
          {partition.fs_type && (
            <>
              <Grid size={{ xs: 12, sm: 6, md: 4 }}>
                <Typography
                  variant="subtitle2"
                  sx={{
                    color: "text.secondary",
                  }}
                >
                  File System
                </Typography>
                <Typography variant="body2">{partition.fs_type}</Typography>
              </Grid>

              {/* Filesystem Status Information */}
              <Grid size={{ xs: 12 }}>
                <Typography
                  variant="subtitle2"
                  sx={{
                    color: "text.secondary",
                    mb: 1,
                  }}
                >
                  Filesystem Status
                </Typography>
                <Stack
                  direction="row"
                  spacing={1}
                  sx={{
                    flexWrap: "wrap",
                    gap: 1,
                  }}
                >
                  {isMounted && (
                    <Chip
                      label="Mounted & Accessible"
                      color="success"
                      size="small"
                      icon={<StorageIcon />}
                    />
                  )}
                  {!isMounted && (
                    <Chip label="Not Mounted" color="default" size="small" />
                  )}
                  {partition.filesystem_info && (
                    <Tooltip title={filesystemStatusTooltip} arrow>
                      <Chip
                        label={
                          partition.filesystem_info.description ||
                          "Filesystem Info"
                        }
                        variant="outlined"
                        size="small"
                        icon={filesystemStatusIcon}
                      />
                    </Tooltip>
                  )}
                </Stack>
              </Grid>
            </>
          )}
          {partition.legacy_device_name && (
            <Grid size={{ xs: 12, sm: 6, md: 4 }}>
              <Typography
                variant="subtitle2"
                sx={{
                  color: "text.secondary",
                }}
              >
                Device
              </Typography>
              <Typography variant="body2" sx={{ fontFamily: "monospace" }}>
                {partition.legacy_device_name}
              </Typography>
            </Grid>
          )}
          {partition.id && (
            <Grid size={{ xs: 12 }}>
              <Typography
                variant="subtitle2"
                sx={{
                  color: "text.secondary",
                }}
              >
                Partition ID
              </Typography>
              <Typography variant="body2" sx={{ fontFamily: "monospace" }}>
                {partition.id}
              </Typography>
            </Grid>
          )}

          {/* Mount Status */}
          <Grid size={{ xs: 12 }}>
            <Typography
              variant="subtitle2"
              sx={{
                color: "text.secondary",
                mb: 1,
              }}
            >
              Status
            </Typography>
            <Stack
              direction="row"
              spacing={1}
              sx={{
                flexWrap: "wrap",
                gap: 1,
              }}
            >
              <Chip
                label={isMounted ? "Mounted" : "Not Mounted"}
                color={isMounted ? "success" : "default"}
                size="small"
              />
              {partition.system && (
                <Chip label="System" size="small" variant="outlined" />
              )}
            </Stack>
          </Grid>

          {/* Mount Information */}
          {isMounted && (
            <>
              {mpds.some((mpd) => mpd.disk_label) && (
                <Grid size={{ xs: 12, sm: 6 }}>
                  <Typography
                    variant="subtitle2"
                    sx={{
                      color: "text.secondary",
                    }}
                  >
                    Disk Label
                  </Typography>
                  <Typography variant="body2">
                    {mpds.find((mpd) => mpd.disk_label)?.disk_label}
                  </Typography>
                </Grid>
              )}
              {mpds.some((mpd) => mpd.time_machine_support) && (
                <Grid size={{ xs: 12, sm: 6 }}>
                  <Typography
                    variant="subtitle2"
                    sx={{
                      color: "text.secondary",
                    }}
                  >
                    Time Machine Support
                  </Typography>
                  <Chip
                    label={
                      mpds.find((mpd) => mpd.time_machine_support)
                        ?.time_machine_support
                    }
                    color={
                      mpds.find((mpd) => mpd.time_machine_support)
                        ?.time_machine_support ===
                      Time_machine_support.Supported
                        ? "success"
                        : mpds.find((mpd) => mpd.time_machine_support)
                              ?.time_machine_support ===
                            Time_machine_support.Experimental
                          ? "warning"
                          : "error"
                    }
                    size="small"
                  />
                </Grid>
              )}
              {mpds.some((mpd) => mpd.warnings) && (
                <Grid size={{ xs: 12 }}>
                  <Typography
                    variant="subtitle2"
                    sx={{
                      color: "warning.main",
                    }}
                  >
                    Warnings
                  </Typography>
                  {mpds
                    .filter((mpd) => mpd.warnings)
                    ?.map((mpd) => (
                      <Typography
                        key={mpd.path}
                        variant="body2"
                        sx={{
                          color: "warning.main",
                        }}
                      >
                        {mpd.warnings}
                      </Typography>
                    ))}
                </Grid>
              )}
              {mpds.some((mpd) => mpd.invalid && mpd.invalid_error) && (
                <Grid size={{ xs: 12 }}>
                  <Typography
                    variant="subtitle2"
                    sx={{
                      color: "error.main",
                    }}
                  >
                    Errors
                  </Typography>
                  <Typography
                    variant="body2"
                    sx={{
                      color: "error.main",
                    }}
                  >
                    {
                      mpds.find((mpd) => mpd.invalid && mpd.invalid_error)
                        ?.invalid_error
                    }
                  </Typography>
                </Grid>
              )}
              {/* Host Mount Information */}
              {mpds.length > 0 && (
                <Grid size={{ xs: 12 }}>
                  <Typography
                    variant="subtitle2"
                    sx={{
                      color: "text.secondary",
                    }}
                  >
                    Mount Point{mpds.length > 1 ? "s" : ""}
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
                    {mpds.map((mpd) => {
                      const badges: string[] = [];
                      if (mpd?.is_to_mount_at_startup) {
                        badges.push("Auto-mount");
                      }
                      if (!mpd.is_write_supported) {
                        badges.push("Read-Only");
                      }
                      const label =
                        badges.length > 0
                          ? `${mpd.path} • ${badges.join(" • ")}`
                          : mpd.path;

                      return (
                        <Chip
                          key={mpd.path}
                          label={label}
                          size="small"
                          variant="outlined"
                          color={
                            !mpd.is_write_supported
                              ? "secondary"
                              : mpd?.is_to_mount_at_startup
                                ? "primary"
                                : "default"
                          }
                          sx={{ fontFamily: "monospace" }}
                        />
                      );
                    })}
                  </Stack>
                </Grid>
              )}
            </>
          )}

          {/* Host Mount Information */}
          {Object.values(partition.host_mount_point_data || {}).length > 0 && (
            <Grid size={{ xs: 12 }}>
              <Typography
                variant="subtitle2"
                sx={{
                  color: "text.secondary",
                }}
              >
                Host Mount Point
                {Object.values(partition.host_mount_point_data || {}).length > 1
                  ? "s"
                  : ""}
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
                {Object.values(partition.host_mount_point_data || {}).map(
                  (hmpd) => (
                    <Chip
                      key={hmpd.path}
                      label={hmpd.path}
                      size="small"
                      variant="outlined"
                      sx={{ fontFamily: "monospace" }}
                    />
                  ),
                )}
              </Stack>
            </Grid>
          )}

          {partitionActionItems && partitionActionItems.length > 0 && (
            <Grid size={{ xs: 12 }}>
              <Typography
                variant="subtitle2"
                sx={{
                  color: "text.secondary",
                  mb: 1,
                }}
              >
                Actions
              </Typography>
              <Box
                sx={{
                  display: "grid",
                  gridTemplateColumns: {
                    xs: "minmax(0, 1fr)",
                    sm: "repeat(auto-fit, minmax(min(100%, 11rem), 1fr))",
                  },
                  gap: 1,
                  alignItems: "stretch",
                }}
              >
                {partitionActionItems.map((action) => {
                  const button = (
                    <Button
                      fullWidth
                      size="small"
                      variant="outlined"
                      onClick={action.onClick}
                      color={action.color || "primary"}
                      disabled={readOnly}
                      title={readOnly ? readOnlyActionTooltip : action.title}
                      sx={{
                        justifyContent: "center",
                        whiteSpace: "nowrap",
                      }}
                    >
                      {action.title}
                    </Button>
                  );

                  if (!readOnly) {
                    return (
                      <Box key={action.key} sx={{ minWidth: 0 }}>
                        {button}
                      </Box>
                    );
                  }

                  return (
                    <Box key={action.key} sx={{ minWidth: 0 }}>
                      <Tooltip title={readOnlyActionTooltip}>
                        <span style={{ display: "block", width: "100%" }}>
                          {button}
                        </span>
                      </Tooltip>
                    </Box>
                  );
                })}
              </Box>
            </Grid>
          )}
        </Grid>
      </CardContent>
    </Card>
  );
}
