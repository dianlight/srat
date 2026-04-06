import {
  Alert,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  FormControlLabel,
  LinearProgress,
  MenuItem,
  Stack,
  Switch,
  TextField,
  Typography,
} from "@mui/material";
import { useEffect, useMemo, useRef, useState } from "react";
import { toast } from "react-toastify";
import {
  type ErrorModel,
  type FilesystemsInfo,
  type FilesystemTask,
  type Partition,
  useGetApiFilesystemSupportQuery,
  useGetApiFilesystemsQuery,
  usePostApiFilesystemFormatMutation,
} from "../../../store/sratApi";
import { useGetServerEventsQuery } from "../../../store/wsApi";
import { decodeEscapeSequence, getFilesystemLabelValidation } from "../utils";

interface FilesystemFormatDialogProps {
  open: boolean;
  partition?: Partition;
  onClose: () => void;
  taskOverride?: FilesystemTask | null;
  initialVerbose?: boolean;
}

interface FilesystemSupportWithLabelRule {
  alpinePackage?: string;
  canFormat?: boolean;
  labelRule?: string | null;
  missingTools?: string[] | null;
}

const isRunningStatus = (status?: string) =>
  status === "start" || status === "running";

const matchesPartitionDevice = (
  task: FilesystemTask,
  partition?: Partition,
) => {
  if (!partition) return false;
  const candidates = new Set<string>();
  if (partition.device_path) candidates.add(partition.device_path);
  if (partition.legacy_device_path) {
    candidates.add(partition.legacy_device_path);
  }
  return candidates.has(task.device ?? "");
};

