import { Backdrop, CircularProgress } from "@mui/material";
import Container from "@mui/material/Container";
import { useEffect, useRef, useState } from "react";
//import { DirtyDataContext, ModeContext } from "./Contexts";
import { Provider as RollbarProvider } from "@rollbar/react";
import { Footer } from "./components/Footer";
import { NavBar } from "./components/NavBar";
import TelemetryModal from "./components/TelemetryModal";
import { ErrorBoundaryWrapper } from "./components/ErrorBoundaryWrapper";
import { useHealth } from "./hooks/healthHook";
import { useTelemetryModal } from "./hooks/useTelemetryModal";
import { useTelemetryInitialization } from "./hooks/useTelemetryInitialization";
import { createRollbarConfig } from "./services/telemetryService";
import telemetryService from "./services/telemetryService";

export function App() {
	//const [status, setStatus] = useState<DtoHealthPing>({ alive: false, read_only: true });
	//    const [dirtyData, setDirtyData] = useState<DtoDataDirtyTracker>({});
	const [errorInfo, _setErrorInfo] = useState<string>("");
	const mainArea = useRef<HTMLDivElement>(null);
	/*
	const status = useSSE(DtoEventType.Heartbeat, { alive: false, read_only: true }, {
		parser(input: any): DtoHealthPing {
			console.log("Got heartbeat", input)
			return JSON.parse(input);
		},
	});
	*/
	const { health: status, isLoading, error: herror } = useHealth();
	const { shouldShow: showTelemetryModal, dismiss: dismissTelemetryModal } = useTelemetryModal();

	// Initialize telemetry service based on settings
	useTelemetryInitialization();
	//const [sseEventSource, sseStatus] = useEventSource(apiContext.instance.getUri() + "/sse", true)

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
		/*
		const mhuuid = ws.subscribe<DtoHealthPing>(DtoEventType.EventHeartbeat, (data) => {
			// console.log("Got heartbeat", data)
			if (timeoutpid) clearTimeout(timeoutpid);
			if (process.env.NODE_ENV === "development" && data.read_only === true) {
				console.log("Dev mode force read_only to false");
				data.read_only = false;
			}
			//data.last_time = Date.now();
			setStatus(data);
			function timeoutStatus() {
				setStatus({ alive: false, read_only: true });
			}
			timeoutpid = setTimeout(timeoutStatus, 10000);
		})
		ws.onError((event) => {
			console.error("WS error2", event.type, JSON.stringify(event))
			setStatus({ alive: false, read_only: true });
			setErrorInfo(JSON.stringify(event));
		})
		const drtyuid = ws.subscribe<DtoDataDirtyTracker>(DtoEventType.EventDirty, (data) => {
			console.log("Got dirty data", data)
			setDirtyData(data);
			sessionStorage.setItem("srat_dirty", (Object.values(data).reduce((acc, value) => acc + (value ? 1 : 0), 0) > 0) ? "true" : "false");
		})
		*/
		/*
		if (sseEventSource) {
			sseEventSource.onerror = () => {
				setStatus({ alive: false, read_only: true });
				setErrorInfo("SSE connection error");
			}
		}
		*/
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
	/*
		useEventSourceListener(
			sseEventSource,
			[DtoEventType.EventHeartbeat],
			(evt) => {
				//console.log("SSE EventHeartbeat", evt);
				setStatus(JSON.parse(evt.data));
				setDirtyData(status.dirty_tracking || {});
			},
			[sseStatus],
		);
	*/
	return (
		/*     <ModeContext.Provider value={status}>
				 <DirtyDataContext.Provider value={dirtyData}>*/
		<RollbarProvider config={createRollbarConfig(telemetryService.getAccessToken())}>
			<ErrorBoundaryWrapper>
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
			</ErrorBoundaryWrapper>
		</RollbarProvider>
		/*
			</DirtyDataContext.Provider>
		</ModeContext.Provider>*/
	);
}
