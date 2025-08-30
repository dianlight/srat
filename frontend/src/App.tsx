import { Backdrop, CircularProgress } from "@mui/material";
import Container from "@mui/material/Container";
import { useEffect, useRef, useState } from "react";
//import { DirtyDataContext, ModeContext } from "./Contexts";
import { Footer } from "./components/Footer";
import { NavBar } from "./components/NavBar";
import { RollbarPersonUpdater } from "./components/RollbarPersonUpdater";
import TelemetryModal from "./components/TelemetryModal";
import { useHealth } from "./hooks/healthHook";
import { useTelemetryModal } from "./hooks/useTelemetryModal";
import { useTelemetryInitialization } from "./hooks/useTelemetryInitialization";

export function App() {
	const [errorInfo, _setErrorInfo] = useState<string>("");
	const mainArea = useRef<HTMLDivElement>(null);
	const { health: status, isLoading, error: herror } = useHealth();
	const { shouldShow: showTelemetryModal, dismiss: dismissTelemetryModal } = useTelemetryModal();

	// Initialize telemetry service based on settings
	useTelemetryInitialization();

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
		function onBeforeUnload(ev: BeforeUnloadEvent) {
			if (sessionStorage.getItem("srat_dirty") === "true") {
				ev.preventDefault();
				return "Are you sure you want to leave? Your changes will be lost.";
			}
			return;
		}

		window.addEventListener("beforeunload", onBeforeUnload);

		return () => {
			//ws.unsubscribe(mhuuid);
			//ws.unsubscribe(drtyuid);
			window.removeEventListener("beforeunload", onBeforeUnload);
		};
	}, []);

	return (
		/*     <ModeContext.Provider value={status}>
				 <DirtyDataContext.Provider value={dirtyData}>*/
		<>
			{/* Update Rollbar person information when machine_id becomes available */}
			<RollbarPersonUpdater />
			<Container
				maxWidth="lg"
				disableGutters={true}
				sx={{
					minHeight: "100vh",
					display: "flex",
					flexDirection: "column",
				}}
			>
				<NavBar error={errorInfo} bodyRef={mainArea} />
				<div ref={mainArea} className="fullBody" style={{ flexGrow: 1 }}></div>
				<Footer healthData={status} />
			</Container>
			<Backdrop
				sx={(theme) => ({ color: "#fff", zIndex: theme.zIndex.drawer + 1 })}
				open={status.alive === false || isLoading}
				content={isLoading ? "Loading..." : "Server is not reachable"}
			>
				<CircularProgress color="inherit" />
			</Backdrop>
			<TelemetryModal
				open={showTelemetryModal}
				onClose={dismissTelemetryModal}
			/>
		</>
		/*
			</DirtyDataContext.Provider>
		</ModeContext.Provider>*/
	);
}
