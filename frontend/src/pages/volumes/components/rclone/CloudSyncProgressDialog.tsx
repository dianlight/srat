import CloseIcon from "@mui/icons-material/Close";
import StopIcon from "@mui/icons-material/Stop";
import {
  Alert,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  LinearProgress,
  Stack,
  Typography,
} from "@mui/material";
import { useEffect, useMemo, useRef, useState } from "react";
import { toast } from "react-toastify";
import {
  createTerminalLines,
  ReadonlyCommandTerminal,
} from "../../../../components/ReadonlyCommandTerminal";
import type {
  CommandOutputLineSnapshot,
  Direction,
  RcloneTask,
} from "../../../../store/sratApi";
import { usePostApiRcloneLinkByTargetKindAndTargetIdAbortMutation } from "../../../../store/sratApi";
import { useGetServerEventsQuery } from "../../../../store/wsApi";
import { extractApiErrorMessage } from "./apiErrors";

export interface CloudSyncProgressDialogProps {
  open: boolean;
  onClose: () => void;
  direction: Direction | null;
  /** True when the observed run was started with dry_run enabled. */
  dryRun?: boolean;
  targetKind: string;
  targetId: string;
  targetLabel: string;
  /** Test seam: forces the observed task instead of reading WS events. */
  taskOverride?: RcloneTask;
}

const DIRECTION_LABELS: Record<string, string> = {
  push: "Pushing local → remote",
  pull: "Pulling remote → local",
  bidi: "Syncing both ways",
};

function isRunningStatus(status?: string): boolean {
  return status === "start" || status === "running";
}

/**
 * Live progress dialog for a running rclone sync job. Mirrors the
 * FilesystemCheckDialog pattern: it observes `rclone_task` WebSocket events
 * matching the target and renders a progress bar plus a terminal-style log
 * built from the cumulative `notes` channel.
 */
export function CloudSyncProgressDialog({
  open,
  onClose,
  direction,
  dryRun = false,
  targetKind,
  targetId,
  targetLabel,
  taskOverride,
}: CloudSyncProgressDialogProps) {
  const { data: eventData } = useGetServerEventsQuery();
  const [abortSync, { isLoading: isAborting }] =
    usePostApiRcloneLinkByTargetKindAndTargetIdAbortMutation();

  const lastNotesRef = useRef<string[]>([]);
  const nextLogTimestampRef = useRef<number>(Date.now());
  const [logs, setLogs] = useState<CommandOutputLineSnapshot[]>([]);

  const task: RcloneTask | undefined = useMemo(() => {
    if (taskOverride) {
      return taskOverride;
    }
    const candidate = eventData?.rclone_task;
    if (
      candidate &&
      candidate.target_kind === targetKind &&
      candidate.target_id === targetId
    ) {
      return candidate;
    }
    return undefined;
  }, [taskOverride, eventData, targetKind, targetId]);

  // Reset the log buffer every time the dialog opens for a new run.
  useEffect(() => {
    if (open) {
      lastNotesRef.current = [];
      nextLogTimestampRef.current = Date.now();
      setLogs([]);
    }
  }, [open]);

  // Route incoming notes into terminal lines with prefix de-duplication:
  // the backend re-sends the whole cumulative notes list on each event.
  useEffect(() => {
    const notes = task?.notes ?? [];
    const previous = lastNotesRef.current;
    if (
      previous.length > 0 &&
      notes.length >= previous.length &&
      previous.every((line, index) => line === notes[index])
    ) {
      return;
    }
    let start = 0;
    while (
      start < previous.length &&
      start < notes.length &&
      previous[start] === notes[start]
    ) {
      start++;
    }
    const fresh = notes.slice(start);
    if (fresh.length === 0) {
      return;
    }
    lastNotesRef.current = notes;
    setLogs((existing) => [
      ...existing,
      ...createTerminalLines(fresh, "info", nextLogTimestampRef.current++),
    ]);
  }, [task?.notes]);

  const running = isRunningStatus(task?.status);
  const failed = task?.status === "failure";
  const succeeded = task?.status === "success";
  const progress = task?.progress ?? -1;
  const showIndeterminate =
    Boolean(running) && (progress < 0 || progress === 999);

  const handleAbort = async () => {
    try {
      await abortSync({ targetKind, targetId }).unwrap();
      toast.info("Abort requested; the job will stop shortly.");
    } catch (err) {
      toast.error(extractApiErrorMessage(err, "Failed to request abort"));
    }
  };

  const directionLabel = direction ? DIRECTION_LABELS[direction] : "Cloud sync";

  return (
    <Dialog
      open={open}
      onClose={running ? () => {} : onClose}
      fullWidth
      maxWidth="sm"
    >
      <DialogTitle
        sx={{ display: "flex", alignItems: "center", pr: 1 }}
        component="div"
      >
        <Typography variant="h6" component="span" sx={{ flexGrow: 1 }}>
          {`${directionLabel}: ${targetLabel}`}
          {dryRun && (
            <Chip
              size="small"
              color="info"
              label="Dry run"
              sx={{ ml: 1, verticalAlign: "middle" }}
            />
          )}
        </Typography>
        {!running && (
          <IconButton aria-label="close" onClick={onClose} size="small">
            <CloseIcon />
          </IconButton>
        )}
      </DialogTitle>
      <DialogContent>
        <Stack spacing={2}>
          {running && (
            <>
              <LinearProgress
                variant={showIndeterminate ? "indeterminate" : "determinate"}
                value={
                  showIndeterminate
                    ? undefined
                    : Math.min(100, Math.max(0, progress))
                }
              />
              <Stack
                direction="row"
                spacing={2}
                sx={{ alignItems: "baseline" }}
              >
                <Typography variant="caption" color="text.secondary">
                  {(task?.status ?? "").toUpperCase()}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  {!showIndeterminate && `${Math.round(progress)}%`}
                </Typography>
                {task?.message && (
                  <Typography variant="caption" color="text.secondary" noWrap>
                    {task.message}
                  </Typography>
                )}
              </Stack>
            </>
          )}
          {failed && (
            <Alert severity="error">
              {task?.error || task?.message || "The sync operation failed."}
            </Alert>
          )}
          {succeeded && (
            <Alert severity="success">Sync completed successfully.</Alert>
          )}
          <ReadonlyCommandTerminal
            lines={logs}
            maxHeight={240}
            emptyText="No logs yet."
          />
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
        {running && (
          <Button
            variant="contained"
            color="error"
            startIcon={<StopIcon />}
            disabled={isAborting}
            onClick={() => {
              void handleAbort();
            }}
          >
            Abort
          </Button>
        )}
      </DialogActions>
    </Dialog>
  );
}
