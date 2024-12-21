import { useContext, useEffect, useRef, useState } from "react";
import { apiContext, ModeContext, wsContext } from "./Contexts";
import type { Api, ConfigShare, ConfigShares, ConfigUser } from "./srat";
import { Modal } from "@materializecss/materialize"
import { createPortal } from "react-dom";
import { useForm, type SubmitHandler } from "react-hook-form";
import useSWR from "swr";
import useSWRMutation from "swr/mutation";


interface UsersProps extends ConfigUser {
    isAdmin?: boolean
    doCreate?: boolean
}

export function Users() {
    const api = useContext(apiContext);
    const mode = useContext(ModeContext);
    const users = useSWR<UsersProps[]>('/users', () => api.users.usersList().then(res => res.data));
    const admin = useSWR<UsersProps>('/admin/user', () => api.admin.userList().then(res => res.data));
    const [errorInfo, setErrorInfo] = useState<string>('')
    const [selectedUser, setSelectedUser] = useState<UsersProps>({});
    const [isPassowrdVisible, setIsPasswordVisible] = useState(false);
    const ws = useContext(wsContext);
    const { register, handleSubmit, watch, formState: { errors } } = useForm<UsersProps>(
        {
            defaultValues: {
                username: '',
                password: '',
                isAdmin: false
            },
            values: selectedUser
        },
    );
    //    const onSubmit: SubmitHandler<ConfigUser> = data => console.log(data);
    const formRef = useRef<HTMLFormElement>(null);

    function onSubmitEditUser(data: UsersProps) {
        if (!data.username || !data.password) {
            setErrorInfo('Unable to update user!');
            return;
        }

        data.username = data.username.trim();
        data.password = data.password.trim();

        // Save Data
        console.log(data);
        if (data.doCreate) {
            api.user.userCreate(data).then((res) => {
                setErrorInfo('')
                setSelectedUser({});
                users.mutate();
            }).catch(err => {
                setErrorInfo(JSON.stringify(err));
            })
            return;
        } else if (data.isAdmin) {
            api.admin.userUpdate(data).then((res) => {
                setErrorInfo('')
                admin.mutate();
            }).catch(err => {
                setErrorInfo(JSON.stringify(err));
            })
        } else {
            api.user.userUpdate(data.username, data).then((res) => {
                setErrorInfo('')
                setSelectedUser({});
                users.mutate();
            }).catch(err => {
                setErrorInfo(JSON.stringify(err));
            })
        }
        // formRef.current?.reset();
    }

    function onSubmitDeleteUser(data: UsersProps) {
        if (!data.username) {
            setErrorInfo('Unable to delete user!');
            return;
        }
        api.user.userDelete(data.username).then((res) => {
            setErrorInfo('')
            setSelectedUser({});
            users.mutate();
        }).catch(err => {
            setErrorInfo(JSON.stringify(err));
        })
    }

    /*
    async function getUsersList() {
        api.users.usersList().then((res) => {
            setStatus(res.data || []);
            setErrorInfo('')
            setTimeout(getUsersList, 5000);
        }).catch(err => {
            setErrorInfo(JSON.stringify(err));
            setStatus([]);
            setTimeout(getUsersList, 5000);
        })
    }
    async function getAdminUser() {
        api.admin.userList().then((res) => {
            setAdmin(res.data || {});
            setErrorInfo('')
        }).catch(err => {
            setErrorInfo(JSON.stringify(err));
            setAdmin({});
        })
    }

    useEffect(() => {
        //getUsersList();
        getAdminUser();
    }, []);

    */

    return <>
        <div id="deluser" className="modal">
            <div className="modal-content">
                <h4>Delete {selectedUser.username}? </h4>
                <p>If you delete this user, all of their configurations will be deleted.</p>
            </div>
            <div className="modal-footer">
                <a href="#!" className="modal-close waves-effect btn-flat">Disagree</a>
                <a href="#!" onClick={() => onSubmitDeleteUser(selectedUser)} className="modal-close waves-effect btn-flat">Agree</a>
            </div>
        </div>
        <div id="edituser" className="modal">
            <div className="modal-content">
                <h4>{selectedUser.username ? "Edit " + selectedUser.username : "New User"} {selectedUser.isAdmin ? "(Admin)" : ""}</h4>
                <p>User attributes:</p>
                <form id="edituserform" ref={formRef} className="row" style={{ gap: "1em" }} onSubmit={handleSubmit(onSubmitEditUser)} action="#">
                    <div className="s12 m6 input-field inline">
                        <i className="material-icons prefix">person</i>
                        <input id="username" type="text" className="validate" placeholder=" "  {...register("username", { required: true })} readOnly={selectedUser.username != undefined && !selectedUser.isAdmin} />
                        <label htmlFor="username">Username</label>
                        {errors.username &&
                            <span className="supporting-text" data-error="wrong" data-success="right">Supporting text for additional information</span>}
                    </div>
                    <div className="s12 m6 input-field inline">
                        <i className="material-icons prefix">password</i>
                        <i onClick={() => setIsPasswordVisible(!isPassowrdVisible)} className="material-icons suffix">{isPassowrdVisible ? 'visibility_off' : 'visibility'}</i>
                        <input id="password" type={isPassowrdVisible ? 'text' : 'password'} className="validate" placeholder=" " maxLength={20} {...register("password", { required: true })} />
                        <label htmlFor="password">Password</label>
                        {errors.password &&
                            <span className="supporting-text" data-error="wrong" data-success="right">Supporting text for additional information</span>}
                    </div>
                </form>
            </div>
            <div className="modal-footer">
                <a href="#!" className="modal-close waves-effect btn-flat">Disagree</a>
                <input type="submit" form="edituserform" className="modal-close waves-effect btn-flat" value="Agree" />
            </div>
        </div>
        <ul className="collection" >
            {/*admin.error && <p> {admin.error} </p>*/}
            < li className="collection-item avatar" key={admin.data?.username}>
                <i className="material-icons circle red" > account_box </i>
                < span className="title" > {admin.data?.username} </span>
                {/*< p > First Line < br />
                Second Line
            </p>*/}
                {mode.read_only ||
                    < a href="#edituser" className="secondary-content btn-floating blue waves-light red modal-trigger" onClick={() => setSelectedUser({ ...{ isAdmin: true }, ...admin.data })} > <i className="material-icons" > edit </i></a >
                }
            </li>
            {/*users.error && <p> {users.error} </p>*/}
            {users.data?.map((user) =>
                <li className="collection-item avatar" key={user.username}>
                    <i className="material-icons circle green"> account_box </i>
                    <span className="title"> {user.username} </span>
                    {/*< p > {props.fs} < br />
                   {props.path}
                </p>*/}
                    {mode.read_only ||
                        <div className="row secondary-content">
                            <div className="col offset-s10 s1"><a href="#edituser" onClick={() => setSelectedUser({ ...{ isAdmin: false }, ...user })} className="btn-floating blue waves-light red modal-trigger"> <i className="material-icons"> edit </i></a></div>
                            <div className="col s1"><a href="#deluser" onClick={() => setSelectedUser({ ...{ isAdmin: false }, ...user })} className="btn-floating waves-effect waves-light red modal-trigger"> <i className="material-icons"> delete_forever </i></a></div>
                        </div>
                    }
                </li>
            )}
            {mode.read_only ||
                < li className="collection-item avatar">
                    {/*
                <i className="material-icons circle red" > account_box </i>
                < span className="title" > {admin.username} </span>
                <a className="waves-effect waves-light btn modal-trigger" href="#modal1">Modal</a>
                < p > First Line < br />
                Second Line
            </p>*/}
                    < a href="#edituser" className="secondary-content btn-floating green waves-light red modal-trigger" onClick={() => setSelectedUser({ isAdmin: false, doCreate: true })} > <i className="material-icons" > person_add </i></a >
                </li>
            }

        </ul>
    </>
}