import {
  Alert,
  CircularProgress,
  DialogContent,
  FormControlLabel,
  Link,
  Radio,
  RadioGroup,
  Typography,
} from "@mui/material";
import { type Control, Controller } from "react-hook-form";
import { Telemetry_mode } from "../../../store/sratApi";
import type { TelemetryFormData } from "../types";

interface TelemetryStepContentProps {
  internetConnection: boolean | undefined;
  isCheckingConnection: boolean;
  control: Control<TelemetryFormData>;
}

export function TelemetryStepContent({
  internetConnection,
  isCheckingConnection,
  control,
}: TelemetryStepContentProps) {
  if (isCheckingConnection) {
    return (
      <DialogContent sx={{ display: "flex", justifyContent: "center", p: 4 }}>
        <CircularProgress />
      </DialogContent>
    );
  }

  if (internetConnection === false) {
    return (
      <DialogContent>
        <Alert severity="info">
          No internet connection detected. Telemetry will remain disabled.
        </Alert>
      </DialogContent>
    );
  }

  return (
    <DialogContent>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Help improve SRAT by sharing anonymous usage data.
      </Typography>
      <Alert severity="info" sx={{ mb: 2 }}>
        All data is sent securely and used solely for improving the software. No
        personal information is transmitted.
      </Alert>
      <Controller
        name="telemetry_mode"
        control={control}
        render={({ field }) => (
          <RadioGroup {...field}>
            <FormControlLabel
              value={Telemetry_mode.All}
              control={<Radio />}
              label="Send usage data and error reports"
            />
            <FormControlLabel
              value={Telemetry_mode.Errors}
              control={<Radio />}
              label="Send only error reports"
            />
            <FormControlLabel
              value={Telemetry_mode.Disabled}
              control={<Radio />}
              label="Don't send any data"
            />
          </RadioGroup>
        )}
      />
      <Typography variant="body2" color="text.secondary" sx={{ mt: 2 }}>
        You can change this in Settings at any time.{" "}
        <Link
          href="https://github.com/dianlight/srat/blob/main/PRIVACY.md"
          target="_blank"
        >
          Privacy policy
        </Link>
      </Typography>
    </DialogContent>
  );
}
