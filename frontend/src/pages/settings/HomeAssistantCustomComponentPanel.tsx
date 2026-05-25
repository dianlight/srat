import ScienceOutlinedIcon from "@mui/icons-material/ScienceOutlined";
import {
  Alert,
  Button,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  Paper,
  Stack,
  Typography,
} from "@mui/material";
import { useState } from "react";
import {
  type HomeAssistantCustomComponentStatus,
  sratApi,
  useDeleteApiSettingsHomeassistantCustomComponentMutation,
  useGetApiSettingsHomeassistantCustomComponentStatusQuery,
  usePostApiSettingsHomeassistantCustomComponentInstallMutation,
  usePostApiSettingsHomeassistantCustomComponentUpgradeMutation,
  usePostApiSettingsHomeassistantRestartCoreMutation,
} from "../../store/sratApi";
import { useAppDispatch } from "../../store/store";

type LifecycleAction = "install" | "upgrade" | "uninstall";

export function HomeAssistantCustomComponentPanel({
  readOnly,
}: {
  readOnly: boolean;
}) {
  const dispatch = useAppDispatch();
  const {
    data: statusResponse,
    isLoading,
    isFetching,
    refetch,
    error: statusError,
  } = useGetApiSettingsHomeassistantCustomComponentStatusQuery();
  const [installComponent, installState] =
    usePostApiSettingsHomeassistantCustomComponentInstallMutation();
  const [upgradeComponent, upgradeState] =
    usePostApiSettingsHomeassistantCustomComponentUpgradeMutation();
  const [uninstallComponent, uninstallState] =
    useDeleteApiSettingsHomeassistantCustomComponentMutation();
  const [restartHACore, restartHACoreState] =
    usePostApiSettingsHomeassistantRestartCoreMutation();
  const [actionFeedback, setActionFeedback] = useState<{
    severity: "success" | "error";
    message: string;
  } | null>(null);
  const [confirmDialog, setConfirmDialog] = useState<{
    open: boolean;
    action: LifecycleAction | null;
  }>({ open: false, action: null });
  const [restartDialogOpen, setRestartDialogOpen] = useState(false);

  const status =
    statusResponse && "component" in statusResponse
      ? (statusResponse as HomeAssistantCustomComponentStatus)
      : undefined;

  const isBusy =
    installState.isLoading ||
    upgradeState.isLoading ||
    uninstallState.isLoading ||
    isFetching;

  const extractErrorMessage = (error: unknown, fallback: string): string => {
    if (!error || typeof error !== "object") {
      return fallback;
    }
    if ("error" in error && typeof error.error === "string") {
      return error.error;
    }
    if ("data" in error && error.data && typeof error.data === "object") {
      const data = error.data as { message?: string; error?: string };
      if (typeof data.message === "string") return data.message;
      if (typeof data.error === "string") return data.error;
    }
    if ("message" in error && typeof error.message === "string") {
      return error.message;
    }
    return fallback;
  };

  const openConfirmDialog = (action: LifecycleAction) => {
    setConfirmDialog({ open: true, action });
  };

  const closeConfirmDialog = () => {
    setConfirmDialog({ open: false, action: null });
  };

  const executeAction = async (action: LifecycleAction) => {
    setActionFeedback(null);
    try {
      if (action === "install") {
        await installComponent().unwrap();
        setActionFeedback({
          severity: "success",
          message: "Custom component installed successfully.",
        });
      } else if (action === "upgrade") {
        await upgradeComponent().unwrap();
        setActionFeedback({
          severity: "success",
          message: "Custom component upgraded successfully.",
        });
      } else {
        await uninstallComponent().unwrap();
        setActionFeedback({
          severity: "success",
          message: "Custom component uninstalled successfully.",
        });
      }
      await refetch();
      dispatch(sratApi.util.invalidateTags(["Issues"]));
      setRestartDialogOpen(true);
    } catch (error) {
      const labels: Record<LifecycleAction, string> = {
        install: "install",
        upgrade: "upgrade",
        uninstall: "uninstall",
      };
      setActionFeedback({
        severity: "error",
        message: extractErrorMessage(
          error,
          `Failed to ${labels[action]} custom component.`,
        ),
      });
    }
  };

  const handleConfirmAction = async () => {
    const action = confirmDialog.action;
    closeConfirmDialog();
    if (action) {
      await executeAction(action);
    }
  };

  const handleRestartNow = async () => {
    setRestartDialogOpen(false);
    try {
      await restartHACore().unwrap();
    } catch (error) {
      setActionFeedback({
        severity: "error",
        message: extractErrorMessage(
          error,
          "Failed to restart Home Assistant core.",
        ),
      });
    }
  };

  const confirmDialogContent = (): { title: string; body: string } => {
    const latestVersion = status?.latest_version ?? "";
    const installedVersion = status?.installed_version ?? "";
    switch (confirmDialog.action) {
      case "install":
        return {
          title: "Install SRAT Custom Component",
          body: latestVersion
            ? `Install the SRAT custom component (version ${latestVersion}) into your Home Assistant configuration?`
            : "Install the SRAT custom component into your Home Assistant configuration?",
        };
      case "upgrade":
        return {
          title: "Upgrade SRAT Custom Component",
          body:
            installedVersion && latestVersion
              ? `Upgrade the SRAT custom component from version ${installedVersion} to version ${latestVersion}?`
              : "Upgrade the SRAT custom component to the latest version?",
        };
      case "uninstall":
        return {
          title: "Uninstall SRAT Custom Component",
          body: installedVersion
            ? `Uninstall the SRAT custom component (version ${installedVersion}) from your Home Assistant configuration?`
            : "Uninstall the SRAT custom component from your Home Assistant configuration?",
        };
      default:
        return { title: "", body: "" };
    }
  };

  const { title: confirmTitle, body: confirmBody } = confirmDialogContent();

  return (
    <Paper variant="outlined" sx={{ p: 2 }}>
      <Stack spacing={1.5}>
        <Stack direction="row" spacing={1} sx={{ alignItems: "center" }}>
          <Typography variant="subtitle1">SRAT Custom Component</Typography>
          <ScienceOutlinedIcon color="warning" fontSize="small" />
        </Stack>

        {isLoading ? (
          <Stack direction="row" spacing={1} sx={{ alignItems: "center" }}>
            <CircularProgress size={16} />
            <Typography
              variant="body2"
              sx={{
                color: "text.secondary",
              }}
            >
              Loading custom component status…
            </Typography>
          </Stack>
        ) : (
          <Stack spacing={0.5}>
            {statusError ? (
              <Alert severity="error">
                {extractErrorMessage(
                  statusError,
                  "Unable to load custom component status.",
                )}
              </Alert>
            ) : null}
            <Typography variant="body2">
              Installed: <strong>{status?.installed ? "Yes" : "No"}</strong>
            </Typography>
            <Typography variant="body2">
              Connected: <strong>{status?.connected ? "Yes" : "No"}</strong>
            </Typography>
            <Typography variant="body2">
              Installed Version:{" "}
              <strong>{status?.installed_version || "—"}</strong>
            </Typography>
            <Typography variant="body2">
              Latest Version: <strong>{status?.latest_version || "—"}</strong>
            </Typography>
            {isBusy ? <Alert severity="info">Processing request…</Alert> : null}
            {actionFeedback ? (
              <Alert severity={actionFeedback.severity}>
                {actionFeedback.message}
              </Alert>
            ) : null}
          </Stack>
        )}

        <Stack direction={{ xs: "column", sm: "row" }} spacing={1}>
          <Button
            variant="outlined"
            color="success"
            disabled={readOnly || isBusy || !status?.can_install}
            onClick={() => {
              openConfirmDialog("install");
            }}
          >
            Install
          </Button>
          <Button
            variant="outlined"
            color="warning"
            disabled={readOnly || isBusy || !status?.can_upgrade}
            onClick={() => {
              openConfirmDialog("upgrade");
            }}
          >
            Upgrade
          </Button>
          <Button
            variant="outlined"
            color="error"
            disabled={readOnly || isBusy || !status?.can_uninstall}
            onClick={() => {
              openConfirmDialog("uninstall");
            }}
          >
            Uninstall
          </Button>
        </Stack>
      </Stack>
      {/* Action confirmation dialog */}
      <Dialog
        open={confirmDialog.open}
        onClose={closeConfirmDialog}
        aria-labelledby="cc-confirm-dialog-title"
      >
        <DialogTitle id="cc-confirm-dialog-title">{confirmTitle}</DialogTitle>
        <DialogContent>
          <DialogContentText>{confirmBody}</DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={closeConfirmDialog}>Cancel</Button>
          <Button
            onClick={() => {
              void handleConfirmAction();
            }}
            autoFocus
          >
            Confirm
          </Button>
        </DialogActions>
      </Dialog>
      {/* Restart required dialog */}
      <Dialog
        open={restartDialogOpen}
        onClose={() => {
          setRestartDialogOpen(false);
        }}
        aria-labelledby="cc-restart-dialog-title"
      >
        <DialogTitle id="cc-restart-dialog-title">Restart Required</DialogTitle>
        <DialogContent>
          <DialogContentText>
            A Home Assistant Core restart is required to apply the custom
            component changes. Would you like to restart now?
          </DialogContentText>
          {restartHACoreState.isLoading ? (
            <Stack
              direction="row"
              spacing={1}
              sx={{ mt: 1, alignItems: "center" }}
            >
              <CircularProgress size={16} />
              <Typography variant="body2">Restarting…</Typography>
            </Stack>
          ) : null}
        </DialogContent>
        <DialogActions>
          <Button
            onClick={() => {
              setRestartDialogOpen(false);
            }}
          >
            Later
          </Button>
          <Button
            onClick={() => {
              void handleRestartNow();
            }}
            color="warning"
            disabled={restartHACoreState.isLoading}
            autoFocus
          >
            Restart Now
          </Button>
        </DialogActions>
      </Dialog>
    </Paper>
  );
}
