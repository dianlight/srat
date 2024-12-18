import { useContext, useEffect, useState } from "react";
import { apiContext, wsContext } from "./Contexts";
import type { Api, MainShare, MainVolume } from "./srat";
import { ObjectTable } from "./components/ObjectTable";

export function Volumes() {
    const api = useContext(apiContext);
    const [status, setStatus] = useState<MainVolume[]>([]);
    const [selected, setSelected] = useState<MainVolume | null>(null);
    const ws = useContext(wsContext);

    useEffect(() => {
        ws.subscribe<MainVolume[]>('volumes', (data) => {
            console.log("Got volumes", data)
            setStatus(data);
        })
    }, [])


    return <>
        <div id="volume" className="modal">
            <div className="modal-content">
                <h4>{selected?.label} ({selected?.device})</h4>
                <p>Volume Attributes:</p>
                <ObjectTable object={selected} />
            </div>
            <div className="modal-footer">
                <a href="#!" className="modal-close waves-effect btn-flat">Close</a>
            </div>
        </div>
        <ul className="collection" >
            {status.filter((vol => vol.fstype !== 'erofs')).map((volume, idx) =>
                < li className="collection-item avatar" key={idx} >
                    <i className="material-icons circle green">{volume.label === 'hassos-data' ? "lock" : "hard_disk"}</i>
                    <span className="title"><a href="#volume" onClick={() => setSelected(volume)} className="modal-trigger">{volume.label}</a></span>
                    <div className="row" >
                        <p className="col s1">FS: {volume.fstype}</p>
                        <p className="col s1">MountPath: {volume.mountpoint}</p>
                        <p className="col s1">SN: {volume.serial_number}</p>
                        <p className="col s1">Dev: {volume.device}</p>
                    </div>
                    <div className="row secondary-content">
                        <div className="col offset-s9 s1"><a href="#edituser" className="btn-floating blue waves-light red modal-trigger"> <i className="material-icons"> share </i></a></div>
                        {volume.lsbk?.rm ? <div className="col s1"><a href="#edituser" className="btn-floating blue waves-light red modal-trigger"> <i className="material-icons"> eject </i></a></div> : ""}
                        <div className="col s1"><a href="#deluser" className="btn-floating waves-effect waves-light red modal-trigger"> <i className="material-icons">share</i></a></div>
                    </div>
                </li>
            )}
        </ul>
    </>
}