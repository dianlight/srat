import { NavBar } from "./components/NavBar";
import { Footer } from "./components/Footer";
import { useContext, useEffect, useRef, useState } from "react";
import { DirtyDataContext, ModeContext, wsContext as ws } from "./Contexts";
import { MainEventType, type ConfigConfigSectionDirtySate, type MainHealth } from "./srat";
import { useErrorBoundary } from "react-use-error-boundary";
import Container from "@mui/material/Container";


export function App() {
    const [status, setStatus] = useState<MainHealth>({ alive: false, read_only: true });
    const [dirtyData, setDirtyData] = useState<ConfigConfigSectionDirtySate>({});
    const [errorInfo, setErrorInfo] = useState<string>('')
    const [error, resetError] = useErrorBoundary(
        (error, errorInfo) => console.error(error, errorInfo)
    );
    const mainArea = useRef<HTMLDivElement>(null);


    useEffect(() => {
        const mhuuid = ws.subscribe<MainHealth>(MainEventType.EventHeartbeat, (data) => {
            // console.log("Got heartbeat", data)
            setStatus(data);
        })
        ws.onError((event) => {
            console.error("WS error2", event.type, JSON.stringify(event))
            setStatus({ alive: false, read_only: true });
            setErrorInfo(JSON.stringify(event));
        })
        const drtyuid = ws.subscribe<ConfigConfigSectionDirtySate>(MainEventType.EventDirty, (data) => {
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

    return (
        <ModeContext.Provider value={status}>
            <DirtyDataContext.Provider value={dirtyData}>
                <Container maxWidth="lg" disableGutters={true} sx={{ minHeight: "100%" }}>
                    <NavBar healthData={status} error={errorInfo} bodyRef={mainArea} />
                    <div ref={mainArea} className="fullBody"></div>
                    <Footer healthData={status} />
                </Container>
            </DirtyDataContext.Provider>
        </ModeContext.Provider>)
}
