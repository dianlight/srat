import * as ReactDOM from 'react-dom/client';
import CssBaseline from '@mui/material/CssBaseline';
import { App } from "./App.tsx"
import "./css/style.css"
import { ErrorBoundaryContext } from 'react-use-error-boundary';
import '@fontsource/roboto/300.css';
import '@fontsource/roboto/400.css';
import '@fontsource/roboto/500.css';
import '@fontsource/roboto/700.css';
import './img/favicon.ico';
import '@mui/icons-material';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import { ConfirmProvider } from "material-ui-confirm";


const theme = createTheme({
    colorSchemes: {
        dark: true,
    },
});

const root = ReactDOM.createRoot(document.getElementById('root')!);
root.render(
    <ErrorBoundaryContext>
        <ThemeProvider theme={theme} noSsr>
            <CssBaseline />
            <ConfirmProvider>
                <App />
            </ConfirmProvider>
        </ThemeProvider>
    </ErrorBoundaryContext>)
