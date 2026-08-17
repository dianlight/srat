import {
  Box,
  FormControlLabel,
  Paper,
  Stack,
  Switch,
  Typography,
} from "@mui/material";
import { useConfirm } from "material-ui-confirm";
import {
  type MouseEvent as ReactMouseEvent,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useLocation, useNavigate } from "react-router";
import { toast } from "react-toastify";
import { PreviewDialog } from "../../components/PreviewDialog";
import { useVolume } from "../../hooks/volumeHook";
import { type LocationState, TabIDs } from "../../store/locationState";
import {
  type Disk,
  type FilesystemState,
  type MountPointData,
  type Partition,
  type PerPartitionInfo,
  sratApi,
  useDeleteApiVolumeMutation,
  usePatchApiVolumeSettingsMutation,
  usePostApiVolumeMountMutation,
} from "../../store/sratApi";
import { useAppDispatch } from "../../store/store";
import { useGetServerEventsQuery } from "../../store/wsApi";
import { TourEvents, TourEventTypes } from "../../utils/TourEvents";
import {
  FilesystemCheckDialog,
  FilesystemFormatDialog,
  FilesystemLabelDialog,
  VolumeDetailsPanel,
  VolumeMountDialog,
  VolumesTreeView,
} from "./components";
import { getTourVolumeSelection } from "./tourSelection";
import {
  decodeEscapeSequence,
  getDiskIdentifier,
  getPartitionIdentifier,
} from "./utils";

const MIN_LEFT_PANEL_PCT = 15;
const MAX_LEFT_PANEL_PCT = 60;
const DEFAULT_LEFT_PANEL_PCT = 30;

export function updatePartitionLabelInDisks(
  disks: Disk[] | undefined,
  partitionId: string,
  label: string,
): Disk[] {
  if (!Array.isArray(disks) || disks.length === 0) {
    return [];
  }

  let hasChanges = false;
  const nextDisks = disks.map((disk) => {
    const partitionEntries = Object.entries(disk.partitions || {});
    if (partitionEntries.length === 0) {
      return disk;
    }

    let diskChanged = false;
    const nextPartitions = Object.fromEntries(
      partitionEntries.map(([key, partition]) => {
        if (
          !partition ||
          partition.id !== partitionId ||
          partition.name === label
        ) {
          return [key, partition];
        }

        diskChanged = true;
        hasChanges = true;
        return [key, { ...partition, name: label }];
      }),
    );

    return diskChanged ? { ...disk, partitions: nextPartitions } : disk;
  });

  return hasChanges ? nextDisks : disks;
}

