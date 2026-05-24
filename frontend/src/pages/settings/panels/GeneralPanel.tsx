import AutorenewIcon from "@mui/icons-material/Autorenew";
import {
  Box,
  CircularProgress,
  Stack,
  Tooltip,
  Typography,
} from "@mui/material";
import IconButton from "@mui/material/IconButton";
import InputAdornment from "@mui/material/InputAdornment";
import { useFormContext } from "react-hook-form";
import { TextFieldElement } from "react-hook-form-mui";
import { TabIDs } from "../../../store/locationState";
import {
  type Settings as ApiSettings,
  useGetApiHostnameQuery,
} from "../../../store/sratApi";
import { SettingSwitchRow } from "../components/SettingSwitchRow";
import { HOSTNAME_REGEX, WORKGROUP_REGEX } from "../settingsConfig";

type GeneralPanelProps = {
  readOnly: boolean;
};

export function GeneralPanel({ readOnly }: GeneralPanelProps) {
  const { control, setValue } = useFormContext<ApiSettings>();
  const {
    data: hostname,
    isLoading: isHostnameFetching,
    refetch: triggerGetSystemHostname,
  } = useGetApiHostnameQuery();

  const handleFetchHostname = async () => {
    if (readOnly || isHostnameFetching) return;
    try {
      await triggerGetSystemHostname().unwrap();
      setValue("hostname", hostname?.toString(), {
        shouldDirty: true,
        shouldValidate: true,
      });
    } catch (error) {
      console.error("Failed to fetch hostname:", error);
    }
  };

  const commonProps = { control, disabled: readOnly };

  return (
    <Stack spacing={3}>
      {/* Hostname */}
      <Box data-tutor={`reactour__tab${TabIDs.SETTINGS}__step3`}>
        <TextFieldElement
          size="small"
          sx={{ display: "flex" }}
          name="hostname"
          label="Hostname"
          required
          rules={{
            required: "Hostname is required.",
            pattern: {
              value: HOSTNAME_REGEX,
              message:
                "Invalid hostname. Use alphanumeric characters and hyphens (not at start/end). Max 63 chars.",
            },
            maxLength: {
              value: 63,
              message: "Hostname cannot exceed 63 characters.",
            },
          }}
          slotProps={{
            input: {
              endAdornment: (
                <InputAdornment position="end">
                  <Tooltip title="Fetch current system hostname">
                    <span>
                      <IconButton
                        aria-label="fetch system hostname"
                        onClick={handleFetchHostname}
                        edge="end"
                        disabled={readOnly || isHostnameFetching}
                        data-tutor={`reactour__tab${TabIDs.SETTINGS}__step4`}
                      >
                        {isHostnameFetching ? (
                          <CircularProgress size={20} />
                        ) : (
                          <AutorenewIcon />
                        )}
                      </IconButton>
                    </span>
                  </Tooltip>
                </InputAdornment>
              ),
            },
          }}
          {...commonProps}
        />
      </Box>

      {/* Workgroup */}
      <TextFieldElement
        size="small"
        sx={{ display: "flex" }}
        name="workgroup"
        label="Workgroup"
        required
        rules={{
          required: "Workgroup is required.",
          pattern: {
            value: WORKGROUP_REGEX,
            message:
              "Invalid workgroup name. Use alphanumeric characters and hyphens (not at start/end). Max 15 chars.",
          },
          maxLength: {
            value: 15,
            message: "Workgroup name cannot exceed 15 characters.",
          },
        }}
        {...commonProps}
      />

      {/* Local Master */}
      <Tooltip
        title={
          <>
            <Typography variant="h6" component="div">
              Enable Local Master
            </Typography>
            <Typography variant="body2">
              This option allows nmbd(8) to try and become a local master
              browser on a subnet. If set to no then nmbd will not attempt to
              become a local master browser on a subnet and will also lose in
              all browsing elections. By default this value is set to yes.
              Setting this value to yes doesn't mean that Samba will become the
              local master browser on a subnet, just that nmbd will participate
              in elections for local master browser.
            </Typography>
            <Typography variant="body2">
              Setting this value to no will cause nmbd never to become a local
              master browser.
            </Typography>
          </>
        }
      >
        <SettingSwitchRow
          ariaLabel="Local Master"
          label="Local Master"
          name="local_master"
          {...commonProps}
        />
      </Tooltip>

      {/* Compatibility Mode */}
      <Box data-tutor={`reactour__tab${TabIDs.SETTINGS}__step7`}>
        <SettingSwitchRow
          ariaLabel="Compatibility Mode"
          id="compatibility_mode"
          label="Compatibility Mode"
          name="compatibility_mode"
          {...commonProps}
        />
      </Box>

      {/* Allow Guest */}
      <SettingSwitchRow
        ariaLabel="Allow Guest"
        id="allow_guest"
        label="Allow Guest"
        name="allow_guest"
        {...commonProps}
      />

      {/* Disable SMART */}
      <Tooltip
        title={
          <>
            <Typography variant="h6" component="div">
              Disable SMART Integration
            </Typography>
            <Typography variant="body2">
              Stops SRAT-side SMART polling and hides SMART-related UI so
              sleeping disks are less likely to spin up in the background.
            </Typography>
            <Typography variant="body2" sx={{ mt: 1 }}>
              Leave this off unless you specifically need to reduce wake-ups on
              idle disks.
            </Typography>
          </>
        }
      >
        <SettingSwitchRow
          ariaLabel="Disable SMART Integration"
          id="disable_smart"
          label="Disable SMART Integration"
          name="disable_smart"
          {...commonProps}
        />
      </Tooltip>
    </Stack>
  );
}
