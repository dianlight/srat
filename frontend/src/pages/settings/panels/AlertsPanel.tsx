import ScienceOutlinedIcon from "@mui/icons-material/ScienceOutlined";
import { Stack, Tooltip, Typography } from "@mui/material";
import { useFormContext } from "react-hook-form";
import { useLabFeatures } from "../../../hooks/useLabFeatures";
import type { Settings as ApiSettings } from "../../../store/sratApi";
import { SettingSwitchRow } from "../components/SettingSwitchRow";

type AlertsPanelProps = {
  readOnly: boolean;
};

export function AlertsPanel({ readOnly }: AlertsPanelProps) {
  const { control, watch } = useFormContext<ApiSettings>();
  const commonProps = { control, disabled: readOnly };
  const experimentalLabMode = Boolean(watch("experimental_lab_mode"));
  const { isAvailable: labFeatureAvailable } = useLabFeatures();
  const customComponentLabActive =
    experimentalLabMode && labFeatureAvailable("ha_custom_component");

  const labLabel = (text: string) => (
    <Stack direction="row" spacing={1} sx={{ alignItems: "center" }}>
      <Typography component="span">{text}</Typography>
      <ScienceOutlinedIcon color="warning" fontSize="small" />
    </Stack>
  );

  return (
    <Stack spacing={3}>
      <Typography variant="body2" sx={{ color: "text.secondary" }}>
        Control which alerts SRAT raises in the dashboard and in Home Assistant.
        Ignoring an alert suppresses it permanently until you re-enable it here
        or re-arm it from the dashboard.
      </Typography>

      {/* Protected mode */}
      <Tooltip
        title={
          <>
            <Typography variant="h6" component="div">
              Protected mode alert
            </Typography>
            <Typography variant="body2">
              Warns while the add-on runs in protected mode, when no disks can
              be mounted. Ignoring this alert hides it everywhere, including
              Home Assistant repairs.
            </Typography>
          </>
        }
      >
        <SettingSwitchRow
          ariaLabel="Protected mode alert"
          label="Protected mode alert"
          name="alert_protected_mode"
          helperText="Raised as an ignorable Home Assistant repair issue while protected mode is on."
          {...commonProps}
        />
      </Tooltip>

      {/* Addon config changed */}
      <Tooltip
        title={
          <>
            <Typography variant="h6" component="div">
              Add-on configuration changed alert
            </Typography>
            <Typography variant="body2">
              Warns when the add-on options file is modified outside of SRAT.
            </Typography>
          </>
        }
      >
        <SettingSwitchRow
          ariaLabel="Add-on configuration changed alert"
          label="Add-on configuration changed alert"
          name="alert_addon_config_changed"
          {...commonProps}
        />
      </Tooltip>

      {/* Custom component alerts — ALPHA lab feature: only in dev/prerelease
          builds (registry drops it in production), AND behind Lab Mode. */}
      {customComponentLabActive ? (
        <Tooltip
          title={
            <>
              <Typography variant="h6" component="div">
                Custom component alerts (Lab)
              </Typography>
              <Typography variant="body2">
                Warns when the Home Assistant custom component needs a restart
                or is missing. Hidden unless lab mode is enabled.
              </Typography>
            </>
          }
        >
          <SettingSwitchRow
            ariaLabel="Custom component alerts"
            label={labLabel("Custom component alerts")}
            name="alert_custom_component"
            {...commonProps}
          />
        </Tooltip>
      ) : null}
    </Stack>
  );
}
