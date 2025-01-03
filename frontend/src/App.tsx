import { NavBar } from "./components/NavBar";
import { Footer } from "./components/Footer";
import { useContext, useEffect, useRef, useState } from "react";
import { DirtyDataContext, ModeContext, wsContext as ws } from "./Contexts";
import { DtoEventType, type DtoDataDirtyTracker, type DtoHealthPing } from "./srat";
import { useErrorBoundary } from "react-use-error-boundary";
import Container from "@mui/material/Container";
import { Backdrop, CircularProgress, Typography } from "@mui/material";


export function App() {
    const [status, setStatus] = useState<DtoHealthPing>({ alive: false, read_only: true });
    const [dirtyData, setDirtyData] = useState<DtoDataDirtyTracker>({});
    const [errorInfo, setErrorInfo] = useState<string>('')
    const [error, resetError] = useErrorBoundary(
        (error, errorInfo) => console.error(error, errorInfo)
    );
    const mainArea = useRef<HTMLDivElement>(null);

    var timeoutpid: ReturnType<typeof setTimeout>

    useEffect(() => {
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
        function onBeforeUnload(ev: BeforeUnloadEvent) {
            if (sessionStorage.getItem("srat_dirty") === "true") {
                ev.preventDefault();
                return "Are you sure you want to leave? Your changes will be lost.";
            }
            return
        };

        window.addEventListener("beforeunload", onBeforeUnload);

        return () => {
            ws.unsubscribe(mhuuid);
            ws.unsubscribe(drtyuid);
            window.removeEventListener("beforeunload", onBeforeUnload);
        };
    }, [])

    if (error) {
        setTimeout(() => { resetError() }, 5000);
        return <Typography> Connecting to the server... </Typography>
    }

    return (
        <ModeContext.Provider value={status}>
            <DirtyDataContext.Provider value={dirtyData}>
                <Container maxWidth="lg" disableGutters={true} sx={{ minHeight: "100%" }}>
                    <NavBar healthData={status} error={errorInfo} bodyRef={mainArea} />
                    <div ref={mainArea} className="fullBody"></div>
                    <Footer healthData={status} />
                </Container>
                <Backdrop
                    sx={(theme) => ({ color: '#fff', zIndex: theme.zIndex.drawer + 1 })}
                    open={status.alive === false}
                >
                    <CircularProgress color="inherit" />
                </Backdrop>
            </DirtyDataContext.Provider>
        </ModeContext.Provider>)
}
