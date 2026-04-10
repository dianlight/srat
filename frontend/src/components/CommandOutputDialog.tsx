import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Typography,
} from "@mui/material";
import type {
  CommandExecutionSnapshot,
  CommandOutputLineSnapshot,
} from "../store/sratApi";
import { ReadonlyCommandTerminal } from "./ReadonlyCommandTerminal";

export interface CommandOutputToastContentProps {
  closeToast?: (reason?: boolean | string) => void;
  commandId: string;
  onOpenOutput: () => void;
}

export function CommandOutputToastContent({
  closeToast,
  commandId,
  onOpenOutput,
}: CommandOutputToastContentProps) {
  const handleOpenOutput = () => {
    onOpenOutput();

    if (!closeToast) {
      return;
    }

    try {
      closeToast();
    } catch {
      // NotificationCenter can re-render the stored toast content outside the
      // original toast lifecycle, leaving a stale closeToast callback behind.
      // Opening the output dialog should still succeed in that case.
    }
  };

  return (
    <Box>
      <Typography variant="body2" sx={{ mb: 1 }}>
        Command failed for {commandId}
      </Typography>
      <Button size="small" variant="outlined" onClick={handleOpenOutput}>
        Open Output
      </Button>
    </Box>
  );
}

interface CommandOutputDialogProps {
  open: boolean;
  session?: CommandExecutionSnapshot;
  onClose: () => void;
  onDownload: () => void;
}

export function getCommandOutputLines(
  session?: CommandExecutionSnapshot,
): CommandOutputLineSnapshot[] {
  const lines = [...(session?.lines ?? [])];
  const errorText = session?.error?.trim();

  if (!errorText) {
    return lines;
  }

  const alreadyIncluded = lines.some(
    (line) => line.channel === "stderr" && line.line.trim() === errorText,
  );

  if (alreadyIncluded) {
    return lines;
  }

  return [
    ...lines,
    {
      channel: "stderr",
      line: errorText,
      timestamp: session?.finished_at ?? session?.started_at ?? Date.now(),
    },
  ];
}

export function CommandOutputDialog({
  open,
  session,
  onClose,
  onDownload,
}: CommandOutputDialogProps) {
  const displayLines = getCommandOutputLines(session);
  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>
        Command Output: {session?.label ?? session?.command_id ?? "Unknown"}
      </DialogTitle>
      <DialogContent dividers>
        <Typography variant="body2" sx={{ mb: 1 }}>
          Execution: {session?.execution_id}
        </Typography>
        <Typography variant="body2" sx={{ mb: 2 }}>
          Status:{" "}
          {session?.running
            ? "Running"
            : `${session?.success ? "Success" : "Failed"} (exit ${session?.exit_code ?? "n/a"})`}
        </Typography>
        <ReadonlyCommandTerminal lines={displayLines} />
      </DialogContent>
      <DialogActions>
        <Button onClick={onDownload} variant="outlined">
          Download
        </Button>
        <Button onClick={onClose} variant="contained">
          Close
        </Button>
      </DialogActions>
    </Dialog>
  );
}
