import AutorenewIcon from "@mui/icons-material/Autorenew";
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  InputAdornment,
  Stack,
  Tooltip,
  Typography,
} from "@mui/material";
import type React from "react";
import { useEffect } from "react";
import {
  FormContainer,
  PasswordElement,
  TextFieldElement,
  useForm,
} from "react-hook-form-mui";
import {
  type Settings,
  type User,
  useGetApiHostnameQuery,
  useGetApiSettingsQuery,
  useGetApiUsersQuery,
  usePutApiSettingsMutation,
  usePutApiUseradminMutation,
} from "../store/sratApi";

interface BaseConfigModalProps {
  open: boolean;
  onClose: () => void;
}

interface BaseConfigFormData {
  newPassword: string;
  confirmPassword: string;
  hostname: string;
  workgroup: string;
}

// Type guard to ensure settings is a Settings object and not an error
const isValidSettings = (data: unknown): data is Settings => {
  return data !== null && typeof data === "object" && "hostname" in data;
};

// Type guard to ensure users is an array and not an error
const isValidUsers = (data: unknown): data is User[] => {
  return Array.isArray(data);
};

const BaseConfigModal: React.FC<BaseConfigModalProps> = ({ open, onClose }) => {
  const { data: settings } = useGetApiSettingsQuery();
  const { data: users } = useGetApiUsersQuery();
  const { data: systemHostname, isLoading: isHostnameFetching } =
    useGetApiHostnameQuery();
  const [updateSettings] = usePutApiSettingsMutation();
  const [updateAdminUser] = usePutApiUseradminMutation();

  const formContext = useForm<BaseConfigFormData>({
    defaultValues: {
      newPassword: "",
      confirmPassword: "",
      hostname: "",
      workgroup: "WORKGROUP",
    },
  });

  const {
    setValue,
    setError,
    formState: { isSubmitting },
  } = formContext;

  useEffect(() => {
    if (isValidSettings(settings)) {
      setValue("hostname", settings.hostname || "");
      setValue("workgroup", settings.workgroup || "");
    }
  }, [settings, setValue]);

  useEffect(() => {
    if (!isHostnameFetching && systemHostname) {
      setValue("hostname", systemHostname as string);
    }
  }, [systemHostname, isHostnameFetching, setValue]);

  const handleSubmit = async (data: BaseConfigFormData) => {
    const completeSettings = {
      ...settings,
      hostname: data.hostname || undefined,
      workgroup: data.workgroup || undefined,
    } as Settings;
    if (!isValidUsers(users) || !isValidSettings(completeSettings)) {
      setError("root", {
        message:
          "Invalid configuration data. Please check the form and try again.",
      });
      return;
    }

    const adminUser = users.find((u) => u.is_admin);
    if (!adminUser) {
      setError("root", {
        message: "No admin user found. Please contact support.",
      });
      return;
    }

    try {
      await updateAdminUser({
        user: { ...adminUser, password: data.newPassword },
      }).unwrap();

      await updateSettings({
        settings: completeSettings,
      }).unwrap();

      onClose();
    } catch {
      setError("root", {
        message: "Failed to save changes. Please try again.",
      });
    }
  };

  return (
    <Dialog
      open={open}
      onClose={(_e, reason) => {
        if (reason !== "backdropClick" && reason !== "escapeKeyDown") onClose();
      }}
      maxWidth="md"
      fullWidth
    >
      <FormContainer formContext={formContext} onSuccess={handleSubmit}>
        <DialogTitle>Secure Your System</DialogTitle>
        <DialogContent>
          <Typography variant="body1" sx={{ mt: 2 }}>
            Welcome to SRAT (Samba Rest Administration Tool)! Your system is
            using the default administrator password. For security, you must
            change it now and configure basic system settings to proceed.
          </Typography>

          <Alert severity="warning" sx={{ mb: 3 }}>
            <Typography variant="body2">
              The current default password is <strong>changeme!</strong>. You
              must change this immediately for security reasons.
            </Typography>
          </Alert>

          <Stack spacing={2}>
            <Box>
              <Typography variant="h6" gutterBottom sx={{ mt: 2 }}>
                Change Administrator Password
              </Typography>
              <Typography
                variant="body2"
                sx={{
                  color: "text.secondary",
                }}
              >
                Create a strong new password for your administrator account.
              </Typography>
            </Box>

            <PasswordElement
              name="newPassword"
              label="New Administrator Password"
              autoComplete="new-password"
              fullWidth
              helperText="Must be at least 6 characters"
              rules={{
                required: "Password is required",
                minLength: {
                  value: 6,
                  message: "Password must be at least 6 characters long",
                },
                validate: (value) =>
                  value !== "changeme!" ||
                  "Password cannot be the default password",
              }}
            />

            <PasswordElement
              name="confirmPassword"
              label="Confirm Password"
              autoComplete="new-password"
              fullWidth
              rules={{
                required: "Please confirm your password",
                validate: (value, formValues) =>
                  value === formValues.newPassword || "Passwords do not match",
              }}
            />

            {formContext.formState.errors.root && (
              <Alert severity="error">
                <Typography variant="body2">
                  {formContext.formState.errors.root.message}
                </Typography>
              </Alert>
            )}

            <Box>
              <Typography variant="h6" gutterBottom sx={{ mt: 2 }}>
                General Configuration
              </Typography>
              <Typography
                variant="body2"
                sx={{
                  color: "text.secondary",
                }}
              >
                Configure basic system settings for your Samba server.
              </Typography>
            </Box>

            <TextFieldElement
              name="hostname"
              label="Hostname"
              fullWidth
              placeholder="e.g., samba-nas"
              helperText="The name of your Samba server on the network"
              rules={{
                required: "Hostname is required",
                minLength: {
                  value: 2,
                  message: "Hostname must be at least 2 characters long",
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
                            onClick={() =>
                              setValue("hostname", systemHostname as string)
                            }
                            edge="end"
                            disabled={isHostnameFetching}
                            size="small"
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
            />

            <TextFieldElement
              name="workgroup"
              label="Workgroup"
              fullWidth
              placeholder="e.g., WORKGROUP"
              helperText="The workgroup name for your Samba server"
              rules={{
                required: "Workgroup is required",
                minLength: {
                  value: 2,
                  message: "Workgroup must be at least 2 characters long",
                },
              }}
            />
          </Stack>

          <Typography
            variant="body2"
            sx={{
              color: "text.secondary",
              mt: 3,
            }}
          >
            You can change these settings later in the Settings page.
          </Typography>
        </DialogContent>
        <DialogActions sx={{ p: 2 }}>
          <Button
            type="submit"
            variant="contained"
            disabled={isSubmitting}
            fullWidth
          >
            {isSubmitting ? "Saving..." : "Secure System"}
          </Button>
        </DialogActions>
      </FormContainer>
    </Dialog>
  );
};

export default BaseConfigModal;
