import ScienceOutlinedIcon from "@mui/icons-material/ScienceOutlined";
import { Stack, Tooltip, Typography } from "@mui/material";
import { useFormContext } from "react-hook-form";
import { useLabFeatures } from "../../../hooks/useLabFeatures";
import {
  type Settings as ApiSettings,
  type SystemCapabilities,
  useGetApiCapabilitiesQuery,
  useGetApiSettingsHomeassistantCustomComponentStatusQuery,
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
  const { isAvailable: labFeatureAvailable } = useLabFeatures();
  const customComponentLabActive =
    experimentalLabMode && labFeatureAvailable("ha_custom_component");
  // Skip the status query when the alpha lab surface is hidden so the UI
  // does not spam a 403-gated endpoint while lab mode is off or in
  // production builds.
  const { data: componentStatus } =
    useGetApiSettingsHomeassistantCustomComponentStatusQuery(undefined, {
      skip: !customComponentLabActive,
    });
  const mdnsEnabled = Boolean(watch("mdns_registration"));
  const isComponentConnected = Boolean(
    componentStatus &&
      "connected" in componentStatus &&
      componentStatus.connected,
  );

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

      {/* Use NFS (remote env only) — beta lab feature, reactive to the
          in-form Lab Mode switch (matches backend: beta available ⟺
          experimental_lab_mode). */}
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

      {/* Custom Component Panel — ALPHA lab feature: only in dev/prerelease
          builds (registry drops it in production), AND behind Lab Mode. */}
      {customComponentLabActive ? (
        <HomeAssistantCustomComponentPanel readOnly={readOnly} />
      ) : null}

      {/* Use Home Assistant mDNS Proxy (implementation choice) */}
      <Tooltip
        title={
          <>
            <Typography variant="h6" component="div">
              Use Home Assistant mDNS Proxy
            </Typography>
            <Typography variant="body2">
              Announce this Samba server on the local network via Home Assistant
              using mDNS (Zeroconf). Requires the Samba mDNS Announce master
              switch to be enabled and an active Home Assistant add-on
              connection.
            </Typography>
            {!mdnsEnabled && (
              <Typography
                variant="body2"
                sx={{ mt: 1, color: "warning.light" }}
              >
                <strong>Disabled:</strong> Samba mDNS Announce is off. Enable it
                in the General section first.
              </Typography>
            )}
            {!isComponentConnected && (
              <Typography
                variant="body2"
                sx={{ mt: 1, color: "warning.light" }}
              >
                <strong>Not available:</strong> Home Assistant custom component
                is not connected.
              </Typography>
            )}
          </>
        }
      >
        <SettingSwitchRow
          ariaLabel="Use Home Assistant mDNS Proxy"
          label="Use Home Assistant mDNS Proxy"
          name="use_component_mdns_proxy"
          control={control}
          disabled={readOnly || !isComponentConnected || !mdnsEnabled}
        />
      </Tooltip>
    </Stack>
  );
}
