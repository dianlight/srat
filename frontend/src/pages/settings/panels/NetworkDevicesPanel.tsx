import ScienceOutlinedIcon from "@mui/icons-material/ScienceOutlined";
import { Box, Stack, Tooltip, Typography } from "@mui/material";
import { useFormContext } from "react-hook-form";
import { AutocompleteElement, CheckboxElement } from "react-hook-form-mui";
import { TabIDs } from "../../../store/locationState";
import {
  type Settings as ApiSettings,
  type InterfaceStat,
  type SystemCapabilities,
  useGetApiCapabilitiesQuery,
  useGetApiNicsQuery,
} from "../../../store/sratApi";
import { SettingSwitchRow } from "../components/SettingSwitchRow";

type NetworkDevicesPanelProps = {
  readOnly: boolean;
};

export function NetworkDevicesPanel({ readOnly }: NetworkDevicesPanelProps) {
  const { control, watch } = useFormContext<ApiSettings>();
  const { data: nic, isLoading: isNicLoading } = useGetApiNicsQuery();
  const { data: capabilities, isLoading: isCapabilitiesLoading } =
    useGetApiCapabilitiesQuery();

  const bindAllWatch = watch("bind_all_interfaces");
  const experimentalLabMode = Boolean(watch("experimental_lab_mode"));
  const commonProps = { control, disabled: readOnly };

  const labLabel = (text: string) => (
    <Stack direction="row" spacing={1} sx={{ alignItems: "center" }}>
      <Typography component="span">{text}</Typography>
      <ScienceOutlinedIcon color="warning" fontSize="small" />
    </Stack>
  );

  return (
    <Stack spacing={3}>
      {/* Bind All Interfaces + Interfaces */}
      <Box data-tutor={`reactour__tab${TabIDs.SETTINGS}__step8`}>
        <CheckboxElement
          size="small"
          id="bind_all_interfaces"
          label="Bind All Interfaces"
          name="bind_all_interfaces"
          {...commonProps}
        />
        <AutocompleteElement
          multiple
          label="Interfaces"
          name="interfaces"
          options={
            (nic as InterfaceStat[])
              ?.map((nc) => nc?.name)
              .filter(
                (name): name is string =>
                  Boolean(name) && name !== "lo" && name !== "hassio",
              ) || []
          }
          loading={isNicLoading}
          autocompleteProps={{
            size: "small",
            disabled: bindAllWatch || readOnly,
          }}
          control={control}
        />
      </Box>
      {/* Multi Channel */}
      <Tooltip
        title={
          <>
            <Typography variant="h6" component="div">
              Enable Multi Channel Mode
            </Typography>
            <Typography variant="body2">
              This boolean parameter controls whether smbd(8) will support SMB3
              multi-channel.
            </Typography>
          </>
        }
      >
        <SettingSwitchRow
          ariaLabel="Multi Channel Mode"
          id="multi_channel"
          label="Multi Channel Mode"
          name="multi_channel"
          {...commonProps}
        />
      </Tooltip>
      {/* SMB over QUIC (lab feature) */}
      {experimentalLabMode ? (
        <Tooltip
          title={
            <>
              <Typography variant="h6" component="div">
                Enable SMB over QUIC (Lab)
              </Typography>
              <Typography variant="body2">
                This parameter enables SMB over QUIC transport protocol for
                improved performance and security. Requires Samba 4.23+ and QUIC
                kernel module support.
              </Typography>
              {capabilities &&
                "supports_quic" in capabilities &&
                !capabilities.supports_quic &&
                "unsupported_reason" in capabilities &&
                capabilities.unsupported_reason && (
                  <Typography
                    variant="body2"
                    sx={{ mt: 1, color: "warning.light" }}
                  >
                    <strong>Not available:</strong>{" "}
                    {capabilities.unsupported_reason}
                  </Typography>
                )}
            </>
          }
        >
          <SettingSwitchRow
            ariaLabel="SMB over QUIC"
            control={control}
            disabled={
              readOnly ||
              isCapabilitiesLoading ||
              !(
                capabilities &&
                "supports_quic" in capabilities &&
                capabilities.supports_quic
              )
            }
            id="smb_over_quic"
            label={labLabel("SMB over QUIC")}
            name="smb_over_quic"
          />
        </Tooltip>
      ) : null}
      {experimentalLabMode &&
        capabilities &&
        "supports_quic" in capabilities &&
        !capabilities.supports_quic &&
        !isCapabilitiesLoading &&
        "unsupported_reason" in capabilities &&
        capabilities.unsupported_reason && (
          <Typography
            variant="caption"
            sx={{
              color: "warning.main",
              mt: 0.5,
              display: "block",
            }}
          >
            {
              (
                capabilities as SystemCapabilities & {
                  unsupported_reason: string;
                }
              ).unsupported_reason
            }
          </Typography>
        )}
    </Stack>
  );
}
