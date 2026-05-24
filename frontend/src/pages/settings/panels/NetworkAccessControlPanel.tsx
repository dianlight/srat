import PlaylistAddIcon from "@mui/icons-material/PlaylistAdd";
import { Box, Stack } from "@mui/material";
import IconButton from "@mui/material/IconButton";
import InputAdornment from "@mui/material/InputAdornment";
import Tooltip from "@mui/material/Tooltip";
import { MuiChipsInput } from "mui-chips-input";
import { useFormContext } from "react-hook-form";
import { Controller } from "react-hook-form-mui";
import default_json from "../../../json/default_config.json";
import { TabIDs } from "../../../store/locationState";
import type { Settings as ApiSettings } from "../../../store/sratApi";
import { isValidIpAddressOrCidr } from "../settingsConfig";

type NetworkAccessControlPanelProps = {
  readOnly: boolean;
};

export function NetworkAccessControlPanel({
  readOnly,
}: NetworkAccessControlPanelProps) {
  const { control, getValues, setValue } = useFormContext<ApiSettings>();

  return (
    <Stack spacing={0.5}>
      <Box data-tutor={`reactour__tab${TabIDs.SETTINGS}__step5`}>
        <Controller
          name="allow_hosts"
          control={control}
          defaultValue={[]}
          disabled={readOnly}
          rules={{
            required: "Allow Hosts cannot be empty.",
            validate: (chips: string[] | undefined) => {
              if (!chips || chips.length === 0) return true;
              for (const chip of chips) {
                if (typeof chip !== "string" || !isValidIpAddressOrCidr(chip)) {
                  return `Invalid entry: "${chip}". Only IPv4/IPv6 addresses or CIDR notation allowed.`;
                }
              }
              return true;
            },
          }}
          render={({ field, fieldState: { error } }) => (
            <MuiChipsInput
              {...field}
              size="small"
              label="Allow Hosts"
              required
              hideClearAll
              validate={(chipValue) =>
                typeof chipValue === "string" &&
                isValidIpAddressOrCidr(chipValue)
              }
              error={!!error}
              helperText={error ? error.message : undefined}
              slotProps={{
                input: {
                  endAdornment: (
                    <InputAdornment position="end" sx={{ pr: 1 }}>
                      {!readOnly && (
                        <Tooltip title="Add suggested default Allow Hosts">
                          <IconButton
                            aria-label="add suggested default allow hosts"
                            onClick={() => {
                              const currentAllowHosts: string[] =
                                getValues("allow_hosts") || [];
                              const defaultAllowHosts: string[] =
                                default_json.allow_hosts || [];
                              const validDefaultHosts =
                                defaultAllowHosts.filter((host) =>
                                  isValidIpAddressOrCidr(host),
                                );
                              const newAllowHostsToAdd =
                                validDefaultHosts.filter(
                                  (defaultHost) =>
                                    !currentAllowHosts.includes(defaultHost),
                                );
                              setValue(
                                "allow_hosts",
                                [...currentAllowHosts, ...newAllowHostsToAdd],
                                { shouldDirty: true, shouldValidate: true },
                              );
                            }}
                            edge="end"
                            data-tutor={`reactour__tab${TabIDs.SETTINGS}__step6`}
                          >
                            <PlaylistAddIcon />
                          </IconButton>
                        </Tooltip>
                      )}
                    </InputAdornment>
                  ),
                },
              }}
              renderChip={(Component, key, props) => {
                const isDefault = default_json.allow_hosts?.includes(
                  props.label as string,
                );
                return (
                  <Component
                    key={key}
                    {...props}
                    sx={{
                      color: isDefault ? "text.secondary" : "text.primary",
                    }}
                    size="small"
                  />
                );
              }}
            />
          )}
        />
      </Box>
    </Stack>
  );
}