export function Volumes({ initialDisks }: { initialDisks?: Disk[] } = {}) {
  const { data: evdata } = useGetServerEventsQuery();
  const [showPreview, setShowPreview] = useState<boolean>(false);
  const [showMount, setShowMount] = useState<boolean>(false);
  const [showFilesystemCheckDialog, setShowFilesystemCheckDialog] =
    useState(false);
  const [showFilesystemLabelDialog, setShowFilesystemLabelDialog] =
    useState(false);
  const [showFilesystemFormatDialog, setShowFilesystemFormatDialog] =
    useState(false);
  const location = useLocation();

  const navigate = useNavigate();
  const dispatch = useAppDispatch();
  const [hideSystemPartitions, setHideSystemPartitions] = useState<boolean>(
    localStorage.getItem("volumes.hideSystemPartitions") === "true",
  );
  const volumeHook = useVolume();
  const sourceDisks = initialDisks ?? volumeHook.disks;
  const isLoading = initialDisks ? false : volumeHook.isLoading;
  const error = initialDisks ? null : volumeHook.error;
  const [selectedDisk, setSelectedDisk] = useState<Disk | undefined>(undefined);
  const [selectedPartition, setSelectedPartition] = useState<
    Partition | undefined
  >(undefined);
  const [selectedPartitionId, setSelectedPartitionId] = useState<
    string | undefined
  >(() => localStorage.getItem("volumes.selectedPartitionId") || undefined);
  const [expandedDisks, setExpandedDisks] = useState<string[]>(() => {
    try {
      const savedExpanded = localStorage.getItem("volumes.expandedDisks");
      if (savedExpanded) {
        const parsed = JSON.parse(savedExpanded);
        if (Array.isArray(parsed)) return parsed as string[];
      }
    } catch {}
    return [];
  });
  const confirm = useConfirm();
  const [mountVolume, _mountVolumeResult] = usePostApiVolumeMountMutation();
  const [umountVolume, _umountVolumeResult] = useDeleteApiVolumeMutation();
  const [patchMountSettings] = usePatchApiVolumeSettingsMutation();
  const loggedLoadErrorRef = useRef<string>("");

  const [leftPanelPct, setLeftPanelPct] = useState<number>(() => {
    try {
      const saved = localStorage.getItem("volumes.leftPanelPct");
      if (saved) {
        const pct = parseFloat(saved);
        if (
          !Number.isNaN(pct) &&
          pct >= MIN_LEFT_PANEL_PCT &&
          pct <= MAX_LEFT_PANEL_PCT
        ) {
          return pct;
        }
      }
    } catch {}
    return DEFAULT_LEFT_PANEL_PCT;
  });

  const isDragging = useRef(false);
  const containerRef = useRef<HTMLDivElement | null>(null);

  const handleDividerMouseDown = useCallback((e: ReactMouseEvent) => {
    e.preventDefault();
    isDragging.current = true;
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";
  }, []);

  useEffect(() => {
    const handleMouseMove = (e: globalThis.MouseEvent) => {
      if (!isDragging.current || !containerRef.current) return;
      const rect = containerRef.current.getBoundingClientRect();
      const offsetX = e.clientX - rect.left;
      const pct = (offsetX / rect.width) * 100;
      const clamped = Math.min(
        MAX_LEFT_PANEL_PCT,
        Math.max(MIN_LEFT_PANEL_PCT, pct),
      );
      setLeftPanelPct(clamped);
    };

    const handleMouseUp = () => {
      if (!isDragging.current) return;
      isDragging.current = false;
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };

    window.addEventListener("mousemove", handleMouseMove);
    window.addEventListener("mouseup", handleMouseUp);
    return () => {
      window.removeEventListener("mousemove", handleMouseMove);
      window.removeEventListener("mouseup", handleMouseUp);
    };
  }, []);

  useEffect(() => {
    try {
      localStorage.setItem("volumes.leftPanelPct", String(leftPanelPct));
    } catch {}
  }, [leftPanelPct]);

  const perPartitionInfo = evdata?.heartbeat?.disk_health?.per_partition_info;
  const filesystemStateByPartitionId = useMemo<
    Record<string, FilesystemState>
  >(() => {
    const result: Record<string, FilesystemState> = {};
    if (!perPartitionInfo) return result;

    (Object.values(perPartitionInfo) as (PerPartitionInfo[] | null)[]).forEach(
      (partitionInfos) => {
        partitionInfos?.forEach((partitionInfo) => {
          if (partitionInfo.device && partitionInfo.filesystem_state) {
            result[partitionInfo.device] = partitionInfo.filesystem_state;
          }
        });
      },
    );

    return result;
  }, [perPartitionInfo]);

  const handleDiskSelect = useCallback(
    (disk: Disk) => {
      setSelectedDisk(disk);
      setSelectedPartition(undefined);
      const diskIdx = Math.max(sourceDisks?.indexOf(disk) ?? -1, 0);
      const diskIdentifier = getDiskIdentifier(disk, diskIdx);
      setSelectedPartitionId(diskIdentifier);
      setExpandedDisks((prev) =>
        prev.includes(diskIdentifier) ? prev : [...prev, diskIdentifier],
      );
    },
    [sourceDisks],
  );

  const handlePartitionSelect = useCallback(
    (disk: Disk, partition: Partition) => {
      setSelectedDisk(disk);
      setSelectedPartition(partition);
      const diskIdx = Math.max(sourceDisks?.indexOf(disk) ?? -1, 0);
      const diskIdentifier = getDiskIdentifier(disk, diskIdx);
      const partitionEntries = Object.entries(disk.partitions || {});
      const partitionEntry = partitionEntries.find(
        ([, value]) => value === partition,
      );
      const partitionKey = partitionEntry?.[0];
      const partIdx = Math.max(
        partitionEntry ? partitionEntries.indexOf(partitionEntry) : -1,
        0,
      );
      const partitionId = getPartitionIdentifier(
        diskIdentifier,
        partition,
        partitionKey,
        partIdx,
      );
      setSelectedPartitionId(partitionId);
      setExpandedDisks((prev) => {
        if (prev.includes(diskIdentifier)) return prev;
        return [...prev, diskIdentifier];
      });
    },
    [sourceDisks],
  );

  const handlePartitionLabelUpdated = useCallback(
    (partitionId: string, label: string) => {
      dispatch(
        sratApi.util.updateQueryData("getApiVolumes", undefined, (draft) => {
          if (Array.isArray(draft)) {
            updatePartitionLabelInDisks(draft, partitionId, label);
          }
        }),
      );
      setSelectedPartition((currentPartition) => {
        if (!currentPartition || currentPartition.id !== partitionId) {
          return currentPartition;
        }
        return { ...currentPartition, name: label };
      });
    },
    [dispatch],
  );

  const openDialogForPartition = useCallback(
    (partition: Partition, setDialogOpen: (open: boolean) => void) => {
      const activeElement = document.activeElement;
      if (activeElement instanceof HTMLElement) {
        activeElement.blur();
      }
      setSelectedPartition(partition);
      setDialogOpen(true);
    },
    [],
  );

  useEffect(() => {
    try {
      if (selectedPartitionId) {
        localStorage.setItem(
          "volumes.selectedPartitionId",
          selectedPartitionId,
        );
      } else {
        localStorage.removeItem("volumes.selectedPartitionId");
      }
    } catch (err) {
      console.warn("Could not persist selectedPartitionId", err);
    }
  }, [selectedPartitionId]);

  useEffect(() => {
    try {
      if (expandedDisks.length > 0) {
        localStorage.setItem(
          "volumes.expandedDisks",
          JSON.stringify(expandedDisks),
        );
      } else {
        localStorage.removeItem("volumes.expandedDisks");
      }
    } catch (err) {
      console.warn("Could not persist expandedDisks", err);
    }
  }, [expandedDisks]);

  useEffect(() => {
    try {
      localStorage.setItem(
        "volumes.hideSystemPartitions",
        hideSystemPartitions ? "true" : "false",
      );
    } catch (err) {
      console.warn("Could not persist hideSystemPartitions", err);
    }
  }, [hideSystemPartitions]);

  useEffect(() => {
    const state = location.state as LocationState | undefined;
    const mountPathFromState = state?.mountPathToView;

    if (
      mountPathFromState &&
      Array.isArray(sourceDisks) &&
      sourceDisks.length > 0
    ) {
      let foundPartition: Partition | undefined;
      let foundDisk: Disk | undefined;

      for (const disk of sourceDisks) {
        const partitions = Object.values(disk.partitions || {});
        if (partitions.length > 0) {
          for (const partition of partitions) {
            const mpds = Object.values(partition.mount_point_data || {});
            if (mpds.some((mpd) => mpd.path === mountPathFromState)) {
              foundPartition = partition;
              foundDisk = disk;
              break;
            }
          }
        }
        if (foundPartition) break;
      }

      if (foundPartition && foundDisk) {
        handlePartitionSelect(foundDisk, foundPartition);
        navigate(location.pathname, { replace: true, state: {} });
      } else {
        console.warn(
          `Volume with mountPathHash ${mountPathFromState} not found.`,
        );
        navigate(location.pathname, { replace: true, state: {} });
      }
    }
  }, [
    sourceDisks,
    location.state,
    navigate,
    location.pathname,
    handlePartitionSelect,
  ]);

  useEffect(() => {
    if (!sourceDisks || sourceDisks.length === 0) return;
    if (!selectedPartitionId) return;

    for (const disk of sourceDisks) {
      const diskIdx = Math.max(sourceDisks.indexOf(disk), 0);
      const diskIdentifier = getDiskIdentifier(disk, diskIdx);
      if (diskIdentifier === selectedPartitionId) {
        setSelectedDisk(disk);
        setSelectedPartition(undefined);
        setExpandedDisks((prev) =>
          prev.includes(diskIdentifier) ? prev : [...prev, diskIdentifier],
        );
        return;
      }
      const partitionEntries = Object.entries(disk.partitions || {});
      if (!partitionEntries || partitionEntries.length === 0) continue;
      for (let partIdx = 0; partIdx < partitionEntries.length; partIdx++) {
        const [partitionKey, partition] = partitionEntries[partIdx] as [
          string,
          Partition,
        ];
        const partitionIdentifier = getPartitionIdentifier(
          diskIdentifier,
          partition,
          partitionKey,
          partIdx,
        );
        if (partitionIdentifier === selectedPartitionId) {
          setSelectedDisk(disk);
          setSelectedPartition(partition);
          return;
        }
      }
    }

    setSelectedPartition(undefined);
    setSelectedDisk(undefined);
    setSelectedPartitionId(undefined);
  }, [sourceDisks, selectedPartitionId]);

  function onSubmitMountVolume(data?: MountPointData): Promise<void> {
    if (!selectedPartition || !data?.path || !data.root) {
      toast.error("Cannot mount: Invalid selection or missing data.");
      console.error("Mount validation failed:", {
        selectedPartition,
        data,
      });
      return Promise.resolve();
    }

    const submitData: MountPointData = {
      ...data,
      device_id: selectedPartition.id,
    };

    return mountVolume({
      mountPointData: submitData,
    })
      .unwrap()
      .then((res) => {
        toast.info(
          `Volume ${(res as MountPointData).path || selectedPartition.name} mounted successfully.`,
        );
      })
      .catch((err) => {
        console.error("Mount Error:", err);
        const errorData = err?.data || {};
        const errorMsg =
          errorData?.detail ||
          errorData?.message ||
          err?.status ||
          "Unknown mount error";
        const errorCode = errorData?.status || "Error";
        toast.error(`${errorCode}: ${errorMsg}`, {
          data: { error: errorData || err },
        });
      })
      .finally(() => {
        setSelectedPartition(undefined);
        setSelectedDisk(undefined);
        setSelectedPartitionId(undefined);
        setShowMount(false);
      });
  }

  function handleCreateShare(partition: Partition) {
    const firstMountPointData = Object.values(
      partition.mount_point_data || {},
    )[0];
    if (firstMountPointData?.path) {
      navigate("/", {
        state: {
          tabId: TabIDs.SHARES,
          newShareData: firstMountPointData,
        } as LocationState,
      });
    } else {
      toast.warn(
        "Cannot create share: Partition is not mounted or has no mount path.",
      );
    }
  }

  function handleGoToShare(partition: Partition) {
    const mountData = Object.values(partition.mount_point_data || {})[0];
    const share = mountData?.share;

    if (share?.name) {
      navigate("/", {
        state: { tabId: TabIDs.SHARES, shareName: share.name } as LocationState,
      });
    }
  }

  function onSubmitUmountVolume(partition: Partition, force = false) {
    console.debug("Umount Request", partition, "Force:", force);
    const mountData = Object.values(partition.mount_point_data || {})[0];
    if (!mountData?.path) {
      toast.error("Cannot unmount: Missing mount point path.");
      console.error("Missing mount path for partition:", partition);
      return;
    }

    const displayName = decodeEscapeSequence(partition.name || "this volume");

    confirm({
      title: `Unmount ${displayName}?`,
      description: `Do you really want to ${force ? "forcefully " : ""}unmount the Volume ${displayName} (${partition.legacy_device_name}) mounted at ${mountData.path}?`,
      confirmationText: force ? "Force Unmount" : "Unmount",
      cancellationText: "Cancel",
      confirmationButtonProps: { color: force ? "error" : "primary" },
      acknowledgement: `Please confirm this action carefully. Unmounting may lead to data loss or corruption if the volume is in use. ${force ? "NOTE:Configured shares will be disabled!" : ""}`,
    }).then(({ reason }) => {
      if (reason === "confirm") {
        console.debug(
          `Proceeding with ${force ? "forced " : ""}unmount for:`,
          mountData.path,
        );
        umountVolume({
          mountPath: mountData.path,
          force: force,
        })
          .unwrap()
          .then(() => {
            toast.info(`Volume ${displayName} unmounted successfully.`);
            if (selectedPartition?.id === partition.id) {
              setSelectedPartition(undefined);
              setSelectedDisk(undefined);
              setSelectedPartitionId(undefined);
            }
          })
          .catch((err) => {
            console.error("Unmount Error:", err);
            const errorData = err?.data || {};
            const errorMsg =
              errorData?.message || err?.status || "Unknown error";
            toast.error(`Error unmounting ${displayName}: ${errorMsg}`, {
              data: { error: err },
            });
          });
      }
    });
  }

  function handleToggleAutomount(partition: Partition) {
    if (evdata?.hello?.read_only) return;

    console.debug("Toggling automount for partition:", partition);

    const partitionName = decodeEscapeSequence(partition.name || "this volume");
    const mountEntries = Object.entries(partition.mount_point_data || {});
    if (mountEntries.length === 0) return;

    // Fire one PATCH per mount point and aggregate the results so a
    // multi-mount partition produces a single summary toast instead of
    // N interleaved toasts and uncoordinated failure handling.
    const patchPromises = mountEntries.map(async ([path, mountData]) => {
      if (!mountData.path) {
        throw new Error(`Cannot toggle automount: ${path} Missing point data.`);
      }

      const newAutomountState = !mountData.is_to_mount_at_startup;

      console.debug(
        partition,
        mountData,
        "Toggling automount to",
        newAutomountState,
      );

      await patchMountSettings({
        patchMountPointData: {
          ...mountData,
          is_to_mount_at_startup: newAutomountState,
          share: undefined,
        },
      }).unwrap();
    });

    void Promise.allSettled(patchPromises).then((settled) => {
      const fulfilledCount = settled.filter(
        (result) => result.status === "fulfilled",
      ).length;
      const rejectedCount = settled.length - fulfilledCount;

      settled.forEach((result, index) => {
        if (result.status === "rejected") {
          const [path] = mountEntries[index];
          console.error(
            `Error toggling automount for ${partitionName} (${path}):`,
            result.reason,
          );
        }
      });

      if (rejectedCount === 0) {
        toast.info(`Automount updated for ${partitionName}.`);
      } else if (fulfilledCount === 0) {
        toast.error(
          `Failed to update automount for ${partitionName} (${rejectedCount} mount point${rejectedCount === 1 ? "" : "s"}).`,
        );
      } else {
        toast.warn(
          `Automount partially updated for ${partitionName}: ${fulfilledCount} ok, ${rejectedCount} failed.`,
        );
      }
    });
  }

  useEffect(() => {
    if (!error) {
      loggedLoadErrorRef.current = "";
      return;
    }

    const errorSignature = JSON.stringify(error);
    if (loggedLoadErrorRef.current === errorSignature) {
      return;
    }

    loggedLoadErrorRef.current = errorSignature;
    console.error("Error loading volumes:", error);
  }, [error]);

  useEffect(() => {
    const selectTourVolume = () => {
      const target = getTourVolumeSelection(sourceDisks, hideSystemPartitions);
      if (!target) return;

      if (target.partition) {
        handlePartitionSelect(target.disk, target.partition);
        return;
      }

      handleDiskSelect(target.disk);
    };

    const disposeVolumesStep3 = TourEvents.on(
      TourEventTypes.VOLUMES_STEP_3,
      selectTourVolume,
    );
    const disposeVolumesStep4 = TourEvents.on(
      TourEventTypes.VOLUMES_STEP_4,
      selectTourVolume,
    );
    const disposeVolumesStep5 = TourEvents.on(
      TourEventTypes.VOLUMES_STEP_5,
      selectTourVolume,
    );

    return () => {
      disposeVolumesStep3();
      disposeVolumesStep4();
      disposeVolumesStep5();
    };
  }, [
    sourceDisks,
    hideSystemPartitions,
    handleDiskSelect,
    handlePartitionSelect,
  ]);

  if (isLoading) {
    return <Typography>Loading volumes...</Typography>;
  }

  if (error) {
    return (
      <Typography color="error">
        Error loading volume information. Please try again later.
      </Typography>
    );
  }

  return (
    <>
      <VolumeMountDialog
        objectToEdit={selectedPartition}
        open={showMount}
        readOnlyView={false}
        onClose={(data) => {
          if (showMount) {
            if (data) {
              return onSubmitMountVolume(data);
            }
            setSelectedPartition(undefined);
            setSelectedDisk(undefined);
            setSelectedPartitionId(undefined);
            setShowMount(false);
          }
          return;
        }}
      />
      {showFilesystemCheckDialog && (
        <FilesystemCheckDialog
          open={showFilesystemCheckDialog}
          partition={selectedPartition}
          onClose={() => setShowFilesystemCheckDialog(false)}
        />
      )}
      {showFilesystemLabelDialog && (
        <FilesystemLabelDialog
          open={showFilesystemLabelDialog}
          partition={selectedPartition}
          onClose={() => setShowFilesystemLabelDialog(false)}
          onLabelUpdated={handlePartitionLabelUpdated}
        />
      )}
      {showFilesystemFormatDialog && (
        <FilesystemFormatDialog
          open={showFilesystemFormatDialog}
          partition={selectedPartition}
          onClose={() => setShowFilesystemFormatDialog(false)}
        />
      )}
      <PreviewDialog
        title={
          selectedDisk && selectedPartition
            ? `Partition: ${decodeEscapeSequence(selectedPartition.name || selectedPartition.id || "Unknown")}`
            : selectedDisk
              ? `Disk: ${selectedDisk.model}`
              : "Details"
        }
        objectToDisplay={selectedPartition || selectedDisk}
        open={showPreview}
        onClose={() => {
          setSelectedPartition(undefined);
          setSelectedDisk(undefined);
          setSelectedPartitionId(undefined);
          setShowPreview(false);
        }}
      />
      <Box
        ref={containerRef}
        sx={{
          display: "flex",
          minHeight: "calc(100vh - 200px)",
          gap: 1,
        }}
        data-tutor={`reactour__tab${TabIDs.VOLUMES}__step0`}
      >
        <Box
          sx={{
            width: { xs: "min(45%, 180px)", sm: `${leftPanelPct}%` },
            minWidth: 0,
            flexShrink: 0,
          }}
          data-tutor={`reactour__tab${TabIDs.VOLUMES}__step3`}
        >
          <Paper sx={{ height: "100%", p: 1 }}>
            <Stack
              direction="row"
              sx={{
                justifyContent: "space-between",
                alignItems: "center",
                mb: 2,
                px: 2,
              }}
            >
              <Typography variant="h6">Volumes</Typography>
            </Stack>

            <Stack
              direction="row"
              data-tutor={`reactour__tab${TabIDs.VOLUMES}__step2`}
              sx={{
                justifyContent: "flex-start",
                pl: 2,
                mb: 1,
              }}
            >
              <FormControlLabel
                control={
                  <Switch
                    checked={hideSystemPartitions}
                    onChange={(e) => setHideSystemPartitions(e.target.checked)}
                    name="hideSystemPartitions"
                    size="small"
                  />
                }
                label={
                  <Typography variant="body2">
                    Hide system partitions
                  </Typography>
                }
              />
            </Stack>

            {isLoading ? (
              <Typography>Loading volumes...</Typography>
            ) : error ? (
              <Typography color="error">
                Error loading volume information. Please try again later.
              </Typography>
            ) : (
              <VolumesTreeView
                disks={sourceDisks}
                selectedItemId={selectedPartitionId}
                expandedItems={expandedDisks}
                onExpandedItemsChange={setExpandedDisks}
                hideSystemPartitions={hideSystemPartitions}
                filesystemStateByPartitionId={filesystemStateByPartitionId}
                onDiskSelect={handleDiskSelect}
                onPartitionSelect={handlePartitionSelect}
                onToggleAutomount={handleToggleAutomount}
                onMount={(partition) => {
                  setSelectedPartition(partition);
                  setShowMount(true);
                }}
                onUnmount={onSubmitUmountVolume}
                onCreateShare={handleCreateShare}
                onGoToShare={handleGoToShare}
                onCheckFilesystem={(partition) =>
                  openDialogForPartition(
                    partition,
                    setShowFilesystemCheckDialog,
                  )
                }
                onSetFilesystemLabel={(partition) =>
                  openDialogForPartition(
                    partition,
                    setShowFilesystemLabelDialog,
                  )
                }
                onFormatPartition={(partition) =>
                  openDialogForPartition(
                    partition,
                    setShowFilesystemFormatDialog,
                  )
                }
                protectedMode={evdata?.hello?.protected_mode === true}
                readOnly={evdata?.hello?.read_only === true}
              />
            )}
          </Paper>
        </Box>

        <Box
          onMouseDown={handleDividerMouseDown}
          sx={{
            width: 6,
            flexShrink: 0,
            cursor: "col-resize",
            backgroundColor: "divider",
            borderRadius: 1,
            alignSelf: "stretch",
            transition: "background-color 0.15s",
            "&:hover": {
              backgroundColor: "primary.main",
            },
          }}
        />

        <Box
          sx={{
            flex: 1,
            minWidth: 0,
          }}
          data-tutor={`reactour__tab${TabIDs.VOLUMES}__step4`}
        >
          <Paper sx={{ height: "100%", overflow: "hidden" }}>
            <Box data-tutor={`reactour__tab${TabIDs.VOLUMES}__step5`}>
              <VolumeDetailsPanel
                disk={selectedDisk}
                partition={selectedPartition}
                protectedMode={evdata?.hello?.protected_mode === true}
                readOnly={evdata?.hello?.read_only === true}
                onToggleAutomount={handleToggleAutomount}
                onMount={(partition) => {
                  setSelectedPartition(partition);
                  setShowMount(true);
                }}
                onUnmount={onSubmitUmountVolume}
                onCreateShare={handleCreateShare}
                onGoToShare={handleGoToShare}
                onLabelUpdated={handlePartitionLabelUpdated}
              />
            </Box>
          </Paper>
        </Box>
      </Box>
    </>
  );
}
