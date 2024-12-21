import { createPortal } from "react-dom";
import { Shares } from "./Shares";
import { Users } from "./Users";
import { NavBar } from "./NavBar";
import { Footer } from "./Footer";
import { Tabs, AutoInit, FormSelect } from "@materializecss/materialize"
import { useRef } from "react";
import { Volumes } from "./Volumes";
import { SmbConf } from "./pages/SmbConf";


export function Page(/*props: { message: string }*/) {

    function onLoadHandler() {
        AutoInit();
        FormSelect.init(document.querySelectorAll('select'), {
            dropdownOptions: {
                container: document.body
            }
        });
    }

    return <div onLoad={onLoadHandler} className="row" style={{ marginTop: "50px" }}>
        {createPortal(
            <NavBar />,
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
        <div id="settings" className="col s12">Settings...</div>
        <div id="smbconf" className="col s12"><SmbConf /></div>
    </div>
}
