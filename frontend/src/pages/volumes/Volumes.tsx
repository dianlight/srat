import {
  Box,
  FormControlLabel,
  Paper,
  Stack,
  Switch,
  Typography,
} from "@mui/material";
import { useConfirm } from "material-ui-confirm";
import { useEffect, useMemo, useRef } from "react";
import { useLocation, useNavigate } from "react-router";
import { PreviewDialog } from "../../components/PreviewDialog";
import { ResizableSplitView } from "../../components/ResizableSplitView";
import { useVolume } from "../../hooks/volumeHook";
import { type LocationState, TabIDs } from "../../store/locationState";
import {
  type Disk,
  type FilesystemState,
  type Partition,
  type PerPartitionInfo,
  sratApi,
  useDeleteApiVolumeMutation,
  usePatchApiVolumeSettingsMutation,
} from "../../store/sratApi";
import { useAppDispatch } from "../../store/store";
import { useGetServerEventsQuery } from "../../store/wsApi";
import { decodeEscapeSequence } from "../../utils/decodeEscapeSequence";
import { TourEvents, TourEventTypes } from "../../utils/TourEvents";
import {
  FilesystemCheckDialog,
  FilesystemFormatDialog,
  FilesystemLabelDialog,
  VolumeDetailsPanel,
  VolumeMountDialog,
  VolumesTreeView,
} from "./components";
import { useMountVolume } from "./hooks/useMountVolume";
import { useVolumeSelection } from "./hooks/useVolumeSelection";
import { getTourVolumeSelection } from "./tourSelection";
import {
  findPartitionByMountPath,
  navigateToCreateShare,
  navigateToShare,
  requestAutomountToggle,
  requestUnmountVolume,
  updatePartitionLabelInDisks,
} from "./utils";

