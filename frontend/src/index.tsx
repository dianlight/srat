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
import { disableBodyScroll, enableBodyScroll } from 'body-scroll-lock'

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
		colorSchemeSelector: "class",
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
	<RollbarProvider>
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
