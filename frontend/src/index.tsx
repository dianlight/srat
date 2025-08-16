import CssBaseline from "@mui/material/CssBaseline";
import * as ReactDOM from "react-dom/client";
import { App } from "./App.tsx";
import "./css/style.css";
import "./img/favicon.ico";
import "@fontsource/roboto/300.css";
import "@fontsource/roboto/400.css";
import "@fontsource/roboto/500.css";
import "@fontsource/roboto/700.css";
import "./img/favicon.ico";
import normalizeUrl from 'normalize-url';
import "@mui/icons-material";
import { createTheme, ThemeProvider } from "@mui/material/styles";
import { ConfirmProvider } from "material-ui-confirm";
import { StrictMode } from "react";
import { type Listener, type Source, SSEProvider } from "react-hooks-sse";
import { Provider } from "react-redux";
import { BrowserRouter } from "react-router";
import { Provider as RollbarProvider } from "@rollbar/react";
import { ErrorBoundaryWrapper } from "./components/ErrorBoundaryWrapper";
import { ConsoleErrorToRollbar } from "./components/ConsoleErrorToRollbar";
import { createRollbarConfig } from "./services/telemetryService";
import telemetryService from "./services/telemetryService";
import { apiUrl } from "./store/emptyApi.ts";
import { Supported_events } from "./store/sratApi.ts";
//import { apiContext } from './Contexts.ts';
import { store } from "./store/store.ts";
import { TourProvider, } from '@reactour/tour'
import { disableBodyScroll, enableBodyScroll } from 'body-scroll-lock'
import { DashboardSteps } from "./pages/dashboard/DashboardTourStep.tsx";
import { SharesSteps } from "./pages/shares/SharesTourStep.tsx";
import { VolumesSteps } from "./pages/volumes/VolumesTourStep.tsx";

const theme = createTheme({
	cssVariables: {
		colorSchemeSelector: "class",
	},
	colorSchemes: {
		light: true,
		dark: true,
	},
});

class SSESource implements Source {
	private eventSource: EventSource;
	private resetTimer?: Timer;
	private heartbeatListener: Listener[] = [];
	private listeners = new Map<string, Listener[]>();
	private faultCount = 0;

	constructor(endpoint: string) {
		this.eventSource = this.newSSEClient(endpoint);
	}

	newSSEClient(endpoint: string): EventSource {
		console.log("Creating SSE client", endpoint);
		const eventSource = new EventSource(endpoint, { withCredentials: true });
		eventSource.onerror = () => {
			console.warn("SSE connection error");
			this.heartbeatListener.forEach((func) =>
				func({ data: '{ "alive": false, "read_only": true }' }),
			);
			this.faultCount++;
			if (this.faultCount > 3 && this.resetTimer === undefined) {
				this.eventSource.close();
				this.resetTimer = setTimeout(
					() => (this.eventSource = this.newSSEClient(endpoint)),
					5000,
				);
			}
		};
		eventSource.onopen = () => {
			console.log("SSE connection open");
			if (this.resetTimer) clearTimeout(this.resetTimer);
			this.faultCount = 0;
			this.listeners.forEach((values, key) =>
				values.forEach((value) => {
					this.eventSource.addEventListener(key, value);
				}),
			);
		};
		return eventSource;
	}

	addEventListener(name: string, listener: Listener): void {
		if (name === Supported_events.Heartbeat) {
			this.heartbeatListener.push(listener);
		}
		if (!this.listeners.has(name)) {
			this.listeners.set(name, []);
		}
		this.listeners.get(name)?.push(listener);
		this.eventSource.addEventListener(name, listener);
	}
	removeEventListener(name: string, listener: Listener): void {
		this.eventSource.removeEventListener(name, listener);
		this.listeners
			.get(name)
			?.splice(this.listeners.get(name)?.indexOf(listener) || 0, 1);
	}
	close(): void {
		this.eventSource.close();
	}
}

const disableBody = (target: any) => {
	if (!target) {
		console.warn("No target element provided for disabling body scroll");
		target = document.body; // Default to body if no target is provided
	}
	console.debug("Disabling body scroll", target);
	disableBodyScroll(target);
};
const enableBody = (target: any) => {
	if (!target) {
		console.warn("No target element provided for enabling body scroll");
		target = document.body; // Default to body if no target is provided
	}
	console.debug("Enabling body scroll", target);
	enableBodyScroll(target);
}


const root = ReactDOM.createRoot(document.getElementById("root")!);
root.render(
	<RollbarProvider config={createRollbarConfig(telemetryService.getAccessToken())}>
		{/* Bridge console.error to Rollbar respecting telemetry mode */}
		<ConsoleErrorToRollbar />
		<ErrorBoundaryWrapper>
			<ThemeProvider theme={theme} noSsr>
				<CssBaseline />
				<Provider store={store}>
					<ConfirmProvider>
						<StrictMode>
							<SSEProvider source={() => new SSESource(normalizeUrl(apiUrl + "/api/sse"))}>
								<BrowserRouter>
									<TourProvider
										afterOpen={disableBody}
										beforeClose={enableBody}
										steps={[]}
										styles={{
											popover: (base) => ({
												...base,
												color: theme.palette.text.primary,
												backgroundColor: theme.palette.background.paper,
												borderRadius: 10,
												opacity: 0.9,
											}),
											maskArea: (base) => ({ ...base, rx: 5 }),
											//maskWrapper: (base) => ({ ...base, color: '#ef5a3d' }),
											badge: (base) => ({ ...base, left: 'auto', right: '-0.8125em' }),
											//controls: (base) => ({ ...base, marginTop: 100 }),
											close: (base) => ({ ...base, right: 'auto', color: theme.palette.text.primary, left: 8, top: 8 }),
										}}
									>
										<App />
									</TourProvider>
								</BrowserRouter>
							</SSEProvider>
						</StrictMode>
					</ConfirmProvider>
				</Provider>
			</ThemeProvider>
		</ErrorBoundaryWrapper>
	</RollbarProvider>,
);
