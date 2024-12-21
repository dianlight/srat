import { createPortal } from "react-dom";
import { Shares } from "./Shares";
import { Users } from "./Users";
import { NavBar } from "./NavBar";
import { Footer } from "./Footer";
import { Tabs, AutoInit, FormSelect } from "@materializecss/materialize"
import { useContext, useRef, useState } from "react";
import { Volumes } from "./Volumes";
import { SmbConf } from "./pages/SmbConf";
import { ModeContext, wsContext } from "./Contexts";
import type { MainHealth } from "./srat";


export function Page(/*props: { message: string }*/) {
    const ws = useContext(wsContext);
    const [status, setStatus] = useState<MainHealth>({ alive: false, read_only: true });
    const [errorInfo, setErrorInfo] = useState<string>('')


    function onLoadHandler() {
        AutoInit();
        FormSelect.init(document.querySelectorAll('select'), {
            dropdownOptions: {
                container: document.body
            }
        });
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

    return <ModeContext.Provider value={status}>
        <div onLoad={onLoadHandler} className="row" style={{ marginTop: "50px" }}>

            {createPortal(
                <NavBar error={errorInfo} />,
                document.getElementById('navbar')!
            )}
            {createPortal(
                <Footer />,
                document.getElementById('footer')!
            )}
            {/*props.message*/}
            <div id="shares" className="col s12"><Shares /></div>
            <div id="volumes" className="col s12"><Volumes /></div>
            <div id="users" className="col s12"><Users /></div>
            <div id="settings" className="col s12">Settings... *****|{status.read_only}|*****</div>
            <div id="smbconf" className="col s12"><SmbConf /></div>
        </div>
    </ModeContext.Provider>
}
