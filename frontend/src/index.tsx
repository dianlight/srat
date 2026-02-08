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
//import { type Listener, type Source, SSEProvider } from "react-hooks-sse";
import { Provider } from "react-redux";
import { BrowserRouter } from "react-router";
import { Provider as RollbarProvider } from "@rollbar/react";
import { ErrorBoundaryWrapper } from "./components/ErrorBoundaryWrapper";
import { ConsoleErrorToRollbar } from "./components/ConsoleErrorToRollbar";
import { store } from "./store/store.ts";
import { TourProvider, } from '@reactour/tour'
import { get } from "react-hook-form";
import { getApiUrl, getCurrentEnv } from "./macro/Environment.ts" with { type: 'macro' };

declare module '@mui/material/styles' {
	interface TypographyVariants {
		supper: React.CSSProperties;
	}

	// allow configuration using `createTheme()`
	interface TypographyVariantsOptions {
		supper?: React.CSSProperties;
	}
}

declare module '@mui/material/Typography' {
	interface TypographyPropsVariantOverrides {
		supper: true;
	}
}

const theme = createTheme({
	cssVariables: {
		colorSchemeSelector: "data-color-mode",
	},
	colorSchemes: {
		light: true,
		dark: true,
	},
	typography: {
		supper: {
			fontSize: "0.50rem",
			fontWeight: 600,
		}
	},
	components: {
		MuiTypography: {
			defaultProps: {
				variantMapping: {
					supper: "sup",
				},
			},
		},
	},
});

if (import.meta.hot) {
	console.debug("✅ Hot Module Replacement (HMR) is enabled!");
}

if (getCurrentEnv() === "development") {
	console.debug("👷‍♂️ Running in development mode");
} else if (getCurrentEnv() === "remote") {
	console.debug(`🌐 Running in remote mode: ${getApiUrl()}`);
} else if (getCurrentEnv() === "production") {
	console.debug("🚀 Running in production mode");
} else {
	console.debug(`ℹ️ Running in unknown mode: ${getCurrentEnv()}`);
}

const disableBody = (target: any) => {
	// Use CSS-based scroll prevention instead of aria-hidden to avoid accessibility issues
	console.trace("Disabling body scroll", target);
	document.body.style.overflow = 'hidden';
	document.body.style.paddingRight = '0px'; // Prevent layout shift
};
const enableBody = (target: any) => {
	console.trace("Enabling body scroll", target);
	document.body.style.overflow = '';
	document.body.style.paddingRight = '';
}

const root = import.meta.hot.data.root ??= ReactDOM.createRoot(document.getElementById("root")!);
root.render(
	<RollbarProvider config={{}} >
		<CssBaseline />
		<ErrorBoundaryWrapper>
			<ThemeProvider theme={theme} noSsr>
				<CssBaseline />
				<Provider store={store}>
					<ConsoleErrorToRollbar />
					<ConfirmProvider>
						<StrictMode>
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
						</StrictMode>
					</ConfirmProvider>
				</Provider>
			</ThemeProvider>
		</ErrorBoundaryWrapper>
	</RollbarProvider>,
);
