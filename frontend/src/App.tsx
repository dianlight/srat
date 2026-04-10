import {
  Alert,
  Backdrop,
  Button,
  CircularProgress,
  Snackbar,
} from "@mui/material";
import Container from "@mui/material/Container";
import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "react-toastify";
import BaseConfigModal from "./components/BaseConfigModal";
import {
  CommandOutputDialog,
  CommandOutputToastContent,
  getCommandOutputLines,
} from "./components/CommandOutputDialog";
import { Footer } from "./components/Footer";
import GlobalEventMonitor from "./components/GlobalEventTracker";
import { NavBar } from "./components/NavBar";
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
  const commandToastDedupRef = useRef<Set<string>>(new Set());
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

  const showCommandFailureToast = useCallback(
    (executionId: string, commandId: string) => {
      if (commandDialogOpen || commandToastDedupRef.current.has(executionId)) {
        return;
      }

      commandToastDedupRef.current.add(executionId);
      toast.error(
        <CommandOutputToastContent
          commandId={commandId}
          onOpenOutput={() => openCommandDialog(executionId)}
        />,
        { autoClose: 7000 },
      );
    },
    [commandDialogOpen, openCommandDialog],
  );

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

    const dedupeKey = `${event.execution_id}:${event.channel}:${event.timestamp}:${event.line}:${event.exit_code ?? "pending"}`;
    if (commandEventDedupRef.current === dedupeKey) {
      return;
    }
    commandEventDedupRef.current = dedupeKey;

    setCommandSessions((previous) => {
      const current = previous[event.execution_id] ?? {
        execution_id: event.execution_id,
        command_id: event.command_id,
        command: event.command_id,
        args: [],
        started_at: event.timestamp,
        running: true,
        lines: [],
      };

      const previousLines = current.lines || [];
      const lastLine = previousLines[previousLines.length - 1];
      const lines =
        lastLine?.channel === event.channel &&
        lastLine?.line === event.line &&
        lastLine?.timestamp === event.timestamp
          ? previousLines
          : [...previousLines, event].slice(-500);

      return {
        ...previous,
        [event.execution_id]: {
          ...current,
          command_id: event.command_id,
          running: current.running,
          lines,
        },
      };
    });

    if (
      event.channel === "stderr" &&
      typeof event.exit_code === "number" &&
      event.exit_code !== 0
    ) {
      showCommandFailureToast(event.execution_id, event.command_id);
    }
  }, [evdata?.command_output, showCommandFailureToast]);

  useEffect(() => {
    const event: CommandTerminatedNotification | undefined =
      evdata?.command_terminated;
    if (!event) {
      return;
    }

    setCommandSessions((previous) => {
      const current = previous[event.execution_id] ?? {
        execution_id: event.execution_id,
        command_id: event.command_id,
        command: event.command_id,
        args: [],
        started_at: event.finished_at,
        running: false,
        lines: [],
      };

      return {
        ...previous,
        [event.execution_id]: {
          ...current,
          running: false,
          success: event.success,
          exit_code: event.exit_code,
          finished_at: event.finished_at,
          error: event.error,
        },
      };
    });

    if (event.exit_code !== 0) {
      showCommandFailureToast(event.execution_id, event.command_id);
    }
  }, [evdata?.command_terminated, showCommandFailureToast]);

  const selectedCommandSession =
    commandDialogExecutionId === null
      ? undefined
      : commandSessions[commandDialogExecutionId];

  const downloadCommandOutput = () => {
    if (!selectedCommandSession) {
      return;
    }
    const lines = getCommandOutputLines(selectedCommandSession)
      .map((line) => `[${line.channel}] ${line.line}`)
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
      <CommandOutputDialog
        open={commandDialogOpen}
        session={selectedCommandSession}
        onClose={() => setCommandDialogOpen(false)}
        onDownload={downloadCommandOutput}
      />
    </>
  );
}
