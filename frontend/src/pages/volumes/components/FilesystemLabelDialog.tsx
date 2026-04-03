import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  Stack,
  TextField,
  Typography,
} from "@mui/material";
import { useEffect, useMemo, useState } from "react";
import { toast } from "react-toastify";
import {
  type ErrorModel,
  type Partition,
  useGetApiFilesystemSupportQuery,
  usePutApiFilesystemLabelMutation,
} from "../../../store/sratApi";
import { decodeEscapeSequence } from "../utils";

interface FilesystemLabelDialogProps {
  open: boolean;
  partition?: Partition;
  onClose: () => void;
}

export function FilesystemLabelDialog({
  open,
  partition,
  onClose,
}: FilesystemLabelDialogProps) {
  const [label, setLabel] = useState("");
  const currentLabel = useMemo(
    () => decodeEscapeSequence(partition?.name ?? ""),
    [partition?.name],
  );
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
  const [setLabelMutation, { isLoading: isSaving }] =
    usePutApiFilesystemLabelMutation();

  useEffect(() => {
    if (!open) {
      return;
    }
    setLabel(currentLabel);
  }, [currentLabel, open]);

  const support = useMemo(() => {
    if (!supportData || !("canSetLabel" in supportData)) {
      return null;
    }
    return supportData;
  }, [supportData]);

  const canSetLabel = useMemo(() => {
    if (support?.canSetLabel !== undefined) {
      return support.canSetLabel;
    }
    if (partition?.filesystem_info?.support?.canSetLabel !== undefined) {
      return partition.filesystem_info.support.canSetLabel;
    }
    return true;
  }, [partition?.filesystem_info?.support?.canSetLabel, support?.canSetLabel]);

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
      "Failed to verify label support."
    );
  }, [isSupportError, supportError]);

  const handleSetLabel = async () => {
    if (!partition?.id) {
      toast.error("Partition not selected.");
      return;
    }
    if (!label.trim()) {
      toast.error("Please enter a label.");
      return;
    }

    try {
      await setLabelMutation({
        setPartitionLabelInput: {
          partitionId: partition.id,
          label: label.trim(),
        },
      }).unwrap();
      toast.success("Filesystem label updated.");
      onClose();
    } catch (err: unknown) {
      const typedErr = err as {
        data?: { detail?: string; title?: string };
        message?: string;
      };
      const errorMsg =
        typedErr?.data?.detail ||
        typedErr?.data?.title ||
        typedErr?.message ||
        "Failed to update filesystem label.";
      toast.error(errorMsg);
    }
  };

  const partitionLabel = decodeEscapeSequence(
    partition?.name || partition?.id || "Selected partition",
  );

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>Set Filesystem Label: {partitionLabel}</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ pt: 1 }}>
          <DialogContentText>
            Set a new filesystem label for this partition.
          </DialogContentText>

          {!isSupportLoading && !canSetLabel && (
            <Alert severity="warning">
              <Typography variant="body2" sx={{ fontWeight: 600 }}>
                Label tools are not available for this filesystem on the current
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

          <TextField
            label="Label"
            value={label}
            onChange={(event) => setLabel(event.target.value)}
            autoFocus
            fullWidth
            disabled={isSaving}
            placeholder="Enter new label"
          />
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} color="secondary" variant="outlined">
          Cancel
        </Button>
        <Button
          onClick={handleSetLabel}
          variant="contained"
          disabled={
            isSaving ||
            !partition?.id ||
            label.trim().length === 0 ||
            isSupportLoading ||
            !canSetLabel
          }
        >
          Set Label
        </Button>
      </DialogActions>
    </Dialog>
  );
}
