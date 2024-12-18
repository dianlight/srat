import { useContext, useEffect, useState } from "react";
import { apiContext, wsContext } from "./Contexts";
import type { Api, MainShare, MainShares } from "./srat";

export function Shares() {
    const api = useContext(apiContext);
    const [status, setStatus] = useState<MainShares>({});
    const ws = useContext(wsContext);

    useEffect(() => {
        /*
        async function getShareList() {
            api.shares.sharesList().then((res) => {
                setStatus(res.data || []);
                setErrorInfo('')
                setTimeout(getShareList, 5000);
            }).catch(err => {
                setErrorInfo(JSON.stringify(err));
                setStatus({});
                setTimeout(getShareList, 5000);
            })
        }
        */

        // setTimeout(getShareList, 1000);

        ws.subscribe<MainShares>('shares', (data) => {
            //console.log("Got shares", data)
            setStatus(data);
        })
    }, [])


    return <ul className="collection" >
        {Object.entries(status).map(([share, props]) =>
            < li className="collection-item avatar" key={share} >
                <i className="material-icons circle" > folder </i>
                < span className="title" > {share} </span>
                < p > {props.fs} < br />
                    {props.path}
                </p>
                <div className="row secondary-content">
                    <div className="col offset-s10 s1"><a href="#edituser" className="btn-floating blue waves-light red modal-trigger"> <i className="material-icons"> settings </i></a></div>
                    <div className="col s1"><a href="#deluser" className="btn-floating waves-effect waves-light red modal-trigger"> <i className="material-icons"> share_off </i></a></div>
                </div>
            </li>
        )}
    </ul>
}