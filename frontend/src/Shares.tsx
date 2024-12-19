import { useContext, useEffect, useState } from "react";
import { apiContext, wsContext } from "./Contexts";
import type { Api, MainShare, MainShares } from "./srat";
import { ObjectTable } from "./components/ObjectTable";

export function Shares() {
    const api = useContext(apiContext);
    const [status, setStatus] = useState<MainShares>({});
    const [selected, setSelected] = useState<[string, MainShare] | null>(null);
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

    function onSubmitDeleteShare(data: string) {
        api.share.shareDelete(data).then((res) => {
            setSelected(null);
            //users.mutate();
        }).catch(err => {
            console.error(err);
            //setErrorInfo(JSON.stringify(err));
        })
    }



    return <>
        <div id="share" className="modal">
            <div className="modal-content">
                <h4>{selected ? selected[0] : ""}</h4>
                <p>Share Attributes:</p>
                <ObjectTable object={selected ? selected[1] : {}} />
            </div>
            <div className="modal-footer">
                <a href="#!" className="modal-close waves-effect btn-flat">Close</a>
            </div>
        </div>
        <div id="delshare" className="modal">
            <div className="modal-content">
                <h4>Delete {selected ? selected[0] : ""}? </h4>
                <p>If you delete this share, all of their configurations will be deleted.</p>
            </div>
            <div className="modal-footer">
                <a href="#!" className="modal-close waves-effect btn-flat">Disagree</a>
                <a href="#!" onClick={() => onSubmitDeleteShare(selected![0])} className="modal-close waves-effect btn-flat">Agree</a>
            </div>
        </div>
        <ul className="collection" >
            {Object.entries(status).map(([share, props]) =>
                < li className="collection-item avatar" key={share} >
                    <i className="material-icons circle" > folder </i>
                    < span className="title" ><a href="#share" onClick={() => setSelected([share, props])} className="modal-trigger">{share}</a></span>
                    < p > {props.fs} < br />
                        {props.path}
                    </p>
                    <div className="row secondary-content">
                        <div className="col offset-s10 s1"><a href="#edituser" className="btn-floating blue waves-light red modal-trigger"> <i className="material-icons"> settings </i></a></div>
                        <div className="col s1"><a href="#delshare" onClick={() => setSelected([share, props])} className="btn-floating waves-effect waves-light red modal-trigger"> <i className="material-icons"> share </i></a></div>
                    </div>
                </li>
            )}
        </ul>
    </>
}