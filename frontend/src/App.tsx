import { Alert, Backdrop, Button, CircularProgress, Snackbar } from "@mui/material";
import Container from "@mui/material/Container";
import { useEffect, useRef, useState } from "react";
import BaseConfigModal from "./components/BaseConfigModal";
import { Footer } from "./components/Footer";
import GlobalEventMonitor from "./components/GlobalEventTracker";
import { NavBar } from "./components/NavBar";
import TelemetryModal from "./components/TelemetryModal";
import { useBaseConfigModal } from "./hooks/useBaseConfigModal";
import { useTelemetryModal } from "./hooks/useTelemetryModal";
import { useGetApiSettingsAppConfigQuery, usePutApiRestartMutation } from "./store/sratApi";
import { useGetServerEventsQuery } from "./store/wsApi";

// Track Backdrop open state to detect transition
function useBackdropReload(open: boolean, error: any, alive: boolean | undefined) {
	const prevOpen = useRef(open);
	useEffect(() => {
		if (prevOpen.current && !open) {
			// Backdrop just closed
			if (error !== undefined || alive === false) {
				window.location.reload();
			}
		}
		prevOpen.current = open;
	}, [open, error, alive]);
}


export function App() {
	const [errorInfo, setErrorInfo] = useState<string>("");
	const [showAddonConfigChangedBanner, setShowAddonConfigChangedBanner] = useState(false);
	const [isRestartingAddon, setIsRestartingAddon] = useState(false);
	const mainArea = useRef<HTMLDivElement>(null);
	const { data: evdata, isLoading, error: herror } = useGetServerEventsQuery();
	const { data: appConfigResponse } = useGetApiSettingsAppConfigQuery();
	const [restartAddon] = usePutApiRestartMutation();
	const { shouldShow: showTelemetryModal, dismiss: dismissTelemetryModal } = useTelemetryModal();
	const { shouldShow: showBaseConfigModal, dismiss: dismissBaseConfigModal } = useBaseConfigModal();

	// Compute Backdrop open state
	const backdropOpen = evdata?.heartbeat?.alive === false || isLoading || herror !== undefined;
	useBackdropReload(backdropOpen, herror, evdata?.heartbeat?.alive);
	//const { reportError, reportEvent, telemetryMode, isLoading: rollbarLoading } = useRollbarTelemetry();

	// This useEffect handles the automatic reset of errors after a delay.
	// It ensures that a timer is set only when an error occurs, and cleared if the error resolves
	// or the component unmounts. This prevents multiple timers from being created.
	useEffect(() => {
		let timer: ReturnType<typeof setTimeout> | undefined;
		if (herror) {
			timer = setTimeout(() => {
				// With the new error boundary, we don't need to manually reset errors
				console.log('Error auto-reset timer triggered');
			}, 5000);
		}
		return () => {
			if (timer) clearTimeout(timer);
		};
	}, [herror]);

	useEffect(() => {
		if (appConfigResponse && "requires_restart" in appConfigResponse && appConfigResponse.requires_restart) {
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
				open={showAddonConfigChangedBanner && !(evdata?.heartbeat?.alive === false || isLoading || herror !== undefined)}
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
							<Button color="inherit" size="small" onClick={handleReloadWithAddonRestart} disabled={isRestartingAddon}>
								{isRestartingAddon ? "Restarting..." : "Reload"}
							</Button>
						</>
					}
				>
					Addon configuration has changed. Reload required.
				</Alert>
			</Snackbar>
		</>
	);
}
