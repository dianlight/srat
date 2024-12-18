import { createPortal } from "react-dom";
import { Shares } from "./Shares";
import { Users } from "./Users";
import { NavBar } from "./NavBar";
import { Footer } from "./Footer";
import { Tabs, AutoInit } from "@materializecss/materialize"
import { useRef } from "react";
import { Volumes } from "./Volumes";


export function Page(/*props: { message: string }*/) {

    function onLoadHandler() {
        AutoInit();
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
        <div id="settings" className="col s12">Settings</div>
    </div>
}
