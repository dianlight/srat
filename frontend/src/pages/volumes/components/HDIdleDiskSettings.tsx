import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import PowerIcon from "@mui/icons-material/Power";
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  CardHeader,
  Collapse,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  Grid,
  IconButton,
  ToggleButton,
  ToggleButtonGroup,
  Tooltip,
  Typography,
} from "@mui/material";
import { useEffect, useState } from "react";
import { Controller, useWatch } from "react-hook-form";
import {
  AutocompleteElement,
  TextFieldElement,
  useForm,
} from "react-hook-form-mui";
import { useLabMode } from "../../../hooks/useLabMode";
import type { Disk, HdIdleDevice } from "../../../store/sratApi";
import {
  Command_type,
  Enabled,
  usePutApiDiskByDiskIdHdidleConfigMutation,
} from "../../../store/sratApi";

const VALID_COMMAND_TYPES = new Set<string>(Object.values(Command_type));

interface HDIdleDiskSettingsProps {
  disk: Disk;
  readOnly?: boolean;
}

interface FormShape {
  enabled: Enabled;
  idle_time: number;
  command_type?: string;
  power_condition: number;
  force_enabled: boolean;
}

/** Single source of truth for form defaults derived from the disk prop. */
function getDiskFormDefaults(disk: Disk): FormShape {
  return {
    enabled: (disk?.hdidle_device?.enabled as Enabled) ?? Enabled.Yes,
    idle_time: disk?.hdidle_device?.idle_time ?? 0,
    command_type: disk?.hdidle_device?.command_type ?? undefined,
    power_condition: disk?.hdidle_device?.power_condition ?? 0,
    force_enabled: disk?.hdidle_device?.force_enabled ?? false,
  };
}

/**
 * Per-disk HDIdle settings card.
 *
 * Visibility rules (post per-disk-only refactor):
 *   - Lab Mode must be on (otherwise the entire HDIdle subsystem is hidden)
 *   - The disk must report HDIdle support (disk.hdidle_device.supported)
 *
 * Force-enable behaviour: when the user toggles Yes/Custom on a non-rotational
 * disk (is_rotational !== true — covers SSD, NVMe, and unknown), a confirm
 * dialog opens. Confirming sets `force_enabled=true` in the PUT body so the
 * backend's 409-on-non-rotational guard accepts the request.
 */