export function FilesystemFormatDialog({
  open,
  partition,
  onClose,
  taskOverride,
  initialVerbose,
}: FilesystemFormatDialogProps) {
  const { data: eventData } = useGetServerEventsQuery();
  const [filesystemType, setFilesystemType] = useState("ext4");
  const [label, setLabel] = useState("");
  const [force, setForce] = useState(false);
  const [verbose, setVerbose] = useState(Boolean(initialVerbose));
  const [logs, setLogs] = useState<string[]>([]);
  const [progress, setProgress] = useState<number>(0);
  const [status, setStatus] = useState<string>("idle");
  const [message, setMessage] = useState<string>("");

  const lastNotesRef = useRef<string[]>([]);
  const lastMessageRef = useRef<string>("");

  const { data: filesystemsData, isFetching: isFilesystemsLoading } =
    useGetApiFilesystemsQuery(undefined, { skip: !open });

  const {
    data: supportData,
    isFetching: isSupportLoading,
    isError: isSupportError,
    error: supportError,
  } = useGetApiFilesystemSupportQuery(
    { fstype: filesystemType },
    { skip: !open || filesystemType.trim() === "" },
  );
  const [formatMutation, { isLoading: isFormatting }] =
    usePostApiFilesystemFormatMutation();

  const partitionId = partition?.id;
  const initialFilesystemType = partition?.fs_type?.trim() || "ext4";

  const task = useMemo<FilesystemTask | null>(() => {
    if (taskOverride) {
      if (taskOverride.operation !== "format") return null;
      return matchesPartitionDevice(taskOverride, partition)
        ? taskOverride
        : null;
    }
    const candidate = eventData?.filesystem_task;
    if (!candidate || candidate.operation !== "format") return null;
    return matchesPartitionDevice(candidate, partition) ? candidate : null;
  }, [eventData?.filesystem_task, partition, taskOverride]);

  const isRunning = useMemo(
    () => isRunningStatus(task?.status) || isRunningStatus(status),
    [task?.status, status],
  );

  useEffect(() => {
    if (!open) {
      return;
    }

    setFilesystemType(initialFilesystemType);
    setLabel("");
    setForce(false);
    setVerbose(Boolean(initialVerbose));
    setLogs([]);
    setProgress(0);
    setStatus("idle");
    setMessage("");
    lastNotesRef.current = [];
    lastMessageRef.current = "";

    if (!partitionId) {
      return;
    }
  }, [open, partitionId, initialFilesystemType, initialVerbose]);

  const support = useMemo<FilesystemSupportWithLabelRule | null>(() => {
    if (!supportData || !("canFormat" in supportData)) {
      return null;
    }
    return supportData as FilesystemSupportWithLabelRule;
  }, [supportData]);

  const isSupportPending =
    open && filesystemType.trim() !== "" && supportData === undefined;

  const formatCapableFilesystemTypes = useMemo(
    () =>
      ((filesystemsData as FilesystemsInfo | undefined)?.filesystems ?? [])
        .filter((filesystem) => filesystem?.support?.canFormat)
        .map((filesystem) => ({
          type: filesystem.type,
          label: filesystem.description
            ? `${filesystem.description} (${filesystem.type})`
            : filesystem.type,
        })),
    [filesystemsData],
  );

  const isSelectedFormatTypeAvailable = useMemo(
    () =>
      formatCapableFilesystemTypes.some(
        (filesystem) => filesystem.type === filesystemType,
      ),
    [filesystemType, formatCapableFilesystemTypes],
  );

  const dropdownFilesystemType = isSelectedFormatTypeAvailable
    ? filesystemType
    : "";

  const canFormat = useMemo(() => {
    if (support?.canFormat !== undefined) {
      return support.canFormat;
    }
    if (partition?.filesystem_info?.support?.canFormat !== undefined) {
      return partition.filesystem_info.support.canFormat;
    }
    return true;
  }, [partition?.filesystem_info?.support?.canFormat, support?.canFormat]);

  const supportErrorMessage = useMemo(() => {
    if (!isSupportError) {
      return "";
    }
    const maybeError = supportError as {
      data?: ErrorModel;
      error?: string;
      message?: string;
    };
    return (
      maybeError?.data?.detail ||
      maybeError?.data?.title ||
      maybeError?.error ||
      maybeError?.message ||
      "Failed to verify format support."
    );
  }, [isSupportError, supportError]);

  useEffect(() => {
    if (!open || !task) {
      return;
    }

    if (task.status) {
      setStatus(task.status);
    }
    if (task.message) {
      setMessage(task.message);
    }
    if (typeof task.progress === "number") {
      setProgress(task.progress);
    }

    const taskNotes = task.notes ?? [];
    if (taskNotes.length > 0) {
      const previousNotes = lastNotesRef.current;
      const isCumulativeNotes =
        taskNotes.length >= previousNotes.length &&
        previousNotes.every((note, index) => note === taskNotes[index]);

      if (isCumulativeNotes) {
        const newNotes = taskNotes.slice(previousNotes.length);
        if (newNotes.length > 0) {
          setLogs((prev) => [...prev, ...newNotes]);
        }
      } else {
        setLogs((prev) => [...prev, ...taskNotes]);
      }

      lastNotesRef.current = taskNotes;
    }

    const taskMessage = task.message?.trim() ?? "";
    if (
      taskMessage &&
      !taskNotes.includes(taskMessage) &&
      taskMessage !== lastMessageRef.current
    ) {
      setLogs((prev) => [...prev, taskMessage]);
    }

    if (taskMessage) {
      lastMessageRef.current = taskMessage;
    }
  }, [open, task]);

  useEffect(() => {
    if (!open) {
      return;
    }

    if (formatCapableFilesystemTypes.length === 0) {
      return;
    }

    const isCurrentTypeAvailable = formatCapableFilesystemTypes.some(
      (filesystem) => filesystem.type === filesystemType,
    );

    if (!isCurrentTypeAvailable) {
      setFilesystemType(formatCapableFilesystemTypes[0].type);
    }
  }, [filesystemType, formatCapableFilesystemTypes, open]);

  const labelRule = useMemo(() => {
    if (support?.labelRule) {
      return support.labelRule;
    }
    const partitionSupport = partition?.filesystem_info?.support as
      | FilesystemSupportWithLabelRule
      | undefined;
    return partitionSupport?.labelRule ?? "";
  }, [partition?.filesystem_info?.support, support?.labelRule]);

  const labelValidation = useMemo(
    () => getFilesystemLabelValidation(label, labelRule, true),
    [label, labelRule],
  );

  const showLabelError = label.trim().length > 0 && !labelValidation.isValid;

  const handleFormat = async () => {
    if (!partition?.id) {
      toast.error("Partition not selected.");
      return;
    }
    if (!filesystemType.trim()) {
      toast.error("Filesystem type is required.");
      return;
    }
    if (!labelValidation.isValid) {
      toast.error(labelValidation.helperText || "Label format is not valid.");
      return;
    }

    setLogs([]);
    setProgress(0);
    setStatus("start");
    setMessage("Starting format operation...");
    lastNotesRef.current = [];
    lastMessageRef.current = "";

    try {
      await formatMutation({
        formatPartitionInput: {
          partitionId: partition.id,
          filesystemType: filesystemType.trim(),
          label: label.trim(),
          force,
          verbose,
        },
      }).unwrap();
      toast.info("Format operation started.");
    } catch (err: unknown) {
      const typedErr = err as {
        data?: { detail?: string; title?: string };
        message?: string;
      };
      const errorMsg =
        typedErr?.data?.detail ||
        typedErr?.data?.title ||
        typedErr?.message ||
        "Failed to start format operation.";
      toast.error(errorMsg);
      setStatus("failure");
      setMessage(errorMsg);
      setLogs((prev) => [...prev, errorMsg]);
    }
  };

  const progressValue =
    typeof task?.progress === "number" ? task.progress : progress;
  const clampedProgressValue = Math.min(100, Math.max(0, progressValue));
  const showIndeterminate =
    isRunning && (progressValue === 999 || progressValue <= 0);
  const partitionLabel = decodeEscapeSequence(
    partition?.name || partition?.id || "Selected partition",
  );

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>Format Partition: {partitionLabel}</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ pt: 1 }}>
          <DialogContentText>
            Format this partition using the selected filesystem type. Enable
            Verbose to inspect the formatter output while it runs.
          </DialogContentText>

          {!isSupportLoading && !canFormat && (
            <Alert severity="warning">
              <Typography variant="body2" sx={{ fontWeight: 600 }}>
                Format tools are not available for this filesystem on the
                current system.
              </Typography>
              {support?.missingTools && support.missingTools.length > 0 && (
                <Typography variant="body2">
                  Missing tools: {support.missingTools.join(", ")}
                </Typography>
              )}
              {support?.alpinePackage && (
                <Typography variant="body2">
                  Install hint: <code>apk add {support.alpinePackage}</code>
                </Typography>
              )}
            </Alert>
          )}

          {isSupportError && (
            <Alert severity="info">{supportErrorMessage}</Alert>
          )}

          <Box>
            <Typography
              variant="subtitle2"
              color="text.secondary"
              sx={{ mb: 0.5 }}
            >
              Progress
            </Typography>
            <LinearProgress
              variant={showIndeterminate ? "indeterminate" : "determinate"}
              value={showIndeterminate ? undefined : clampedProgressValue}
            />
            <Stack
              direction="row"
              justifyContent="space-between"
              sx={{ mt: 0.5 }}
            >
              <Typography variant="caption" color="text.secondary">
                {status ? status.toUpperCase() : "IDLE"}
              </Typography>
              <Typography variant="caption" color="text.secondary">
                {showIndeterminate
                  ? "Working..."
                  : `${Math.round(clampedProgressValue)}%`}
              </Typography>
            </Stack>
            {progressValue === 999 && (
              <Typography variant="caption" color="text.secondary">
                This tool does not report incremental progress. Live output is
                shown in logs.
              </Typography>
            )}
          </Box>

          {message && (
            <Typography variant="body2" color="text.secondary">
              {message}
            </Typography>
          )}

          <TextField
            label="Filesystem type"
            value={dropdownFilesystemType}
            onChange={(event) => setFilesystemType(event.target.value)}
            fullWidth
            disabled={isFormatting || isRunning || isFilesystemsLoading}
            select
            helperText={
              isFilesystemsLoading
                ? "Loading available filesystems..."
                : formatCapableFilesystemTypes.length === 0
                  ? "No filesystem available for format"
                  : undefined
            }
          >
            {formatCapableFilesystemTypes.map((filesystem) => (
              <MenuItem key={filesystem.type} value={filesystem.type}>
                {filesystem.label}
              </MenuItem>
            ))}
          </TextField>

          <TextField
            label="Label (optional)"
            value={label}
            onChange={(event) => setLabel(event.target.value)}
            fullWidth
            disabled={isFormatting || isRunning}
            error={showLabelError}
            helperText={labelValidation.helperText}
          />

          <Stack direction={{ xs: "column", sm: "row" }} spacing={2}>
            <FormControlLabel
              control={
                <Switch
                  checked={force}
                  onChange={(event) => setForce(event.target.checked)}
                  disabled={isFormatting || isRunning}
                />
              }
              label="Force"
            />
            <FormControlLabel
              control={
                <Switch
                  checked={verbose}
                  onChange={(event) => setVerbose(event.target.checked)}
                  disabled={isFormatting || isRunning}
                />
              }
              label="Verbose"
            />
          </Stack>

          {verbose && (
            <TextField
              label="Logs"
              value={logs.join("\n")}
              multiline
              minRows={6}
              maxRows={12}
              InputProps={{ readOnly: true }}
              placeholder="No logs yet."
            />
          )}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} color="secondary" variant="outlined">
          Close
        </Button>
        <Button
          onClick={handleFormat}
          color="error"
          variant="contained"
          disabled={
            isFormatting ||
            isRunning ||
            !partition?.id ||
            filesystemType.trim().length === 0 ||
            formatCapableFilesystemTypes.length === 0 ||
            !isSelectedFormatTypeAvailable ||
            !labelValidation.isValid ||
            isSupportPending ||
            isSupportLoading ||
            !canFormat
          }
        >
          {isRunning ? "Formatting..." : "Format"}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
