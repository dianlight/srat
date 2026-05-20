import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  Link,
  Radio,
  RadioGroup,
  Typography,
} from "@mui/material";
import type React from "react";
import { useEffect } from "react";
import { Controller, useForm } from "react-hook-form";
import {
  type Settings,
  Telemetry_mode,
  useGetApiSettingsQuery,
  useGetApiTelemetryInternetConnectionQuery,
  usePutApiSettingsMutation,
} from "../store/sratApi";

interface TelemetryModalProps {
  open: boolean;
  onClose: () => void;
}

interface TelemetryFormData {
  telemetry_mode: Telemetry_mode;
}

const TelemetryModal: React.FC<TelemetryModalProps> = ({ open, onClose }) => {
  const formContext = useForm<TelemetryFormData>({
    defaultValues: { telemetry_mode: Telemetry_mode.All },
  });
  const {
    control,
    handleSubmit,
    setError,
    formState: { isSubmitting, errors },
  } = formContext;

  const { data: internetConnection, isLoading: isCheckingConnection } =
    useGetApiTelemetryInternetConnectionQuery();
  const { data: settings } = useGetApiSettingsQuery();
  const [updateSettings] = usePutApiSettingsMutation();

  // Don't show modal if no internet connection
  useEffect(() => {
    if (!isCheckingConnection && internetConnection === false) {
      onClose();
    }
  }, [internetConnection, isCheckingConnection, onClose]);

  const onSuccess = async (data: TelemetryFormData) => {
    if (!settings) return;
    try {
      await updateSettings({
        settings: {
          ...settings,
          telemetry_mode: data.telemetry_mode,
        } as Settings,
      }).unwrap();
      onClose();
    } catch {
      setError("root", {
        message: "Failed to update telemetry settings. Please try again.",
      });
    }
  };

  // Show loading if checking connection
  if (isCheckingConnection) {
    return (
      <Dialog open={open} maxWidth="sm" fullWidth>
        <DialogContent sx={{ display: "flex", justifyContent: "center", p: 4 }}>
          <CircularProgress />
        </DialogContent>
      </Dialog>
    );
  }

  // Don't show modal if no internet connection
  if (internetConnection === false) {
    return null;
  }

  return (
    <Dialog
      open={open}
      onClose={(_e, reason) => {
        if (reason !== "backdropClick" && reason !== "escapeKeyDown") onClose();
      }}
      maxWidth="md"
      fullWidth
    >
      <form onSubmit={handleSubmit(onSuccess)}>
        <DialogTitle>Help Improve SRAT</DialogTitle>
        <DialogContent>
          <Typography variant="body1">
            Help us improve SRAT by sharing anonymous usage data and error
            reports. This helps us identify issues and improve the software for
            everyone.
          </Typography>

          <Alert severity="info" sx={{ mb: 2 }}>
            <Typography variant="body2">
              All data is sent securely to Sentry servers and is used solely for
              improving the software. No personal information or file contents
              are transmitted.
            </Typography>
          </Alert>

          {errors.root && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {errors.root.message}
            </Alert>
          )}

          <Box sx={{ mt: 2 }}>
            <Typography variant="h6" gutterBottom>
              Choose your preference:
            </Typography>

            <Controller
              name="telemetry_mode"
              control={control}
              render={({ field }) => (
                <RadioGroup {...field}>
                  <FormControlLabel
                    value={Telemetry_mode.All}
                    control={<Radio />}
                    label={
                      <Box>
                        <Typography
                          variant="body1"
                          sx={{
                            fontWeight: "medium",
                          }}
                        >
                          Send usage data and error reports
                        </Typography>
                        <Typography
                          variant="body2"
                          sx={{
                            color: "text.secondary",
                          }}
                        >
                          Help us improve SRAT by sharing anonymous usage
                          statistics and error reports
                        </Typography>
                      </Box>
                    }
                  />
                  <FormControlLabel
                    value={Telemetry_mode.Errors}
                    control={<Radio />}
                    label={
                      <Box>
                        <Typography
                          variant="body1"
                          sx={{
                            fontWeight: "medium",
                          }}
                        >
                          Send only error reports
                        </Typography>
                        <Typography
                          variant="body2"
                          sx={{
                            color: "text.secondary",
                          }}
                        >
                          Share only error reports to help us fix bugs and
                          improve stability
                        </Typography>
                      </Box>
                    }
                  />
                  <FormControlLabel
                    value={Telemetry_mode.Disabled}
                    control={<Radio />}
                    label={
                      <Box>
                        <Typography
                          variant="body1"
                          sx={{
                            fontWeight: "medium",
                          }}
                        >
                          Don't send any data
                        </Typography>
                        <Typography
                          variant="body2"
                          sx={{
                            color: "text.secondary",
                          }}
                        >
                          No data will be sent to external servers
                        </Typography>
                      </Box>
                    }
                  />
                </RadioGroup>
              )}
            />
          </Box>

          <Typography
            variant="body2"
            sx={{
              color: "text.secondary",
              mt: 2,
            }}
          >
            You can change this setting at any time in the Settings page. For
            more information about data collection, visit our{" "}
            <Link
              href="https://github.com/dianlight/srat/blob/main/PRIVACY.md"
              target="_blank"
            >
              privacy policy
            </Link>
            .
          </Typography>
        </DialogContent>
        <DialogActions sx={{ p: 2 }}>
          <Button
            type="submit"
            variant="outlined"
            disabled={isSubmitting}
            fullWidth
          >
            {isSubmitting ? "Saving..." : "Continue"}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  );
};

export default TelemetryModal;
