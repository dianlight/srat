import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Typography,
} from "@mui/material";
import type { CommandExecutionSnapshot } from "../store/sratApi";
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
    closeToast?.(true);
  };

  return (
    <Box>
      <Typography variant="body2" sx={{ mb: 1 }}>
        Command stderr detected for {commandId}
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

export function CommandOutputDialog({
  open,
  session,
  onClose,
  onDownload,
}: CommandOutputDialogProps) {
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
        <ReadonlyCommandTerminal lines={session?.lines} />
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
