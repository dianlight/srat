import { useContext, useEffect, useRef, useState } from "react";
import { apiContext, ModeContext, wsContext } from "./Contexts";
import type { Api, ConfigShare, ConfigShares, ConfigUser } from "./srat";
import { ObjectTable } from "./components/ObjectTable";
import { useForm } from "react-hook-form";
import useSWR from "swr";
import { InView } from "react-intersection-observer";
import { FormSelect } from "@materializecss/materialize";


interface ShareEditProps extends ConfigShare {
    name: string,
    doCreate?: boolean
}

export function Shares() {
    const api = useContext(apiContext);
    const mode = useContext(ModeContext);
    const [status, setStatus] = useState<ConfigShares>({});
    const [selected, setSelected] = useState<[string, ConfigShare] | null>(null);
    const ws = useContext(wsContext);
    const [errorInfo, setErrorInfo] = useState<string>('')
    const formRef = useRef<HTMLFormElement>(null);
    const users = useSWR<ConfigUser[]>('/users', () => api.users.usersList().then(res => res.data));
    const admin = useSWR<ConfigUser>('/admin/user', () => api.admin.userList().then(res => res.data));

    const { register, handleSubmit, watch, formState: { errors } } = useForm<ShareEditProps>(
        {

            values: { ...selected?.[1], name: selected?.[0] || "" }
        },
    );

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

        ws.subscribe<ConfigShares>('shares', (data) => {
            console.log("Got shares", data)
            setStatus(data);
        })
    }, [])

    function onSubmitDeleteShare(data?: string) {
        if (!data) return
        api.share.shareDelete(data).then((res) => {
            setSelected(null);
            //users.mutate();
        }).catch(err => {
            console.error(err);
            //setErrorInfo(JSON.stringify(err));
        })
    }

    function onSubmitEditShare(data: ShareEditProps) {
        if (!data.name || !data.path) {
            setErrorInfo('Unable to update share!');
            return;
        }

        // Save Data
        console.log(data);
        if (data.doCreate) {
            api.share.shareCreate(data.name, data).then((res) => {
                setErrorInfo('')
                setSelected(null);
            }).catch(err => {
                setErrorInfo(JSON.stringify(err));
            })
            return;
        } else {
            api.share.shareUpdate(data.name, data).then((res) => {
                setErrorInfo('')
                //setSelectedUser(null);
                //users.mutate();
            }).catch(err => {
                setErrorInfo(JSON.stringify(err));
            })
        }
        //formRef.current?.reset();
        return false;
    }




    return <>
        <div id="share" className="modal  modal-fixed-footer">
            <div className="modal-content">
                <h4>{selected ? selected[0] : ""}</h4>
                <p>Share Attributes:</p>
                <ObjectTable object={selected?.[1]} />
            </div>
            <div className="modal-footer">
                <a href="#!" className="modal-close waves-effect btn-flat">Close</a>
            </div>
        </div>
        <div id="delshare" className="modal  modal-fixed-footer">
            <div className="modal-content">
                <h4>Delete {selected?.[0]}? </h4>
                <p>If you delete this share, all of their configurations will be deleted.</p>
            </div>
            <div className="modal-footer">
                <a href="#!" className="modal-close waves-effect btn-flat">Disagree</a>
                <a href="#!" onClick={() => onSubmitDeleteShare(selected?.[0])} className="modal-close waves-effect btn-flat">Agree</a>
            </div>
        </div>
        <InView as="div" id="editshare" className="modal modal-fixed-footer" onChange={(inView, entry) => {
            inView && formRef.current && FormSelect.init(formRef.current.querySelectorAll('select'), {
                dropdownOptions: {
                    container: document.body
                }
            })
        }}>
            <div className="modal-content">
                <h4>{selected ? "Edit " + selected[0] : "New Share"}</h4>
                <p>Share attributes:{JSON.stringify(selected?.[1])}</p>
                <form id="editshareform" ref={formRef} className="row" onSubmit={handleSubmit(onSubmitEditShare)} action="#">
                    <p className="col s12 m6 input-field inline">
                        <i className="material-icons prefix">folder_shared</i>
                        <input id="name" type="text" className="validate" placeholder=" "  {...register("name", { required: true })} />
                        <label htmlFor="name">Share Name</label>
                        {errors.name &&
                            <span className="supporting-text" data-error="wrong" data-success="right">Supporting text for additional information</span>}
                    </p>
                    <p className="col s12 m6 input-field inline">
                        <select id="usage" {...register("usage", { required: true, disabled: selected?.[1].usage === "native" })}>
                            <option value="">Choose your option</option>
                            <option value="native">Native</option>
                            <option value="backup">Backup</option>
                            <option value="media">Media</option>
                            <option value="share">Share</option>
                        </select>
                        <label htmlFor="usage">Usage</label>
                    </p>
                    <p className="col s12 m6 input-field inline">
                        <i className="material-icons prefix">dns</i>
                        <i onClick={() => true} className="material-icons suffix">travel_explore</i>
                        <input id="path" type="text" className="validate" placeholder=" " maxLength={20} {...register("path", { required: true })} disabled={selected?.[1].usage === "native"} />
                        <label htmlFor="path">Path</label>
                        {errors.path &&
                            <span className="supporting-text" data-error="wrong" data-success="right">Supporting text for additional information</span>}
                    </p>
                    <p className="col s12 m6 input-field inline">
                        <label>
                            <input type="checkbox" className="filled-in" {...register("timemachine")} disabled={selected?.[1].usage === "native"} />
                            <span>Time Machine</span>
                        </label>
                    </p>
                    <p className="col s12 m6 input-field inline">
                        <label>
                            <input type="checkbox" className="filled-in" {...register("disabled")} />
                            <span>Disabled</span>
                        </label>
                    </p>
                    <p className="input-field col s12 m6">
                        <select id="share_users" multiple {...register("users")}>
                            <option value={admin.data?.username} key={admin.data?.username + "_rw"}>{admin.data?.username}</option>
                            {users.data?.map(user => <option key={user.username + "_rw"} value={user.username}>{user.username}</option>)}
                        </select>
                        <label htmlFor="share_users">Read and Write users</label>
                    </p>
                    <p className="input-field col s12 m6">
                        <select id="share_rousers" multiple {...register("ro_users")}>
                            <option value={admin.data?.username} key={admin.data?.username + "_rw"}>{admin.data?.username}</option>
                            {users.data?.map(user => <option key={user.username + "_ro"} value={user.username}>{user.username}</option>)}
                        </select>
                        <label htmlFor="share_rousers">Read Only users</label>
                    </p>
                </form>
            </div>
            <div className="modal-footer">
                <a href="#!" className="modal-close waves-effect btn-flat">Disagree</a>
                <button type="submit" form="editshareform" className="modal-close waves-effect btn-flat">Agree</button>
            </div>
        </InView>
        <ul className="collection" >
            {Object.entries(status).map(([share, props]) =>
                < li className="collection-item avatar" key={share} >
                    <i className="material-icons circle" > folder </i>
                    < span className="title" ><a href="#share" onClick={() => setSelected([share, props])} className="modal-trigger">{share}</a></span>
                    < p > {props.fs} < br />
                        {props.path}
                    </p>
                    {mode.read_only ||
                        <div className="row secondary-content">
                            <div className="col offset-s10 s1"><a href="#editshare" onClick={() => setSelected([share, props])} className="btn-floating blue waves-light red modal-trigger"> <i className="material-icons"> settings </i></a></div>
                            <div className="col s1"><a href="#delshare" onClick={() => setSelected([share, props])} className="btn-floating waves-effect waves-light red modal-trigger"> <i className="material-icons"> share </i></a></div>
                        </div>
                    }
                </li>
            )}
        </ul>
    </>
}