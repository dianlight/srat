import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import PowerIcon from "@mui/icons-material/Power";
import WarningIcon from "@mui/icons-material/Warning";
import {
  Box,
  Button,
  Card,
  CardContent,
  CardHeader,
  Collapse,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Grid,
  IconButton,
  Stack,
  ToggleButton,
  ToggleButtonGroup,
  Tooltip,
  Typography,
} from "@mui/material";
import { useEffect, useMemo, useState } from "react";
import { Controller, useWatch } from "react-hook-form";
import {
  AutocompleteElement,
  TextFieldElement,
  useForm,
} from "react-hook-form-mui";
import { getCurrentEnv } from "../../../macro/Environment" with {
  type: "macro",
};
import type { Command_type, Disk, HdIdleDevice } from "../../../store/sratApi";
import {
  Enabled,
  usePutApiDiskByDiskIdHdidleConfigMutation,
} from "../../../store/sratApi";

interface HDIdleDiskSettingsProps {
  disk: Disk;
  readOnly?: boolean;
}

type DiskExtended = Disk & {
  hdidle_status?: {
    idle_time?: number;
    command_type?: Command_type;
    power_condition?: number;
    enabled?: Enabled;
  };
};

export function HDIdleDiskSettings({
  disk,
  readOnly = false,
}: HDIdleDiskSettingsProps) {
  const { control, reset, formState, getValues, setValue } = useForm({
    defaultValues: {
      ...disk?.hdidle_device,
    },
  });
  const diskId = disk?.id || "";
  const [saveConfig, { isLoading: isSaving }] =
    usePutApiDiskByDiskIdHdidleConfigMutation();
  const [expanded, setExpanded] = useState(false);
  const [forceDialogOpen, setForceDialogOpen] = useState(false);
  const [pendingEnabled, setPendingEnabled] = useState<Enabled | null>(null);
  const isTestEnv = (globalThis as Record<string, unknown>).__TEST__ === true;
  const [visible, setVisible] = useState(isTestEnv);
  const diskName = disk.model || disk.id || "Unknown";

  // Watch the local enabled toggle to disable/enable the rest of the form
  const enabled = useWatch({ control, name: "enabled" }) as Enabled | undefined;
  const fieldsDisabled = enabled === Enabled.No || readOnly;
  const applyDisabled = !formState.isDirty || isSaving;

  useEffect(() => {
    if (isTestEnv) {
      setVisible(true);
    } else {
      setVisible(getCurrentEnv() !== "production");
    }

    // When disk prop or API config changes, update form values
    const apiValues = disk?.hdidle_device as HdIdleDevice | undefined;
    reset({
      enabled: (apiValues?.enabled as Enabled | undefined) ?? Enabled.Yes,
      idle_time:
        apiValues?.idle_time ??
        (disk as DiskExtended)?.hdidle_status?.idle_time ??
        0,
      command_type:
        apiValues?.command_type ??
        (disk as DiskExtended)?.hdidle_status?.command_type ??
        undefined,
      power_condition:
        apiValues?.power_condition ??
        (disk as DiskExtended)?.hdidle_status?.power_condition ??
        0,
    });
  }, [disk, reset, isTestEnv]);

  // Close accordion if enabled is not Custom
  useEffect(() => {
    if (enabled !== Enabled.Custom) {
      setExpanded(false);
    }
  }, [enabled]);

  // Read HDIdle config snapshot from disk dto when available
  const hdidleStatus = useMemo(() => {
    const s = (disk as DiskExtended)?.hdidle_status as
      | {
          idle_time?: number;
          command_type?: Command_type;
          power_condition?: number;
          enabled?: Enabled;
        }
      | undefined;
    return s;
  }, [disk]);

  const handleExpandChange = () => {
    setExpanded(!expanded);
  };

  const handleApply = async () => {
    if (!diskId) return;
    const values = getValues();
    const payload: HdIdleDevice = {
      // The backend expects the full by-id device path in the payload
      disk_id: diskId,
      device_path: `/dev/disk/by-id/${diskId}`,
      enabled: values.enabled as Enabled,
      idle_time: Number(values.idle_time ?? 0),
      command_type: values.command_type || undefined,
      power_condition: Number(values.power_condition ?? 0),
      force_enabled: disk.hdidle_device?.force_enabled ?? false,
      suggestion_ignored: false,
      supported: disk.hdidle_device?.supported ?? false,
      supports_ata: disk.hdidle_device?.supports_ata ?? false,
      supports_scsi: disk.hdidle_device?.supports_scsi ?? false,
    };
    try {
      await saveConfig({ diskId, hdIdleDevice: payload }).unwrap();
    } catch {
      // No-op; errors should be surfaced by global error UI
    }
  };

  const handleCancel = () => {
    // Restore last loaded API values
    const apiValues = disk?.hdidle_device as HdIdleDevice | undefined;
    reset({
      enabled: (apiValues?.enabled as Enabled | undefined) ?? Enabled.Yes,
      idle_time: apiValues?.idle_time ?? disk?.hdidle_device?.idle_time ?? 0,
      command_type:
        apiValues?.command_type ??
        disk?.hdidle_device?.command_type ??
        undefined,
      power_condition:
        apiValues?.power_condition ?? disk?.hdidle_device?.power_condition ?? 0,
    });
  };

  // Force dialog handlers
  const handleForceDialogOpen = (newValue: Enabled) => {
    setPendingEnabled(newValue);
    setForceDialogOpen(true);
  };

  const handleForceDialogConfirm = () => {
    if (pendingEnabled !== null) {
      setValue("enabled", pendingEnabled, { shouldDirty: true });
      setPendingEnabled(null);
    }
    setForceDialogOpen(false);
  };

  const handleForceDialogCancel = () => {
    setPendingEnabled(null);
    setForceDialogOpen(false);
  };

  if (!disk.hdidle_device?.supported) {
    console.warn("HDIdle not supported on this disk");
    return null;
  }

  const effectiveLoading = false;
  return (
    visible &&
    !effectiveLoading && (
      <>
        <Card>
          <CardHeader
            title="Power Settings ( 🚧 Work In Progress )"
            avatar={
              <IconButton size="small" sx={{ pointerEvents: "none" }}>
                <PowerIcon color="primary" />
              </IconButton>
            }
            action={
              <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                <Tooltip
                  title={
                    <Typography variant="body2">
                      Enable disk-specific override. When Off, fields are
                      read-only.
                    </Typography>
                  }
                >
                  <span>
                    <Controller
                      name="enabled"
                      control={control}
                      render={({ field: { value, onChange } }) => (
                        <ToggleButtonGroup
                          value={value}
                          exclusive
                          size="small"
                          color={value === Enabled.Yes ? "success" : "standard"}
                          disabled={readOnly}
                          onChange={(_, newValue) => {
                            if (newValue === null) return;
                            // Check if enabling HDIdle on a non-rotational disk (not previously force_enabled)
                            const isNonRotational =
                              disk.is_rotational === false;
                            const wasForceEnabled =
                              disk.hdidle_device?.force_enabled === true;
                            const isEnabling =
                              newValue === Enabled.Yes ||
                              newValue === Enabled.Custom;
                            const wasDisabled = value === Enabled.No;

                            if (
                              isNonRotational &&
                              !wasForceEnabled &&
                              isEnabling &&
                              wasDisabled
                            ) {
                              handleForceDialogOpen(newValue as Enabled);
                              return;
                            }

                            onChange(newValue as Enabled);
                            if (newValue !== Enabled.Custom) {
                              handleApply();
                            }
                          }}
                          aria-label="toggle disk override"
                        >
                          <ToggleButton value={Enabled.Yes}>
                            {Enabled.Yes}
                          </ToggleButton>
                          <ToggleButton value={Enabled.Custom}>
                            {Enabled.Custom}
                          </ToggleButton>
                          <ToggleButton value={Enabled.No}>
                            {Enabled.No}
                          </ToggleButton>
                        </ToggleButtonGroup>
                      )}
                    />
                  </span>
                </Tooltip>

                <IconButton
                  onClick={handleExpandChange}
                  disabled={readOnly || enabled !== Enabled.Custom}
                  aria-expanded={expanded}
                  aria-label="show more"
                  sx={{
                    transform: expanded ? "rotate(180deg)" : "rotate(0deg)",
                    transition: "transform 150ms cubic-bezier(0.4, 0, 0.2, 1)",
                  }}
                >
                  <ExpandMoreIcon />
                </IconButton>
              </Box>
            }
          />
          <Collapse in={expanded} timeout="auto" unmountOnExit>
            <CardContent>
              <Grid container spacing={2}>
                <Grid size={12}>
                  <Typography
                    variant="body2"
                    gutterBottom
                    sx={{
                      color: "text.secondary",
                    }}
                  >
                    Configure specific spin-down settings for{" "}
                    <strong>{disk.model || diskName}</strong>. Leave fields at 0
                    or empty to use default settings.
                  </Typography>

                  {hdidleStatus && (
                    <Box
                      sx={{
                        mt: 1,
                        mb: 1,
                        p: 1,
                        backgroundColor: "info.lighter",
                        borderRadius: 1,
                      }}
                    >
                      <Typography
                        variant="caption"
                        sx={{
                          color: "text.secondary",
                        }}
                      >
                        Current config: idle time
                        <strong> {hdidleStatus.idle_time ?? 0}s</strong>,
                        command
                        <strong>
                          {" "}
                          {hdidleStatus.command_type || "default"}
                        </strong>
                        , power condition
                        <strong> {hdidleStatus.power_condition ?? 0}</strong>
                        {hdidleStatus.enabled && (
                          <span>
                            {" "}
                            — enabled: <strong>{hdidleStatus.enabled}</strong>
                          </span>
                        )}
                      </Typography>
                    </Box>
                  )}
                </Grid>

                <Grid size={{ xs: 12, md: 4 }}>
                  <Tooltip
                    title={
                      <Typography variant="body2">
                        Idle time before spinning down this specific disk. Set
                        to 0 to use the default timeout.
                      </Typography>
                    }
                  >
                    <span style={{ display: "inline-block", width: "100%" }}>
                      <TextFieldElement
                        name={`idle_time`}
                        label="Idle Time (seconds)"
                        type="number"
                        control={control}
                        disabled={fieldsDisabled}
                        slotProps={{ htmlInput: { min: 0 } }}
                        size="small"
                        helperText="0 = use default"
                      />
                    </span>
                  </Tooltip>
                </Grid>

                <Grid size={{ xs: 12, md: 4 }}>
                  <Tooltip
                    title={
                      <>
                        <Typography variant="body2">
                          Command type for this disk. Leave empty to use
                          default.
                        </Typography>
                        <Typography variant="body2" sx={{ mt: 1 }}>
                          <strong>SCSI:</strong> For most modern SATA/SAS drives
                        </Typography>
                        <Typography variant="body2">
                          <strong>ATA:</strong> For legacy ATA/IDE drives
                        </Typography>
                      </>
                    }
                  >
                    <span style={{ display: "inline-block", width: "100%" }}>
                      <AutocompleteElement
                        name={`command_type`}
                        label="Command Type"
                        control={control}
                        options={["scsi", "ata"]}
                        autocompleteProps={{
                          size: "small",
                          disabled: fieldsDisabled,
                        }}
                        textFieldProps={{
                          helperText: "Empty = use default",
                        }}
                      />
                    </span>
                  </Tooltip>
                </Grid>

                <Grid size={{ xs: 12, md: 4 }}>
                  <Tooltip
                    title={
                      <Typography variant="body2">
                        SCSI power condition for this disk (0-15). Set to 0 for
                        default behavior.
                      </Typography>
                    }
                  >
                    <span style={{ display: "inline-block", width: "100%" }}>
                      <TextFieldElement
                        name={`power_condition`}
                        label="Power Condition"
                        type="number"
                        control={control}
                        disabled={fieldsDisabled}
                        slotProps={{ htmlInput: { min: 0, max: 15 } }}
                        size="small"
                        helperText="0 = default"
                      />
                    </span>
                  </Tooltip>
                </Grid>

                <Grid size={12}>
                  <Box
                    sx={{
                      mt: 1,
                      p: 1,
                      backgroundColor: "info.lighter",
                      borderRadius: 1,
                    }}
                  >
                    <Typography
                      variant="caption"
                      sx={{
                        color: "text.secondary",
                      }}
                    >
                      <strong>Note:</strong> Device-specific settings override
                      global defaults. Changes take effect after the next
                      service restart or configuration update.
                    </Typography>
                  </Box>
                </Grid>

                <Grid size={12}>
                  <Box
                    sx={{
                      display: "flex",
                      gap: 1,
                      justifyContent: "flex-end",
                      mt: 2,
                    }}
                  >
                    <Tooltip
                      title={
                        formState.isDirty
                          ? "Apply changes"
                          : "No changes to apply"
                      }
                    >
                      <span>
                        <ToggleButton
                          value="apply"
                          disabled={applyDisabled}
                          onClick={handleApply}
                          color="success"
                          size="small"
                        >
                          Apply
                        </ToggleButton>
                      </span>
                    </Tooltip>
                    <Tooltip title="Restore last loaded values">
                      <span>
                        <ToggleButton
                          value="cancel"
                          disabled={isSaving}
                          onClick={handleCancel}
                          size="small"
                        >
                          Cancel
                        </ToggleButton>
                      </span>
                    </Tooltip>
                  </Box>
                </Grid>
              </Grid>
            </CardContent>
          </Collapse>
        </Card>
        {forceDialogOpen && (
          <ForceEnableDialog
            open={forceDialogOpen}
            onClose={handleForceDialogCancel}
            onConfirm={handleForceDialogConfirm}
          />
        )}
      </>
    )
  );
}

