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
  Stack,
  Switch,
  Typography,
} from "@mui/material";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { toast } from "react-toastify";
import {
  createTerminalLines,
  ReadonlyCommandTerminal,
} from "../../../components/ReadonlyCommandTerminal";
import {
  type CommandOutputLineSnapshot,
  type ErrorModel,
  type FilesystemTask,
  type Partition,
  useGetApiFilesystemSupportQuery,
  usePostApiFilesystemCheckAbortMutation,
  usePostApiFilesystemCheckMutation,
} from "../../../store/sratApi";
import { useGetServerEventsQuery } from "../../../store/wsApi";
import { decodeEscapeSequence } from "../utils";

interface FilesystemCheckDialogProps {
  open: boolean;
  partition?: Partition;
  onClose: () => void;
  taskOverride?: FilesystemTask | null;
  initialVerbose?: boolean;
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
  if (partition.legacy_device_path)
    candidates.add(partition.legacy_device_path);
  return candidates.has(task.device ?? "");
};

const extractTaskResultMessage = (task?: FilesystemTask | null) => {
  const result = task?.result;
  if (!result || typeof result !== "object") {
    return "";
  }

  const maybeResult = result as {
    Message?: string;
    message?: string;
  };

  return (maybeResult.Message ?? maybeResult.message ?? "").trim();
};