export function Volumes({ initialDisks }: { initialDisks?: Disk[] } = {}) {
  const { data: evdata } = useGetServerEventsQuery();
  const location = useLocation();
  const navigate = useNavigate();
  const dispatch = useAppDispatch();
  const volumeHook = useVolume();
  const sourceDisks = initialDisks ?? volumeHook.disks;
  const isLoading = initialDisks ? false : volumeHook.isLoading;
  const error = initialDisks ? null : volumeHook.error;
  const sel = useVolumeSelection({ sourceDisks });
  // Mount hook clears selection and closes the mount dialog when done.
  const mount = useMountVolume({
    selectedPartition: sel.selectedPartition,
    onCleared: sel.closeMount,
  });
  const [umountVolume] = useDeleteApiVolumeMutation();
  const [patchMountSettings] = usePatchApiVolumeSettingsMutation();
  const confirm = useConfirm();
  const loggedLoadErrorRef = useRef<string>("");

  const perPartitionInfo = evdata?.heartbeat?.disk_health?.per_partition_info;
  const filesystemStateByPartitionId = useMemo<
    Record<string, FilesystemState>
  >(() => {
    const result: Record<string, FilesystemState> = {};
    const infos = Object.values(perPartitionInfo ?? {}) as (
      | PerPartitionInfo[]
      | null
    )[];
    for (const partitionInfos of infos) {
      for (const info of partitionInfos ?? []) {
        if (info.device && info.filesystem_state) {
          result[info.device] = info.filesystem_state;
        }
      }
    }
    return result;
  }, [perPartitionInfo]);

  function handlePartitionLabelUpdated(partitionId: string, label: string) {
    dispatch(
      sratApi.util.updateQueryData("getApiVolumes", undefined, (draft) => {
        // The helper is pure (returns new arrays); the recipe must return its
        // result so Immer writes it into the cache. Discarding the return
        // leaves the draft untouched and the optimistic update never lands.
        if (!Array.isArray(draft)) return undefined;
        return updatePartitionLabelInDisks(draft, partitionId, label);
      }),
    );
    sel.setSelectedPartition((current) =>
      current && current.id === partitionId
        ? { ...current, name: label }
        : current,
    );
  }

  const handleCreateShare = (partition: Partition) =>
    navigateToCreateShare(navigate, partition);

  const handleGoToShare = (partition: Partition) =>
    navigateToShare(navigate, partition);

  function onSubmitUmountVolume(partition: Partition, force = false) {
    requestUnmountVolume({
      confirm,
      unmount: umountVolume,
      partition,
      force,
      isSelected: sel.selectedPartition?.id === partition.id,
      onCleared: sel.clearSelection,
    });
  }

  function handleToggleAutomount(partition: Partition) {
    requestAutomountToggle({
      patchSettings: patchMountSettings,
      readOnly: evdata?.hello?.read_only === true,
      partition,
    });
  }

  useEffect(() => {
    if (!error) {
      loggedLoadErrorRef.current = "";
      return;
    }
    const signature = JSON.stringify(error);
    if (loggedLoadErrorRef.current === signature) return;
    loggedLoadErrorRef.current = signature;
    console.error("Error loading volumes:", error);
  }, [error]);

  useEffect(() => {
    const mountPathFromState = (location.state as LocationState | undefined)
      ?.mountPathToView;
    if (
      !mountPathFromState ||
      !Array.isArray(sourceDisks) ||
      sourceDisks.length === 0
    )
      return;
    const match = findPartitionByMountPath(sourceDisks, mountPathFromState);
    if (match) sel.handlePartitionSelect(match.disk, match.partition);
    else
      console.warn(
        `Volume with mountPathHash ${mountPathFromState} not found.`,
      );
    navigate(location.pathname, { replace: true, state: {} });
  }, [
    sourceDisks,
    location.state,
    navigate,
    location.pathname,
    sel.handlePartitionSelect,
  ]);

  useEffect(() => {
    const selectTourVolume = () => {
      const target = getTourVolumeSelection(
        sourceDisks,
        sel.hideSystemPartitions,
      );
      if (!target) return;
      if (target.partition)
        sel.handlePartitionSelect(target.disk, target.partition);
      else sel.handleDiskSelect(target.disk);
    };
    const eventTypes = [
      TourEventTypes.VOLUMES_STEP_3,
      TourEventTypes.VOLUMES_STEP_4,
      TourEventTypes.VOLUMES_STEP_5,
    ];
    const disposers = eventTypes.map((type) =>
      TourEvents.on(type, selectTourVolume),
    );
    return () => {
      disposers.forEach((dispose) => {
        dispose();
      });
    };
  }, [
    sourceDisks,
    sel.hideSystemPartitions,
    sel.handleDiskSelect,
    sel.handlePartitionSelect,
  ]);

  if (isLoading) return <Typography>Loading volumes...</Typography>;
  if (error) {
    return (
      <Typography color="error">
        Error loading volume information. Please try again later.
      </Typography>
    );
  }

  const treeSelection = {
    selectedId: sel.selectedPartitionId,
    expanded: sel.expandedDisks,
    onExpandedChange: sel.setExpandedDisks,
    onDiskSelect: sel.handleDiskSelect,
    onPartitionSelect: sel.handlePartitionSelect,
    hideSystemPartitions: sel.hideSystemPartitions,
  };
  const treeActions = {
    onToggleAutomount: handleToggleAutomount,
    onMount: sel.openMount,
    onUnmount: onSubmitUmountVolume,
    onCreateShare: handleCreateShare,
    onGoToShare: handleGoToShare,
    onCheckFilesystem: (partition: Partition) =>
      sel.openDialogForPartition(partition, sel.setShowFilesystemCheckDialog),
    onSetFilesystemLabel: (partition: Partition) =>
      sel.openDialogForPartition(partition, sel.setShowFilesystemLabelDialog),
    onFormatPartition: (partition: Partition) =>
      sel.openDialogForPartition(partition, sel.setShowFilesystemFormatDialog),
  };
  const treeStatus = {
    protectedMode: evdata?.hello?.protected_mode === true,
    readOnly: evdata?.hello?.read_only === true,
    filesystemStateByPartitionId,
  };
  const detailsActions = {
    ...treeStatus,
    onToggleAutomount: handleToggleAutomount,
    onMount: sel.openMount,
    onUnmount: onSubmitUmountVolume,
    onCreateShare: handleCreateShare,
    onGoToShare: handleGoToShare,
    onLabelUpdated: handlePartitionLabelUpdated,
  };
  const previewTitle =
    sel.selectedDisk && sel.selectedPartition
      ? `Partition: ${decodeEscapeSequence(sel.selectedPartition.name || sel.selectedPartition.id || "Unknown")}`
      : sel.selectedDisk
        ? `Disk: ${sel.selectedDisk.model}`
        : "Details";

  return (
    <>
      <VolumeMountDialog
        objectToEdit={sel.selectedPartition}
        open={sel.showMount}
        readOnlyView={false}
        onClose={(data) => {
          if (!sel.showMount) return;
          if (data) {
            void mount.onSubmitMountVolume(data);
            return;
          }
          sel.closeMount();
        }}
      />
      {sel.showFilesystemCheckDialog && (
        <FilesystemCheckDialog
          open
          partition={sel.selectedPartition}
          onClose={() => sel.setShowFilesystemCheckDialog(false)}
        />
      )}
      {sel.showFilesystemLabelDialog && (
        <FilesystemLabelDialog
          open
          partition={sel.selectedPartition}
          onClose={() => sel.setShowFilesystemLabelDialog(false)}
          onLabelUpdated={handlePartitionLabelUpdated}
        />
      )}
      {sel.showFilesystemFormatDialog && (
        <FilesystemFormatDialog
          open
          partition={sel.selectedPartition}
          onClose={() => sel.setShowFilesystemFormatDialog(false)}
        />
      )}
      <PreviewDialog
        title={previewTitle}
        objectToDisplay={sel.selectedPartition || sel.selectedDisk}
        open={sel.showPreview}
        onClose={sel.closePreview}
      />
      <ResizableSplitView
        storageKey="volumes.leftPanelPct"
        containerProps={{
          "data-tutor": `reactour__tab${TabIDs.VOLUMES}__step0`,
        }}
        leftPanelProps={{
          "data-tutor": `reactour__tab${TabIDs.VOLUMES}__step3`,
        }}
        leftPanel={
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
              sx={{ justifyContent: "flex-start", pl: 2, mb: 1 }}
            >
              <FormControlLabel
                control={
                  <Switch
                    checked={sel.hideSystemPartitions}
                    onChange={(e) =>
                      sel.setHideSystemPartitions(e.target.checked)
                    }
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
            <VolumesTreeView
              disks={sourceDisks}
              selection={treeSelection}
              actions={treeActions}
              status={treeStatus}
            />
          </Paper>
        }
        rightPanelProps={{
          "data-tutor": `reactour__tab${TabIDs.VOLUMES}__step4`,
        }}
        rightPanel={
          <Paper sx={{ height: "100%", overflow: "hidden" }}>
            <Box data-tutor={`reactour__tab${TabIDs.VOLUMES}__step5`}>
              <VolumeDetailsPanel
                disk={sel.selectedDisk}
                partition={sel.selectedPartition}
                actions={detailsActions}
              />
            </Box>
          </Paper>
        }
      />
    </>
  );
}