// Force enable dialog for non-rotational disks
const ForceEnableDialog = ({
  open,
  onClose,
  onConfirm,
}: {
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
}) => (
  <Dialog
    open={open}
    onClose={onClose}
    maxWidth="sm"
    fullWidth
    aria-labelledby="force-dialog-title"
    aria-describedby="force-dialog-description"
  >
    <DialogTitle id="force-dialog-title">
      <Stack direction="row" spacing={1} sx={{ alignItems: "center" }}>
        <WarningIcon color="warning" />
        <Typography variant="h6">
          Enable HDIdle on a non-rotational disk
        </Typography>
      </Stack>
    </DialogTitle>
    <DialogContent dividers>
      <Typography
        id="force-dialog-description"
        variant="body1"
        sx={{ mt: 1, mb: 2 }}
      >
        You are about to enable HDIdle on a non-rotational disk (SSD/NVMe).
        HDIdle is designed for rotational hard drives and may not provide
        benefits on solid-state storage. Enabling it could cause unnecessary
        wear or have no effect.
      </Typography>
      <Typography variant="body2" color="text.secondary">
        If you are certain you want to proceed, click "Force Enable". The
        setting will be saved with force_enabled=true.
      </Typography>
    </DialogContent>
    <DialogActions sx={{ p: 2, pt: 0 }}>
      <Button onClick={onClose} size="small">
        Cancel
      </Button>
      <Button
        onClick={onConfirm}
        color="warning"
        variant="contained"
        size="small"
      >
        Force Enable
      </Button>
    </DialogActions>
  </Dialog>
);
