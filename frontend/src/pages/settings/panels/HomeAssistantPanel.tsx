import ScienceOutlinedIcon from "@mui/icons-material/ScienceOutlined";
import { Stack, Tooltip, Typography } from "@mui/material";
import { useFormContext } from "react-hook-form";
import { AutocompleteElement } from "react-hook-form-mui";
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
  const { data: componentStatus } =
    useGetApiSettingsHomeassistantCustomComponentStatusQuery();
  const commonProps = { control, disabled: readOnly };
  const experimentalLabMode = Boolean(watch("experimental_lab_mode"));
  const addonMDNSEnabled = Boolean(watch("addon_mdns_registration"));
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

      {/* mDNS / Zeroconf Registration */}
      <Tooltip
        title={
          <>
            <Typography variant="h6" component="div">
              mDNS / Zeroconf Registration
            </Typography>
            <Typography variant="body2">
              Announce this Samba server on the local network via Home Assistant
              using mDNS (Zeroconf). When enabled, other devices can discover
              the server automatically. Requires an active Home Assistant add-on
              connection.
            </Typography>
            {!isComponentConnected && (
              <Typography
                variant="body2"
                sx={{ mt: 1, color: "warning.light" }}
              >
                <strong>Not available:</strong> Home Assistant custom component
                is not connected.
              </Typography>
            )}
            {addonMDNSEnabled && (
              <Typography
                variant="body2"
                sx={{ mt: 1, color: "warning.light" }}
              >
                <strong>Disabled:</strong> addon-side direct mDNS is active and
                is mutually exclusive with Home Assistant mDNS registration.
              </Typography>
            )}
          </>
        }
      >
        <SettingSwitchRow
          ariaLabel="mDNS Registration"
          label="mDNS Registration"
          name="mdns_registration"
          control={control}
          disabled={readOnly || !isComponentConnected || addonMDNSEnabled}
        />
      </Tooltip>

      {/* Addon-side direct mDNS (lab feature) */}
      {experimentalLabMode ? (
        <Stack spacing={2}>
          <Tooltip
            title={
              <>
                <Typography variant="h6" component="div">
                  Addon-side Direct mDNS (Lab)
                </Typography>
                <Typography variant="body2">
                  Announce this Samba server directly from the add-on using mDNS
                  (Zeroconf). When enabled, the Home Assistant mDNS registration
                  toggle is disabled because the two modes are mutually
                  exclusive.
                </Typography>
              </>
            }
          >
            <SettingSwitchRow
              ariaLabel="Addon-side Direct mDNS"
              control={control}
              disabled={readOnly}
              label={labLabel("Addon-side Direct mDNS")}
              name="addon_mdns_registration"
            />
          </Tooltip>

          {addonMDNSEnabled ? (
            <AutocompleteElement
              name="addon_mdns_interfaces"
              label="mDNS Interfaces"
              multiple
              options={
                (capabilities as SystemCapabilities)
                  ?.available_mdns_interfaces ?? []
              }
              textFieldProps={{
                helperText:
                  "Leave empty to publish on all eligible interfaces.",
                disabled: readOnly,
              }}
              control={control}
            />
          ) : null}
        </Stack>
      ) : null}
    </Stack>
  );
}
