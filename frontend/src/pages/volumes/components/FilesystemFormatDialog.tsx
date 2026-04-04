import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  FormControlLabel,
  MenuItem,
  Stack,
  Switch,
  TextField,
  Typography,
} from "@mui/material";
import { useEffect, useMemo, useState } from "react";
import { toast } from "react-toastify";
import {
  type ErrorModel,
  type FilesystemsInfo,
  type Partition,
  useGetApiFilesystemSupportQuery,
  useGetApiFilesystemsQuery,
  usePostApiFilesystemFormatMutation,
} from "../../../store/sratApi";
import { decodeEscapeSequence, getFilesystemLabelValidation } from "../utils";

interface FilesystemFormatDialogProps {
  open: boolean;
  partition?: Partition;
  onClose: () => void;
}

interface FilesystemSupportWithLabelRule {
  alpinePackage?: string;
  canFormat?: boolean;
  labelRule?: string | null;
  missingTools?: string[] | null;
}

export function FilesystemFormatDialog({
  open,
  partition,
  onClose,
}: FilesystemFormatDialogProps) {
  const [filesystemType, setFilesystemType] = useState("ext4");
  const [label, setLabel] = useState("");
  const [force, setForce] = useState(false);

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

  useEffect(() => {
    if (!open) {
      return;
    }
    setFilesystemType(partition?.fs_type?.trim() || "ext4");
    setLabel("");
    setForce(false);
  }, [open, partition?.fs_type]);

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

    try {
      await formatMutation({
        formatPartitionInput: {
          partitionId: partition.id,
          filesystemType: filesystemType.trim(),
          label: label.trim(),
          force,
        },
      }).unwrap();
      toast.info("Format operation started.");
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
        "Failed to start format operation.";
      toast.error(errorMsg);
    }
  };

  const partitionLabel = decodeEscapeSequence(
    partition?.name || partition?.id || "Selected partition",
  );

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>Format Partition: {partitionLabel}</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ pt: 1 }}>
          <DialogContentText>
            Format this partition using the selected filesystem type.
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

          <TextField
            label="Filesystem type"
            value={dropdownFilesystemType}
            onChange={(event) => setFilesystemType(event.target.value)}
            fullWidth
            disabled={isFormatting || isFilesystemsLoading}
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
            disabled={isFormatting}
            error={showLabelError}
            helperText={labelValidation.helperText}
          />

          <FormControlLabel
            control={
              <Switch
                checked={force}
                onChange={(event) => setForce(event.target.checked)}
                disabled={isFormatting}
              />
            }
            label="Force"
          />
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} color="secondary" variant="outlined">
          Cancel
        </Button>
        <Button
          onClick={handleFormat}
          color="error"
          variant="contained"
          disabled={
            isFormatting ||
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
          Format
        </Button>
      </DialogActions>
    </Dialog>
  );
}
