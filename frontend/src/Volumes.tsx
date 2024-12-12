import { useContext, useEffect, useState } from "react";
import { apiContext, wsContext } from "./Contexts";
import type { Api, MainShare, MainVolume } from "./srat";

export function Volumes() {
    const api = useContext(apiContext);
    const [status, setStatus] = useState<MainVolume[]>([]);
    const ws = useContext(wsContext);

    useEffect(() => {
        ws.subscribe<MainVolume[]>('volumes', (data) => {
            console.log("Got volumes", data)
            setStatus(data);
        })
    }, [])


    return <ul className="collection" >
        {status.map((volume, idx) =>
            < li className="collection-item avatar" key={idx} >
                <i className="material-icons circle" > disk </i>
                < span className="title" > {volume.label} </span>
                < p > {volume.fstype} < br />
                    {volume.mountpoint}
                </p>
                <div className="row secondary-content">
                    <div className="col offset-s10 s1"><a href="#edituser" className="btn-floating blue waves-light red modal-trigger"> <i className="material-icons"> folder_shared </i></a></div>
                    <div className="col s1"><a href="#deluser" className="btn-floating waves-effect waves-light red modal-trigger"> <i className="material-icons"> delete_forever </i></a></div>
                </div>
            </li>
        )}
    </ul>
}