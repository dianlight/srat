import { Stack, Typography } from "@mui/material";
import { useFormContext } from "react-hook-form";
import { AutocompleteElement } from "react-hook-form-mui";
import {
  type Settings as ApiSettings,
  Telemetry_mode,
  useGetApiTelemetryInternetConnectionQuery,
  useGetApiTelemetryModesQuery,
} from "../../../store/sratApi";

type TelemetryPanelProps = {
  readOnly: boolean;
};

export function TelemetryPanel({ readOnly }: TelemetryPanelProps) {
  const { control } = useFormContext<ApiSettings>();
  const { data: telemetryModes, isLoading: isTelemetryLoading } =
    useGetApiTelemetryModesQuery();
  const { data: internetConnection, isLoading: isInternetLoading } =
    useGetApiTelemetryInternetConnectionQuery();

  return (
    <Stack spacing={1}>
      <AutocompleteElement
        label="Telemetry Mode"
        name="telemetry_mode"
        required
        loading={isTelemetryLoading}
        autocompleteProps={{
          size: "small",
          disabled: readOnly || isInternetLoading || !internetConnection,
          contentEditable: false,
          disableClearable: true,
          autoComplete: false,
        }}
        textFieldProps={{ autoComplete: "off" }}
        options={
          (telemetryModes as string[])?.filter(
            (mode) => mode !== Telemetry_mode.Ask,
          ) || []
        }
        control={control}
      />
      {!internetConnection && !isInternetLoading && (
        <Typography
          variant="caption"
          color="text.secondary"
          sx={{ mt: 0.5, display: "block" }}
        >
          Internet connection required for telemetry settings
        </Typography>
      )}
    </Stack>
  );
}
