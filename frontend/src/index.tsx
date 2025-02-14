import * as ReactDOM from 'react-dom/client';
import CssBaseline from '@mui/material/CssBaseline';
import { App } from "./App.tsx"
import "./css/style.css"
import "./img/favicon.ico"
import { ErrorBoundaryContext } from 'react-use-error-boundary';
import '@fontsource/roboto/300.css';
import '@fontsource/roboto/400.css';
import '@fontsource/roboto/500.css';
import '@fontsource/roboto/700.css';
import './img/favicon.ico';
import '@mui/icons-material';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import { ConfirmProvider } from "material-ui-confirm";
import { StrictMode } from 'react';
import { SSEProvider, type Listener, type Source } from 'react-hooks-sse';
import { apiContext } from './Contexts.ts';
import { DtoEventType } from './srat.ts';

const theme = createTheme({
    colorSchemes: {
        dark: true,
    },
});


class SSESource implements Source {
    private eventSource: EventSource;
    private resetTimer?: Timer
    private heartbeatListener: Listener[] = []
    private listeners = new Map<string, Listener[]>()
    private faultCount = 0

    constructor(endpoint: string) {
        this.eventSource = this.newSSEClient(endpoint)
    }

    newSSEClient(endpoint: string): EventSource {
        console.log("Creating SSE client", endpoint);
        let eventSource = new EventSource(endpoint, { withCredentials: true });
        eventSource.onerror = () => {
            console.error("SSE connection error");
            this.heartbeatListener.forEach((func) => func({ data: "{ \"alive\": false, \"read_only\": true }" }));
            this.faultCount++;
            if (this.faultCount > 3 && this.resetTimer === undefined) {
                this.eventSource.close();
                this.resetTimer = setTimeout(() => this.eventSource = this.newSSEClient(endpoint), 5000);
            }
        }
        eventSource.onopen = () => {
            console.log("SSE connection open");
            if (this.resetTimer) clearTimeout(this.resetTimer);
            this.faultCount = 0;
            this.listeners.forEach((values, key) => values.forEach(value => {
                this.eventSource.addEventListener(key, value);
            }));
        }
        return eventSource;
    }

    addEventListener(name: string, listener: Listener): void {
        if (name === DtoEventType.EventHeartbeat) {
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
        this.listeners.get(name)?.splice(this.listeners.get(name)?.indexOf(listener) || 0, 1);
    }
    close(): void {
        this.eventSource.close();
    }
}

const root = ReactDOM.createRoot(document.getElementById('root')!);
root.render(
    <ErrorBoundaryContext>
        <ThemeProvider theme={theme} noSsr>
            <CssBaseline />
            <ConfirmProvider>
                <StrictMode>
                    <SSEProvider source={() => new SSESource(apiContext.instance.getUri() + "/sse")}>
                        <App />
                    </SSEProvider>
                </StrictMode>
            </ConfirmProvider>
        </ThemeProvider>
    </ErrorBoundaryContext>)
