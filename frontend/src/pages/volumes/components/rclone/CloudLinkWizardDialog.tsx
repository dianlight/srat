import ScienceOutlinedIcon from "@mui/icons-material/ScienceOutlined";
import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  LinearProgress,
  Stack,
  Typography,
} from "@mui/material";
import { useEffect, useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import {
  FormContainer,
  PasswordElement,
  SelectElement,
  TextFieldElement,
} from "react-hook-form-mui";
import { toast } from "react-toastify";
import {
  Auth_mode,
  type RcloneConfigField,
  useGetApiRcloneLinkQuery,
  useGetApiRcloneProvidersQuery,
  usePostApiRcloneLinkAuthStartMutation,
  usePutApiRcloneLinkMutation,
} from "../../../../store/sratApi";
import { extractApiErrorMessage } from "./apiErrors";
import { isAuthStartResponse, isProvidersResponse } from "./typeGuards";

export interface CloudLinkWizardDialogProps {
  open: boolean;
  onClose: () => void;
  targetKind: string;
  targetId: string;
  targetLabel: string;
}

type WizardFormValues = {
  provider: string;
  remote_path: string;
  settings: Record<string, string>;
  // Explicit authorization mode picked by the user; "" until the providers
  // payload arrives, then defaulted to broker/custom_app per availability.
  // "ha_dropbox" is a UI-only stub value (option disabled) — never sent.
  auth_mode: Auth_mode | "ha_dropbox" | "";
};

const STEPS = ["Provider", "Account", "Remote folder"] as const;

/**
 * Browser-visible origin used by the backend to build the OAuth redirect
 * URI. The SRAT UI is served from the origin root, so this matches what the
 * user's browser can reach (unlike the addon-internal address).
 */
function publicBaseUrl(): string {
  return window.location.origin;
}

/**
 * Three-step wizard that links a local target to a cloud provider:
 * 1. pick a provider and fill its config fields (app key/secret...),
 * 2. authorize the account (the OAuth page opens in a new tab; SRAT
 *    receives the callback server-side so we poll the link status),
 * 3. choose the remote folder and create the link.
 *
 * The link row must exist before authorization can start (the backend
 * stores OAuth state on it), so step 2 saves a provisional link first.
 */
export function CloudLinkWizardDialog({
  open,
  onClose,
  targetKind,
  targetId,
  targetLabel,
}: CloudLinkWizardDialogProps) {
  const [step, setStep] = useState(0);
  const [authUrl, setAuthUrl] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const form = useForm<WizardFormValues>({
    defaultValues: {
      provider: "",
      remote_path: "",
      settings: {},
      auth_mode: "",
    },
  });

  const providersQuery = useGetApiRcloneProvidersQuery(undefined, {
    skip: !open,
  });
  // Poll the link while waiting for the OAuth callback to land backend-side.
  const linkQuery = useGetApiRcloneLinkQuery(
    { targetKind, targetId },
    { skip: !open || step !== 1, pollingInterval: 3000 },
  );

  const [putLink, { isLoading: isPutting }] = usePutApiRcloneLinkMutation();
  const [startAuth, { isLoading: isStartingAuth }] =
    usePostApiRcloneLinkAuthStartMutation();

  const authorized = linkQuery.data?.status === "authorized";

  const providersData = isProvidersResponse(providersQuery.data)
    ? providersQuery.data
    : undefined;

  const selectedProviderName = form.watch("provider");
  const selectedProvider = useMemo(
    () =>
      providersData?.providers?.find((p) => p.name === selectedProviderName),
    [providersData, selectedProviderName],
  );
  const configFields: RcloneConfigField[] = useMemo(
    () => selectedProvider?.config_fields ?? [],
    [selectedProvider],
  );

  useEffect(() => {
    if (open) {
      setStep(0);
      setAuthUrl(null);
      setActionError(null);
      form.reset({
        provider: "",
        remote_path: `/srat/${targetLabel}`,
        settings: {},
        auth_mode: "",
      });
    }
  }, [open, targetLabel, form]);

  // Default the explicit mode picker once the providers payload arrives:
  // hosted OAuth when the broker is configured, custom app otherwise.
  // Task 050: prefer HA Dropbox reuse when available for dropbox targets.
  const brokerAvailable = Boolean(providersData?.broker_available);
  const haDropboxAvailable = Boolean(
    (providersData as { ha_dropbox_available?: boolean } | undefined)
      ?.ha_dropbox_available,
  );
  const authMode = form.watch("auth_mode");
  useEffect(() => {
    if (!open || !providersData) return;
    if (form.getValues("auth_mode") === "") {
      const providers = providersData as {
        broker_available?: boolean;
        ha_dropbox_available?: boolean;
      };
      const defaultMode = providers.ha_dropbox_available
        ? (Auth_mode.HaDropbox as Auth_mode | "ha_dropbox")
        : providers.broker_available
          ? Auth_mode.Broker
          : Auth_mode.CustomApp;
      form.setValue("auth_mode", defaultMode as Auth_mode | "ha_dropbox" | "");
    }
  }, [open, providersData, form]);

  const connectAccount = async () => {
    setActionError(null);
    const validModes = new Set<string>([
      Auth_mode.Broker,
      Auth_mode.CustomApp,
      Auth_mode.HaDropbox,
      "ha_dropbox",
    ]);
    if (!validModes.has(authMode as string)) {
      setActionError("This authorization mode is not available yet.");
      return;
    }
    const isHaDropbox =
      authMode === Auth_mode.HaDropbox ||
      authMode === ("ha_dropbox" as Auth_mode);
    const settings = isHaDropbox ? {} : (form.getValues("settings") ?? {});
    try {
      // The row must exist before StartAuth (it stores OAuth state on it).
      await putLink({
        targetKind,
        targetId,
        rcloneLinkRequest: {
          provider: selectedProviderName,
          remote_path: "/",
          auto_sync: false,
          schedule_minutes: 0,
          settings,
        },
      }).unwrap();
      const res = await startAuth({
        targetKind,
        targetId,
        startRcloneAuthInputBody: {
          settings,
          // Browser-visible origin so the redirect URI registered in the
          // provider console actually resolves for this user.
          public_base_url: publicBaseUrl(),
          auth_mode: isHaDropbox
            ? Auth_mode.HaDropbox
            : authMode === Auth_mode.Broker
              ? Auth_mode.Broker
              : Auth_mode.CustomApp,
        },
      }).unwrap();
      if (isAuthStartResponse(res)) {
        setAuthUrl(res.auth_url);
      } else {
        throw Object.assign(new Error(), { data: res });
      }
    } catch (err) {
      setActionError(
        extractApiErrorMessage(err, "Failed to start authorization"),
      );
    }
  };

  const handleSubmit = async (values: WizardFormValues) => {
    setActionError(null);
    try {
      await putLink({
        targetKind,
        targetId,
        rcloneLinkRequest: {
          provider: values.provider,
          remote_path: values.remote_path,
          auto_sync: false,
          schedule_minutes: 0,
          settings: values.settings ?? {},
        },
      }).unwrap();
      toast.success(`Linked ${targetLabel} to ${values.provider}.`);
      onClose();
    } catch (err) {
      setActionError(extractApiErrorMessage(err, "Failed to create the link"));
    }
  };

  const goNext = async () => {
    if (step === 0) {
      const requiredSettingPaths = configFields
        .filter((f) => fieldRequired(f))
        .map((f): `settings.${string}` => `settings.${f.name}`);
      const paths: Array<"provider" | `settings.${string}`> = [
        "provider",
        ...requiredSettingPaths,
      ];
      const valid = await form.trigger(paths);
      if (valid) {
        setStep(1);
      }
      return;
    }
    if (step === 1 && authorized) {
      setStep(2);
    }
  };

  const libraryAvailable = Boolean(providersData?.library_available);
  // Credential fields only matter in custom-app mode: hosted OAuth and HA
  // Dropbox reuse need no credentials.
  const isHaDropboxMode =
    authMode === Auth_mode.HaDropbox ||
    authMode === ("ha_dropbox" as Auth_mode);
  const fieldRequired = (field: RcloneConfigField): boolean =>
    Boolean(field.required) &&
    authMode === Auth_mode.CustomApp &&
    !isHaDropboxMode;

  return (
    <Dialog open={open} onClose={() => {}} fullWidth maxWidth="sm">
      <DialogTitle>{`Link “${targetLabel}” to cloud`}</DialogTitle>
      <FormContainer formContext={form} onSuccess={handleSubmit}>
        <DialogContent>
          <Stack spacing={2} sx={{ alignItems: "flex-start" }}>
            <Stack direction="row" spacing={1} sx={{ alignItems: "center" }}>
              <ScienceOutlinedIcon color="warning" fontSize="small" />
              <Typography variant="caption" color="text.secondary">
                {`Lab feature · Step ${step + 1}/3 — ${STEPS[step]}`}
              </Typography>
            </Stack>

            {!providersQuery.isLoading && !libraryAvailable && (
              <Alert severity="warning" sx={{ width: "100%" }}>
                The rclone library is not available in this build.
              </Alert>
            )}
            {!providersQuery.isLoading &&
              libraryAvailable &&
              (providersData?.providers?.length ?? 0) === 0 && (
                <Alert severity="info" sx={{ width: "100%" }}>
                  No cloud providers are registered.
                </Alert>
              )}

            {step === 0 && (
              <>
                <SelectElement
                  name="provider"
                  label="Provider"
                  fullWidth
                  required
                  options={(providersData?.providers ?? []).map((p) => ({
                    id: p.name,
                    label: p.display_name,
                  }))}
                />
                <SelectElement
                  name="auth_mode"
                  label="Authorization"
                  fullWidth
                  options={[
                    {
                      id: "custom_app",
                      label: "Custom app (App key / secret)",
                    },
                    {
                      id: "broker",
                      label: "Hosted SRAT OAuth (shared app)",
                      disabled: !brokerAvailable,
                    },
                    {
                      id: "ha_dropbox",
                      label: "Reuse Dropbox integration auth",
                      disabled:
                        !haDropboxAvailable ||
                        selectedProviderName !== "dropbox",
                    },
                  ]}
                />
                {!brokerAvailable && !haDropboxAvailable && (
                  <Typography variant="caption" color="text.secondary">
                    Hosted SRAT OAuth is unavailable: no OAuth broker is
                    configured on this server (SRAT_OAUTH_BROKER_URL unset).
                  </Typography>
                )}
                {!haDropboxAvailable && selectedProviderName === "dropbox" && (
                  <Typography variant="caption" color="text.secondary">
                    Reuse Dropbox integration auth is unavailable: no HA Dropbox
                    token has been pushed (ensure the Home Assistant Dropbox
                    integration is configured).
                  </Typography>
                )}
                {authMode === Auth_mode.CustomApp &&
                  configFields.length > 0 && (
                    <Alert severity="info" sx={{ width: "100%" }}>
                      <Typography variant="body2">
                        Unlike plain rclone, SRAT completes the OAuth flow
                        server-side, so provider credentials cannot be left
                        empty (rclone&apos;s built-in app only allows its own
                        loopback redirect). Create your own app at the provider
                        console and register this redirect URI:
                      </Typography>
                      <Typography
                        variant="body2"
                        sx={{
                          fontFamily: "monospace",
                          wordBreak: "break-all",
                          mt: 0.5,
                        }}
                      >
                        {`${publicBaseUrl()}${providersData?.oauth_callback_path ?? "/api/rclone/oauth/callback"}`}
                      </Typography>
                    </Alert>
                  )}
                {(authMode as string) === Auth_mode.Broker && (
                  <Alert severity="info" sx={{ width: "100%" }}>
                    <Typography variant="body2">
                      Credentials are optional: authorization runs through
                      SRAT&apos;s hosted shared app, so you can leave the fields
                      empty and just sign in to your account.
                    </Typography>
                  </Alert>
                )}
                {isHaDropboxMode && (
                  <Alert severity="info" sx={{ width: "100%" }}>
                    <Typography variant="body2">
                      Reuses the OAuth token already held by Home Assistant
                      core&apos;s Dropbox integration (
                      homeassistant/components/dropbox, Cloud Account Linking
                      via accounts.home-assistant.io or your own Application
                      Credentials). No browser window is needed — the SRAT
                      add-on reuses hass.config_entries. The refresh token stays
                      bound to the HA app (same trade-off as broker mode).
                    </Typography>
                  </Alert>
                )}
                {configFields.map((field) =>
                  field.secret ? (
                    <PasswordElement
                      key={field.name}
                      name={`settings.${field.name}`}
                      label={field.label}
                      rules={
                        fieldRequired(field)
                          ? { required: `${field.label} is required` }
                          : undefined
                      }
                      fullWidth
                      autoComplete="off"
                    />
                  ) : (
                    <TextFieldElement
                      key={field.name}
                      name={`settings.${field.name}`}
                      label={field.label}
                      rules={
                        fieldRequired(field)
                          ? { required: `${field.label} is required` }
                          : undefined
                      }
                      fullWidth
                      helperText={field.description}
                    />
                  ),
                )}
              </>
            )}

            {step === 1 &&
              (authorized ? (
                <Alert severity="success" sx={{ width: "100%" }}>
                  Account connected. Continue to choose the remote folder.
                </Alert>
              ) : authUrl ? (
                <>
                  <Typography variant="body2">
                    Authorization pending. Complete the sign-in in the opened
                    page; this dialog continues automatically.
                  </Typography>
                  <Button href={authUrl} target="_blank" rel="noreferrer">
                    Open authorization page again
                  </Button>
                  <LinearProgress sx={{ width: "100%" }} />
                </>
              ) : (
                <Button
                  type="button"
                  variant="contained"
                  onClick={() => {
                    void connectAccount();
                  }}
                  disabled={
                    isPutting || isStartingAuth || !selectedProviderName
                  }
                >
                  {isPutting || isStartingAuth
                    ? "Connecting…"
                    : `Connect ${selectedProviderName || "provider"}`}
                </Button>
              ))}

            {step === 2 && (
              <TextFieldElement
                name="remote_path"
                label="Remote folder"
                required
                fullWidth
                helperText="Path inside the cloud provider, created if missing."
              />
            )}

            {actionError && (
              <Alert severity="error" sx={{ width: "100%" }}>
                {actionError}
              </Alert>
            )}
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={onClose}>Cancel</Button>
          {step > 0 && (
            <Button onClick={() => setStep(step - 1)} disabled={isPutting}>
              Back
            </Button>
          )}
          {step < 2 ? (
            <Button
              type="button"
              variant="contained"
              disabled={
                (step === 0 && !libraryAvailable) || (step === 1 && !authorized)
              }
              onClick={() => {
                void goNext();
              }}
            >
              Next
            </Button>
          ) : (
            <Button type="submit" variant="contained" disabled={isPutting}>
              Create link
            </Button>
          )}
        </DialogActions>
      </FormContainer>
    </Dialog>
  );
}