export function HDIdleDiskSettings({
  disk,
  readOnly = false,
}: HDIdleDiskSettingsProps) {
  const { control, reset, formState, getValues, setValue } = useForm<FormShape>(
    {
      defaultValues: getDiskFormDefaults(disk),
    },
  );
  const { labMode, isLoading: isLoadingLabMode } = useLabMode();
  const diskId = disk?.id || "";
  const [saveConfig, { isLoading: isSaving }] =
    usePutApiDiskByDiskIdHdidleConfigMutation();
  const [expanded, setExpanded] = useState(false);
  const [forceDialogOpen, setForceDialogOpen] = useState(false);
  // pending toggle target while the force-confirm dialog is open
  const [pendingEnabled, setPendingEnabled] = useState<Enabled | null>(null);
  const diskName = disk.model || disk.id || "Unknown";
  const isRotational = disk.is_rotational === true; // strict — null/undefined = unknown = treat as non-rotational

  const enabled = useWatch({ control, name: "enabled" }) as Enabled | undefined;
  // Reactive subscription so the force-enable warning Alert re-renders
  // immediately after handleForceConfirm fires (fix #8: getValues is a
  // snapshot and does not subscribe).
  const forceEnabled = useWatch({ control, name: "force_enabled" }) as boolean;
  const fieldsDisabled = enabled === Enabled.No || readOnly;

  // Refresh form when the parent disk prop changes (e.g. cache invalidation).
  useEffect(() => {
    reset(getDiskFormDefaults(disk));
  }, [disk, reset]);

  // Auto-collapse the accordion when leaving Custom mode.
  useEffect(() => {
    if (enabled !== Enabled.Custom) {
      setExpanded(false);
    }
  }, [enabled]);

  const handleExpandChange = () => setExpanded((prev) => !prev);

  /** Intercept the toggle change. On a non-rotational disk, opening Yes/Custom
   * requires explicit force confirmation. The toggle change is committed only
   * after the dialog resolves. */
  const handleToggleChange = (newValue: Enabled | null) => {
    if (newValue === null) return;
    const turningOn = newValue === Enabled.Yes || newValue === Enabled.Custom;
    const alreadyForced = getValues("force_enabled") === true;
    if (turningOn && !isRotational && !alreadyForced) {
      setPendingEnabled(newValue);
      setForceDialogOpen(true);
      return;
    }
    setValue("enabled", newValue, { shouldDirty: true });
  };

  const handleForceConfirm = () => {
    if (pendingEnabled !== null) {
      setValue("force_enabled", true, { shouldDirty: true });
      setValue("enabled", pendingEnabled, { shouldDirty: true });
    }
    setPendingEnabled(null);
    setForceDialogOpen(false);
  };

  const handleForceCancel = () => {
    setPendingEnabled(null);
    setForceDialogOpen(false);
  };

  const handleApply = async () => {
    if (!diskId) return;
    const values = getValues();
    const payload: HdIdleDevice = {
      // disk_id is the only routing key. The backend resolves device_path
      // itself (Phase 2.3 ResolveDevicePath) and stamps it into the response.
      disk_id: diskId,
      enabled: values.enabled,
      idle_time: Number(values.idle_time ?? 0),
      // Only pass command_type when it is a known enum value; discard anything
      // else (empty string from clearing the autocomplete, or a future value
      // the enum hasn't defined yet).
      command_type:
        values.command_type && VALID_COMMAND_TYPES.has(values.command_type)
          ? (values.command_type as HdIdleDevice["command_type"])
          : undefined,
      power_condition: Number(values.power_condition ?? 0),
      force_enabled: values.force_enabled,
      // Server-managed fields — included to satisfy the required TypeScript
      // type after omitempty removal. The PUT handler ignores these.
      suggestion_ignored: false,
      supported: false,
      supports_ata: false,
      supports_scsi: false,
    };
    try {
      await saveConfig({ diskId, hdIdleDevice: payload }).unwrap();
    } catch (err) {
      console.warn("[HDIdleDiskSettings] save failed:", err);
    }
  };

  const handleCancel = () => {
    reset(getDiskFormDefaults(disk));
  };

  if (!disk.hdidle_device?.supported) return null;
  if (isLoadingLabMode) return null;
  if (!labMode) return null;

  return (
    <>
      <Card>
        <CardHeader
          title="Power Settings"
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
                    Enable HDIdle for this disk. "Custom" exposes per-disk idle
                    time and command overrides. "No" disables monitoring.
                  </Typography>
                }
              >
                <span>
                  <Controller
                    name="enabled"
                    control={control}
                    render={({ field: { value } }) => (
                      <ToggleButtonGroup
                        value={value}
                        exclusive
                        size="small"
                        color={value === Enabled.Yes ? "success" : "standard"}
                        onChange={(_, newValue) =>
                          handleToggleChange(newValue as Enabled | null)
                        }
                        disabled={readOnly}
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
                  sx={{ color: "text.secondary" }}
                >
                  Configure spin-down for <strong>{diskName}</strong>. Leave
                  fields at 0 or empty to use sensible defaults (60s idle, SCSI
                  command, power condition 0).
                </Typography>
              </Grid>

              <Grid size={{ xs: 12, md: 4 }}>
                <Tooltip
                  title={
                    <Typography variant="body2">
                      Idle time before spinning down this disk (seconds). 0 uses
                      the default timeout.
                    </Typography>
                  }
                >
                  <span style={{ display: "inline-block", width: "100%" }}>
                    <TextFieldElement
                      name="idle_time"
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
                        Command type. Leave empty to use the recommended one.
                      </Typography>
                      <Typography variant="body2" sx={{ mt: 1 }}>
                        <strong>SCSI:</strong> most modern SATA/SAS drives
                      </Typography>
                      <Typography variant="body2">
                        <strong>ATA:</strong> legacy ATA/IDE drives
                      </Typography>
                    </>
                  }
                >
                  <span style={{ display: "inline-block", width: "100%" }}>
                    <AutocompleteElement
                      name="command_type"
                      label="Command Type"
                      control={control}
                      options={["scsi", "ata"]}
                      autocompleteProps={{
                        size: "small",
                        disabled: fieldsDisabled,
                      }}
                      textFieldProps={{ helperText: "Empty = use default" }}
                    />
                  </span>
                </Tooltip>
              </Grid>

              <Grid size={{ xs: 12, md: 4 }}>
                <Tooltip
                  title={
                    <Typography variant="body2">
                      SCSI power condition (0-15). 0 is the default.
                    </Typography>
                  }
                >
                  <span style={{ display: "inline-block", width: "100%" }}>
                    <TextFieldElement
                      name="power_condition"
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
                        disabled={readOnly || !formState.isDirty || isSaving}
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
        {/* Force-enable warning rendered outside the Collapse so it stays
            visible on the Enabled.Yes path (which auto-collapses the accordion,
            hiding anything inside Collapse). Uses the reactive forceEnabled
            watch instead of getValues() snapshot. */}
        {!isRotational && forceEnabled && (
          <Alert severity="warning" sx={{ mx: 2, mb: 2 }}>
            HDIdle is force-enabled on a non-rotational disk. Spin-down commands
            have no effect on SSD/NVMe and may cause wear or errors on some
            controllers.
          </Alert>
        )}
      </Card>

      <Dialog open={forceDialogOpen} onClose={handleForceCancel}>
        <DialogTitle>Enable HDIdle on a non-rotational disk?</DialogTitle>
        <DialogContent>
          <DialogContentText>
            This disk does not appear to be a rotational HDD (SSD/NVMe or USB
            enclosure that hides the rotational flag). Spin-down commands are
            <strong> meaningless on solid-state media</strong> and may cause
            controller errors on some bridges.
          </DialogContentText>
          <DialogContentText sx={{ mt: 2 }}>
            If you proceed, the per-disk record will be persisted with{" "}
            <code>force_enabled=true</code> so future loads of this card skip
            this warning.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleForceCancel}>Cancel</Button>
          <Button
            onClick={handleForceConfirm}
            color="warning"
            variant="contained"
          >
            Force enable
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}
