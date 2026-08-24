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
import type { RcloneConfigField } from "../../../../store/sratApi";
import {
  useGetApiRcloneLinkByTargetKindAndTargetIdQuery,
  useGetApiRcloneProvidersQuery,
  usePostApiRcloneLinkByTargetKindAndTargetIdAuthStartMutation,
  usePutApiRcloneLinkByTargetKindAndTargetIdMutation,
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
};

const STEPS = ["Provider", "Account", "Remote folder"] as const;

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
    defaultValues: { provider: "", remote_path: "", settings: {} },
  });

  const providersQuery = useGetApiRcloneProvidersQuery(undefined, {
    skip: !open,
  });
  // Poll the link while waiting for the OAuth callback to land backend-side.
  const linkQuery = useGetApiRcloneLinkByTargetKindAndTargetIdQuery(
    { targetKind, targetId },
    { skip: !open || step !== 1, pollingInterval: 3000 },
  );

  const [putLink, { isLoading: isPutting }] =
    usePutApiRcloneLinkByTargetKindAndTargetIdMutation();
  const [startAuth, { isLoading: isStartingAuth }] =
    usePostApiRcloneLinkByTargetKindAndTargetIdAuthStartMutation();

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
      });
    }
  }, [open, targetLabel, form]);

  const connectAccount = async () => {
    setActionError(null);
    const settings = form.getValues("settings") ?? {};
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
        startRcloneAuthInputBody: { settings },
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
        .filter((f) => f.required)
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
                {configFields.map((field) =>
                  field.secret ? (
                    <PasswordElement
                      key={field.name}
                      name={`settings.${field.name}`}
                      label={field.label}
                      rules={
                        field.required
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
                        field.required
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
