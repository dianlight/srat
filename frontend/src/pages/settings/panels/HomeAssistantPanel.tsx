import ScienceOutlinedIcon from "@mui/icons-material/ScienceOutlined";
import { Stack, Tooltip, Typography } from "@mui/material";
import { useFormContext } from "react-hook-form";
import {
  type Settings as ApiSettings,
  type SystemCapabilities,
  useGetApiCapabilitiesQuery,
} from "../../../store/sratApi";
import { SettingSwitchRow } from "../components/SettingSwitchRow";
import { HomeAssistantCustomComponentPanel } from "../HomeAssistantCustomComponentPanel";

type HomeAssistantPanelProps = {
  readOnly: boolean;
};

export function HomeAssistantPanel({ readOnly }: HomeAssistantPanelProps) {
  const { control, watch } = useFormContext<ApiSettings>();
  const { data: capabilities } = useGetApiCapabilitiesQuery();
  const commonProps = { control, disabled: readOnly };
  const experimentalLabMode = Boolean(watch("experimental_lab_mode"));

  const labLabel = (text: string) => (
    <Stack direction="row" spacing={1} sx={{ alignItems: "center" }}>
      <Typography component="span">{text}</Typography>
      <ScienceOutlinedIcon color="warning" fontSize="small" />
    </Stack>
  );

  return (
    <Stack spacing={3}>
      {/* Export Stats to HA */}
      <Tooltip
        title={
          <>
            <Typography variant="h6" component="div">
              Export stats to Home Assistant
            </Typography>
            <Typography variant="body2">
              If enabled, the status of disks, volumes and the server will be
              transmitted to Home Assistant.
            </Typography>
          </>
        }
      >
        <SettingSwitchRow
          ariaLabel="Export Stats to HA"
          label="Export Stats to HA"
          name="export_stats_to_ha"
          {...commonProps}
        />
      </Tooltip>

      {/* Use NFS (remote env only) */}
      {experimentalLabMode ? (
        <Tooltip
          title={
            <>
              <Typography variant="h6" component="div">
                Use NFS for Home Assistant Integration (Lab)
              </Typography>
              <Typography variant="body2">
                If enabled, Home Assistant will mount shares using NFS instead
                of SMB/CIFS. This can be more efficient but is currently
                considered a lab feature.
              </Typography>
              {!(
                (capabilities as SystemCapabilities)?.support_nfs ?? false
              ) && (
                <Typography
                  variant="body2"
                  sx={{ mt: 1, color: "warning.light" }}
                >
                  <strong>Not available:</strong> NFS support is not detected on
                  this system.
                </Typography>
              )}
            </>
          }
        >
          <SettingSwitchRow
            ariaLabel="Use NFS for HA"
            control={control}
            disabled={!(capabilities as SystemCapabilities)?.support_nfs}
            label={labLabel("Use NFS for HA")}
            name="ha_use_nfs"
          />
        </Tooltip>
      ) : null}

      {/* Custom Component Panel (remote env only) */}
      {experimentalLabMode ? (
        <HomeAssistantCustomComponentPanel readOnly={readOnly} />
      ) : null}
    </Stack>
  );
}
