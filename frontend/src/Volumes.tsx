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
        {status.map((volume) =>
            < li className="collection-item avatar" key={volume.serial_number} >
                <i className="material-icons circle" > disk </i>
                < span className="title" > {volume.label} </span>
                < p > {volume.fstype} < br />
                    {volume.mountpoint}
                </p>
                < a href="#!" className="secondary-content" > <i className="material-icons" > grade </i></a >
            </li>
        )}
    </ul>
}