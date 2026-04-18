import { Stack, Tooltip, Typography } from "@mui/material";
import { useFormContext } from "react-hook-form";
import { SwitchElement } from "react-hook-form-mui";
import { getCurrentEnv } from "../../../macro/Environment" with {
  type: "macro",
};
import {
  type Settings as ApiSettings,
  type SystemCapabilities,
  useGetApiCapabilitiesQuery,
} from "../../../store/sratApi";
import { HomeAssistantCustomComponentPanel } from "../HomeAssistantCustomComponentPanel";

type HomeAssistantPanelProps = {
  readOnly: boolean;
};

export function HomeAssistantPanel({ readOnly }: HomeAssistantPanelProps) {
  const { control } = useFormContext<ApiSettings>();
  const { data: capabilities } = useGetApiCapabilitiesQuery();
  const commonProps = { control, disabled: readOnly };

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
        <span style={{ display: "inline-block", width: "100%" }}>
          <SwitchElement
            switchProps={{ "aria-label": "Export Stats to HA", size: "small" }}
            sx={{ display: "flex" }}
            name="export_stats_to_ha"
            label="Export Stats to HA"
            labelPlacement="start"
            {...commonProps}
          />
        </span>
      </Tooltip>

      {/* Use NFS (remote env only) */}
      {getCurrentEnv() === "remote" ? (
        <Tooltip
          title={
            <>
              <Typography variant="h6" component="div">
                Use NFS for Home Assistant Integration (Experimental)
              </Typography>
              <Typography variant="body2">
                If enabled, Home Assistant will mount shares using NFS instead
                of SMB/CIFS. This can be more efficient but is currently
                experimental.
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
          <span style={{ display: "inline-block", width: "100%" }}>
            <SwitchElement
              disabled={!(capabilities as SystemCapabilities)?.support_nfs}
              switchProps={{ "aria-label": "Use NFS for HA", size: "small" }}
              sx={{ display: "flex" }}
              name="ha_use_nfs"
              label={
                <>
                  Use NFS for HA{" "}
                  <Typography
                    component="span"
                    variant="caption"
                    sx={{ color: "warning.main", ml: 1 }}
                  >
                    (Experimental)
                  </Typography>
                </>
              }
              labelPlacement="start"
              control={control}
            />
          </span>
        </Tooltip>
      ) : null}

      {/* Custom Component Panel (remote env only) */}
      {getCurrentEnv() === "remote" ? (
        <HomeAssistantCustomComponentPanel readOnly={readOnly} />
      ) : null}
    </Stack>
  );
}
