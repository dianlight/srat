import { createPortal } from "react-dom";
import { Shares } from "./pages/Shares";
import { Users } from "./pages/Users";
import { NavBar } from "./components/NavBar";
import { Footer } from "./components/Footer";
import { useContext, useRef, useState } from "react";
import { Volumes } from "./pages/Volumes";
import { SmbConf } from "./pages/SmbConf";
import { ModeContext, wsContext } from "./Contexts";
import type { MainHealth } from "./srat";
import { ErrorBoundaryContext, useErrorBoundary } from "react-use-error-boundary";
import { Settings } from "./pages/Settings";
import Container from "@mui/material/Container";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Link from "@mui/material/Link";
import type { SvgIconProps } from "@mui/material/SvgIcon";
import SvgIcon from "@mui/material/SvgIcon";

function Copyright() {
    return (
        <Typography
            variant="body2"
            align="center"
            sx={{
                color: 'text.secondary',
            }}
        >
            {'Copyright © '}
            <Link color="inherit" href="https://mui.com/">
                Your Website
            </Link>{' '}
            {new Date().getFullYear()}.
        </Typography>
    );
}

function LightBulbIcon(props: SvgIconProps) {
    return (
        <SvgIcon {...props}>
            <path d="M9 21c0 .55.45 1 1 1h4c.55 0 1-.45 1-1v-1H9v1zm3-19C8.14 2 5 5.14 5 9c0 2.38 1.19 4.47 3 5.74V17c0 .55.45 1 1 1h6c.55 0 1-.45 1-1v-2.26c1.81-1.27 3-3.36 3-5.74 0-3.86-3.14-7-7-7zm2.85 11.1l-.85.6V16h-4v-2.3l-.85-.6C7.8 12.16 7 10.63 7 9c0-2.76 2.24-5 5-5s5 2.24 5 5c0 1.63-.8 3.16-2.15 4.1z" />
        </SvgIcon>
    );
}


export function App() {
    const ws = useContext(wsContext);
    const [status, setStatus] = useState<MainHealth>({ alive: false, read_only: true });
    const [errorInfo, setErrorInfo] = useState<string>('')
    const [error, resetError] = useErrorBoundary(
        (error, errorInfo) => console.error(error, errorInfo)
    );
    const mainArea = useRef<HTMLDivElement>(null);


    function onLoadHandler() {
        ws.subscribe<MainHealth>('heartbeat', (data) => {
            //console.log("Got heartbeat", data)
            setStatus(data);
        })
        ws.onError((event) => {
            console.error("WS error2", event.type, JSON.stringify(event))
            setStatus({ alive: false, read_only: true });
            setErrorInfo(JSON.stringify(event));
        })
    }

    if (error) {
        setTimeout(() => { resetError() }, 5000);
        return <div className="row center">
            <h5 className="header col s12 light">Connecting to the server...</h5>
            <div className="spinner-layer spinner-blue">
                <div className="circle-clipper left">
                    <div className="circle"></div>
                </div>
                <div className="gap-patch">
                    <div className="circle"></div>
                </div>
                <div className="circle-clipper right">
                    <div className="circle"></div>
                </div>
            </div>
        </div>
    }

    return <ModeContext.Provider value={status}>
        <Container onLoad={onLoadHandler} maxWidth="lg" disableGutters={true} sx={{ minHeight: "100%" }}>
            <NavBar error={errorInfo} bodyRef={mainArea} />
            <div ref={mainArea} className="fullBody"></div>
            <Footer />
        </Container>
    </ModeContext.Provider>
}
