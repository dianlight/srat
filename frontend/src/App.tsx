import {
  Alert,
  Backdrop,
  Box,
  Button,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Snackbar,
  Typography,
} from "@mui/material";
import Container from "@mui/material/Container";
import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "react-toastify";
import BaseConfigModal from "./components/BaseConfigModal";
import { Footer } from "./components/Footer";
import GlobalEventMonitor from "./components/GlobalEventTracker";
import { NavBar } from "./components/NavBar";
import { ReadonlyCommandTerminal } from "./components/ReadonlyCommandTerminal";
import TelemetryModal from "./components/TelemetryModal";
import { useBaseConfigModal } from "./hooks/useBaseConfigModal";
import { useTelemetryModal } from "./hooks/useTelemetryModal";
import {
  type CommandExecutionSnapshot,
  type CommandStartedNotification,
  type CommandTerminatedNotification,
  useGetApiSettingsAppConfigQuery,
  usePutApiRestartMutation,
} from "./store/sratApi";
import { useGetServerEventsQuery } from "./store/wsApi";

export function App() {
  const [errorInfo, setErrorInfo] = useState<string>("");
  const [showAddonConfigChangedBanner, setShowAddonConfigChangedBanner] =
    useState(false);
  const [isRestartingAddon, setIsRestartingAddon] = useState(false);
  const mainArea = useRef<HTMLDivElement>(null);
  const { data: evdata, isLoading, error: herror } = useGetServerEventsQuery();
  const { data: appConfigResponse } = useGetApiSettingsAppConfigQuery();
  const [restartAddon] = usePutApiRestartMutation();
  const { shouldShow: showTelemetryModal, dismiss: dismissTelemetryModal } =
    useTelemetryModal();
  const { shouldShow: showBaseConfigModal, dismiss: dismissBaseConfigModal } =
    useBaseConfigModal();
  const [backdropOpen, setBackdropOpen] = useState(true);
  const backdropPrevOpen = useRef(undefined as boolean | undefined);
  const [commandSessions, setCommandSessions] = useState<
    Record<string, CommandExecutionSnapshot>
  >({});
  const [commandDialogExecutionId, setCommandDialogExecutionId] = useState<
    string | null
  >(null);
  const [commandDialogOpen, setCommandDialogOpen] = useState(false);
  const commandEventDedupRef = useRef<string>("");
  // Compute Backdrop open state
  useEffect(() => {
    const newBackdropOpen =
      evdata?.heartbeat?.alive === false || isLoading || herror !== undefined;
    //console.log("Computing backdrop open state:", { alive: evdata?.heartbeat?.alive, isLoading, herror, newBackdropOpen });
    setBackdropOpen(newBackdropOpen);
    return () => {
      if (backdropPrevOpen.current === true && newBackdropOpen === false) {
        //console.log("Backdrop is closing, reloading page to recover from error or server unavailability");
        setShowAddonConfigChangedBanner(false);
        window.location.reload();
      }
      if (backdropPrevOpen.current !== undefined || backdropOpen === false) {
        //console.log("Cleaning up backdrop open state effect", { alive: evdata?.heartbeat?.alive, isLoading, herror, backdropPrevOpen: backdropPrevOpen.current, backdropOpen });
        backdropPrevOpen.current = backdropOpen;
      }
    };
  }, [evdata, isLoading, herror, backdropOpen]);

  // This useEffect handles the automatic reset of errors after a delay.
  // It ensures that a timer is set only when an error occurs, and cleared if the error resolves
  // or the component unmounts. This prevents multiple timers from being created.
  useEffect(() => {
    let timer: ReturnType<typeof setTimeout> | undefined;
    if (herror) {
      timer = setTimeout(() => {
        // With the new error boundary, we don't need to manually reset errors
        console.log("Error auto-reset timer triggered");
      }, 5000);
    }
    return () => {
      if (timer) clearTimeout(timer);
    };
  }, [herror]);

  useEffect(() => {
    if (
      appConfigResponse &&
      "requires_restart" in appConfigResponse &&
      appConfigResponse.requires_restart
    ) {
      setShowAddonConfigChangedBanner(true);
    }
  }, [appConfigResponse]);

  useEffect(() => {
    if (evdata?.app_config_changed) {
      setShowAddonConfigChangedBanner(true);
    }
  }, [evdata?.app_config_changed]);

  useEffect(() => {
    function onBeforeUnload(ev: BeforeUnloadEvent) {
      if (sessionStorage.getItem("srat_dirty") === "true") {
        ev.preventDefault();
        return "Are you sure you want to leave? Your changes will be lost.";
      }
      return;
    }

    window.addEventListener("beforeunload", onBeforeUnload);

    return () => {
      window.removeEventListener("beforeunload", onBeforeUnload);
    };
  }, []);

  async function handleReloadWithAddonRestart() {
    if (isRestartingAddon) {
      return;
    }

    setIsRestartingAddon(true);
    setErrorInfo("");

    try {
      await restartAddon().unwrap();

      window.location.reload();
    } catch (error) {
      console.error("Addon restart failed", error);
      setErrorInfo("Addon restart failed. Please retry.");
      setIsRestartingAddon(false);
    }
  }

  const upsertCommandStarted = useCallback(
    (event: CommandStartedNotification) => {
      setCommandSessions((previous) => {
        const current = previous[event.execution_id];
        return {
          ...previous,
          [event.execution_id]: {
            execution_id: event.execution_id,
            command_id: event.command_id,
            label: event.label,
            command: event.command,
            args: event.args ?? [],
            started_at: event.started_at,
            running: true,
            success: current?.success,
            exit_code: current?.exit_code,
            finished_at: current?.finished_at,
            error: current?.error,
            lines: current?.lines ?? [],
          },
        };
      });
    },
    [],
  );

  const openCommandDialog = useCallback((executionId: string) => {
    setCommandDialogExecutionId(executionId);
    setCommandDialogOpen(true);
  }, []);

  useEffect(() => {
    const event = evdata?.command_started;
    if (!event) {
      return;
    }
    upsertCommandStarted(event);
  }, [evdata?.command_started, upsertCommandStarted]);

  useEffect(() => {
    const event = evdata?.command_output;
    if (!event) {
      return;
    }

    const dedupeKey = `${event.execution_id}:${event.channel}:${event.timestamp}:${event.line}`;
    if (commandEventDedupRef.current === dedupeKey) {
      return;
    }
    commandEventDedupRef.current = dedupeKey;

    setCommandSessions((previous) => {
      const current = previous[event.execution_id] ?? {
        executionId: event.execution_id,
        command_id: event.command_id,
        command: event.command_id,
        args: [],
        started_at: event.timestamp,
        running: true,
        lines: [],
      };

      const lines = [...(current.lines || []), event].slice(-500);

      return {
        ...previous,
        [event.execution_id]: {
          ...current,
          commandId: event.command_id,
          running: current.running,
          lines,
        },
      };
    });

    if (event.channel === "stderr" && !commandDialogOpen) {
      toast.error(
        <Box>
          <Typography variant="body2" sx={{ mb: 1 }}>
            Command stderr detected for {event.command_id}
          </Typography>
          <Button
            size="small"
            variant="outlined"
            onClick={() => openCommandDialog(event.execution_id)}
          >
            Open Output
          </Button>
        </Box>,
        { autoClose: 7000 },
      );
    }
  }, [evdata?.command_output, commandDialogOpen, openCommandDialog]);

  useEffect(() => {
    const event: CommandTerminatedNotification | undefined =
      evdata?.command_terminated;
    if (!event) {
      return;
    }

    setCommandSessions((previous) => {
      const current = previous[event.execution_id] ?? {
        executionId: event.execution_id,
        commandId: event.command_id,
        command: event.command_id,
        args: [],
        startedAt: event.finished_at,
        running: false,
        lines: [],
      };

      return {
        ...previous,
        [event.execution_id]: {
          ...current,
          running: false,
          success: event.success,
          exitCode: event.exit_code,
          finishedAt: event.finished_at,
          error: event.error,
        },
      };
    });
  }, [evdata?.command_terminated]);

  const selectedCommandSession =
    commandDialogExecutionId === null
      ? undefined
      : commandSessions[commandDialogExecutionId];

  const downloadCommandOutput = () => {
    if (!selectedCommandSession) {
      return;
    }
    const lines = selectedCommandSession.lines
      ?.map((line) => `[${line.channel}] ${line.line}`)
      .join("\n");
    if (!lines) {
      return;
    }
    const blob = new Blob([lines], { type: "text/plain;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `${selectedCommandSession.execution_id}.log.txt`;
    link.click();
    URL.revokeObjectURL(url);
  };

  return (
    <>
      <GlobalEventMonitor />
      <Container
        maxWidth={false}
        disableGutters={true}
        sx={{
          minHeight: "100vh",
          display: "flex",
          flexDirection: "column",
        }}
      >
        <NavBar error={errorInfo} bodyRef={mainArea} />
        <div ref={mainArea} className="fullBody" style={{ flexGrow: 1 }}></div>
        <Footer />
      </Container>
      <Backdrop
        sx={(theme) => ({ color: "#fff", zIndex: theme.zIndex.drawer + 1 })}
        open={backdropOpen}
        content={isLoading ? "Loading..." : "Server is not reachable"}
      >
        <CircularProgress color="inherit" />
      </Backdrop>
      <BaseConfigModal
        open={showBaseConfigModal}
        onClose={dismissBaseConfigModal}
      />
      <TelemetryModal
        open={showTelemetryModal}
        onClose={dismissTelemetryModal}
      />
      <Snackbar
        anchorOrigin={{ vertical: "top", horizontal: "center" }}
        open={
          showAddonConfigChangedBanner &&
          !(
            evdata?.heartbeat?.alive === false ||
            isLoading ||
            herror !== undefined
          )
        }
      >
        <Alert
          severity="warning"
          variant="filled"
          action={
            <>
              <Button
                color="inherit"
                size="small"
                onClick={() => setShowAddonConfigChangedBanner(false)}
                disabled={isRestartingAddon}
              >
                Ignore
              </Button>
              <Button
                color="inherit"
                size="small"
                onClick={handleReloadWithAddonRestart}
                disabled={isRestartingAddon}
              >
                {isRestartingAddon ? "Restarting..." : "Reload"}
              </Button>
            </>
          }
        >
          Addon configuration has changed. Reload required.
        </Alert>
      </Snackbar>
      <Dialog
        open={commandDialogOpen}
        onClose={() => setCommandDialogOpen(false)}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>
          Command Output:{" "}
          {selectedCommandSession?.label ??
            selectedCommandSession?.command_id ??
            "Unknown"}
        </DialogTitle>
        <DialogContent dividers>
          <Typography variant="body2" sx={{ mb: 1 }}>
            Execution: {selectedCommandSession?.execution_id}
          </Typography>
          <Typography variant="body2" sx={{ mb: 2 }}>
            Status:{" "}
            {selectedCommandSession?.running
              ? "Running"
              : `${selectedCommandSession?.success ? "Success" : "Failed"} (exit ${selectedCommandSession?.exit_code ?? "n/a"})`}
          </Typography>
          <ReadonlyCommandTerminal lines={selectedCommandSession?.lines} />
        </DialogContent>
        <DialogActions>
          <Button onClick={downloadCommandOutput} variant="outlined">
            Download
          </Button>
          <Button
            onClick={() => setCommandDialogOpen(false)}
            variant="contained"
          >
            Close
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}