export function FilesystemCheckDialog({
  open,
  partition,
  onClose,
  taskOverride,
  initialVerbose,
}: FilesystemCheckDialogProps) {
  const { data: eventData } = useGetServerEventsQuery();
  const fsType = partition?.fs_type ?? "";
  const {
    data: supportData,
    isFetching: isSupportLoading,
    isError: isSupportError,
    error: supportError,
  } = useGetApiFilesystemSupportQuery(
    { fstype: fsType },
    { skip: !open || fsType === "" },
  );
  const [startCheck, { isLoading: isStarting }] =
    usePostApiFilesystemCheckMutation();
  const [abortCheck, { isLoading: isStopping }] =
    usePostApiFilesystemCheckAbortMutation();

  const [autoFix, setAutoFix] = useState(false);
  const [force, setForce] = useState(false);
  const [verbose, setVerbose] = useState(
    initialVerbose !== undefined ? Boolean(initialVerbose) : true,
  );
  const [logs, setLogs] = useState<CommandOutputLineSnapshot[]>([]);
  const [progress, setProgress] = useState<number>(0);
  const [status, setStatus] = useState<string>("idle");
  const [message, setMessage] = useState<string>("");

  const lastNotesRef = useRef<string[]>([]);
  const lastMessageRef = useRef<string>("");
  const lastErrorRef = useRef<string>("");
  const lastResultMessageRef = useRef<string>("");
  const nextLogTimestampRef = useRef(0);

  const appendLogs = useCallback(
    (entries: string[], channel: "stdout" | "stderr" | "info" = "info") => {
      if (entries.length === 0) {
        return;
      }

      const startTimestamp = Date.now() + nextLogTimestampRef.current;
      nextLogTimestampRef.current += entries.length;
      setLogs((prev) => [
        ...prev,
        ...createTerminalLines(entries, channel, startTimestamp),
      ]);
    },
    [],
  );

  const task = useMemo<FilesystemTask | null>(() => {
    if (taskOverride) {
      if (taskOverride.operation !== "check") return null;
      return matchesPartitionDevice(taskOverride, partition)
        ? taskOverride
        : null;
    }
    const candidate = eventData?.filesystem_task;
    if (!candidate || candidate.operation !== "check") return null;
    return matchesPartitionDevice(candidate, partition) ? candidate : null;
  }, [eventData?.filesystem_task, partition, taskOverride]);

  const isRunning = useMemo(
    () => isRunningStatus(task?.status) || isRunningStatus(status),
    [task?.status, status],
  );

  const support = useMemo(() => {
    if (!supportData || !("canCheck" in supportData)) {
      return null;
    }
    return supportData;
  }, [supportData]);

  const canCheck = useMemo(() => {
    if (support?.canCheck !== undefined) {
      return support.canCheck;
    }
    if (partition?.filesystem_info?.support?.canCheck !== undefined) {
      return partition.filesystem_info.support.canCheck;
    }
    return true;
  }, [partition?.filesystem_info?.support?.canCheck, support?.canCheck]);

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
      "Failed to verify filesystem check support."
    );
  }, [isSupportError, supportError]);

  const partitionId = partition?.id;

  useEffect(() => {
    if (!open) return;
    if (!partitionId) return;
    setLogs([]);
    setProgress(0);
    setStatus("idle");
    setMessage("");
    setVerbose(initialVerbose !== undefined ? Boolean(initialVerbose) : true);
    lastNotesRef.current = [];
    lastMessageRef.current = "";
    lastErrorRef.current = "";
    lastResultMessageRef.current = "";
    nextLogTimestampRef.current = 0;
  }, [open, partitionId, initialVerbose]);

  useEffect(() => {
    if (!open || !task) return;
    if (task.status) {
      setStatus(task.status);
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

      const routeNotes = (notes: string[]) => {
        const stdoutNotes = notes.filter((n) => !n.startsWith("ERROR: "));
        const stderrNotes = notes
          .filter((n) => n.startsWith("ERROR: "))
          .map((n) => n.slice("ERROR: ".length));
        if (stdoutNotes.length > 0) appendLogs(stdoutNotes, "stdout");
        if (stderrNotes.length > 0) appendLogs(stderrNotes, "stderr");
      };

      if (isCumulativeNotes) {
        const newNotes = taskNotes.slice(previousNotes.length);
        if (newNotes.length > 0) {
          routeNotes(newNotes);
        }
      } else {
        routeNotes(taskNotes);
      }

      lastNotesRef.current = taskNotes;

      // Auto-enable verbose log when tool doesn't report incremental progress
      if (taskNotes.some((n) => n === "Progress Status Not Supported")) {
        setVerbose(true);
      }
    }

    const taskMessage = task.message?.trim() ?? "";
    const taskError = task.error?.trim() ?? "";
    const resultMessage = extractTaskResultMessage(task);
    const preferredMessage =
      task.status === "failure"
        ? taskError || taskMessage || resultMessage
        : taskMessage || resultMessage || taskError;

    if (preferredMessage) {
      setMessage(preferredMessage);
    }

    const newErrorMessages =
      taskError &&
      !taskNotes.includes(taskError) &&
      taskError !== lastErrorRef.current
        ? [taskError]
        : [];
    // Only suppress running-status messages from the log when notes are present
    // (the message is a compound echo of notes: "Check /dev/...: running - <line>").
    // When there are no notes, the message is meaningful and should appear.
    const newTaskMessages =
      taskMessage && !(task.status === "running" && taskNotes.length > 0)
        ? [taskMessage].filter(
            (line) =>
              !taskNotes.includes(line) && line !== lastMessageRef.current,
          )
        : [];
    // Split multi-line result message so each line is checked/deduplicated individually.
    // Skip entirely if notes are present — they already contain all output lines.
    const newResultMessages =
      resultMessage && taskNotes.length === 0
        ? resultMessage
            .split("\n")
            .map((l) => l.trim())
            .filter((l) => l !== "")
        : [];

    appendLogs(newTaskMessages, task.status === "failure" ? "stderr" : "info");
    appendLogs(newResultMessages, "stdout");
    appendLogs(newErrorMessages, "stderr");

    if (taskMessage) {
      lastMessageRef.current = taskMessage;
    }
    if (taskError) {
      lastErrorRef.current = taskError;
    }
    if (resultMessage) {
      lastResultMessageRef.current = resultMessage;
    }
  }, [appendLogs, open, task]);

  const handleStart = async () => {
    if (!partition?.id) {
      toast.error("Partition not selected.");
      return;
    }
    setLogs([]);
    nextLogTimestampRef.current = 0;
    setProgress(0);
    setStatus("start");
    setMessage("Starting filesystem check...");
    appendLogs(["Starting filesystem check..."], "info");
    try {
      await startCheck({
        checkPartitionInput: {
          partitionId: partition.id,
          autoFix,
          force,
          verbose,
        },
      }).unwrap();
      // toast.info("Filesystem check started.");
    } catch (err: unknown) {
      const typedErr = err as {
        data?: { detail?: string; message?: string };
        message?: string;
      };
      const errorMsg =
        typedErr?.data?.detail ||
        typedErr?.data?.message ||
        typedErr?.message ||
        "Failed to start filesystem check";
      toast.error(errorMsg);
      setStatus("failure");
      appendLogs([errorMsg], "stderr");
    }
  };

  const handleStop = async () => {
    if (!partition?.id) {
      toast.error("Partition not selected.");
      return;
    }
    try {
      await abortCheck({
        abortCheckPartitionInput: {
          partitionId: partition.id,
        },
      }).unwrap();
      toast.info("Filesystem check abort requested.");
    } catch (err: unknown) {
      const typedErr = err as {
        data?: { detail?: string; message?: string };
        message?: string;
      };
      const errorMsg =
        typedErr?.data?.detail ||
        typedErr?.data?.message ||
        typedErr?.message ||
        "Failed to abort filesystem check";
      toast.error(errorMsg);
    }
  };

  const progressValue =
    typeof task?.progress === "number" ? task.progress : progress;
  const clampedProgressValue = Math.min(100, Math.max(0, progressValue));
  const showIndeterminate =
    isRunning && (progressValue === 999 || progressValue <= 0);
  const showUnsupportedHint = !isSupportLoading && !canCheck;
  const partitionLabel = decodeEscapeSequence(
    partition?.name || partition?.id || "Selected partition",
  );

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>Filesystem Check: {partitionLabel}</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ pt: 1 }}>
          <DialogContentText>
            Run a filesystem consistency check. Use AutoFix to repair errors
            when possible.
          </DialogContentText>

          {showUnsupportedHint && (
            <Alert severity="warning">
              <Typography variant="body2" sx={{ fontWeight: 600 }}>
                Check tools are not available for this filesystem on the current
                system.
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
              sx={{
                color: "text.secondary",
                mb: 0.5,
              }}
            >
              Progress
            </Typography>
            <LinearProgress
              variant={showIndeterminate ? "indeterminate" : "determinate"}
              value={showIndeterminate ? undefined : clampedProgressValue}
            />
            <Stack
              direction="row"
              sx={{
                justifyContent: "space-between",
                mt: 0.5,
              }}
            >
              <Typography
                variant="caption"
                sx={{
                  color: "text.secondary",
                }}
              >
                {status ? status.toUpperCase() : "IDLE"}
              </Typography>
              <Typography
                variant="caption"
                sx={{
                  color: "text.secondary",
                }}
              >
                {showIndeterminate
                  ? "Working..."
                  : `${Math.round(clampedProgressValue)}%`}
              </Typography>
            </Stack>
            {progressValue === 999 && (
              <Typography
                variant="caption"
                sx={{
                  color: "text.secondary",
                }}
              >
                This tool does not report incremental progress. Live output is
                shown in logs.
              </Typography>
            )}
          </Box>

          {message && !verbose && (
            <Typography
              variant="body2"
              sx={{
                color: "text.secondary",
              }}
            >
              {message}
            </Typography>
          )}

          <Stack direction={{ xs: "column", sm: "row" }} spacing={2}>
            <FormControlLabel
              control={
                <Switch
                  checked={autoFix}
                  onChange={(event) => setAutoFix(event.target.checked)}
                  disabled={isRunning}
                />
              }
              label="AutoFix"
            />
            <FormControlLabel
              control={
                <Switch
                  checked={force}
                  onChange={(event) => setForce(event.target.checked)}
                  disabled={isRunning}
                />
              }
              label="Force"
            />
            <FormControlLabel
              control={
                <Switch
                  checked={verbose}
                  onChange={(event) => setVerbose(event.target.checked)}
                  disabled={isRunning}
                />
              }
              label="Verbose"
            />
          </Stack>

          {verbose && (
            <Box>
              <Typography
                variant="subtitle2"
                sx={{
                  color: "text.secondary",
                  mb: 0.5,
                }}
              >
                Logs
              </Typography>
              <ReadonlyCommandTerminal
                lines={logs}
                maxHeight={240}
                emptyText="No logs yet."
              />
            </Box>
          )}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} color="secondary" variant="outlined">
          Close
        </Button>
        <Button
          onClick={isRunning ? handleStop : handleStart}
          color={isRunning ? "error" : "primary"}
          variant="contained"
          disabled={
            isStarting ||
            isStopping ||
            !partition?.id ||
            (!isRunning && isSupportLoading) ||
            (!isRunning && !canCheck)
          }
        >
          {isRunning ? "Stop" : "Start"}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
