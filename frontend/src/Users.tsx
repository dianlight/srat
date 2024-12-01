import { useContext, useEffect, useState } from "react";
import { apiContext, wsContext } from "./Contexts";
import type { Api, MainShare, MainShares, MainUser } from "./srat";

export function Users() {
    const api = useContext(apiContext);
    const [status, setStatus] = useState<MainUser[]>([]);
    const [admin, setAdmin] = useState<MainUser>({});
    const [errorInfo, setErrorInfo] = useState<string>('')
    const ws = useContext(wsContext);

    useEffect(() => {
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
        getUsersList();
        async function getAdminUser() {
            api.admin.userList().then((res) => {
                setAdmin(res.data || {});
                setErrorInfo('')
            }).catch(err => {
                setErrorInfo(JSON.stringify(err));
                setAdmin({});
            })
        }
        getAdminUser();

    }, []);


    return <ul className="collection" >
        < li className="collection-item avatar" key={admin.username}>
            <i className="material-icons circle red" > account_box </i>
            < span className="title" > {admin.username} </span>
            {/*< p > First Line < br />
                Second Line
            </p>*/}
            < a href="#!" className="secondary-content" > <i className="material-icons" > grade </i></a >
        </li>
        {status.map((user) =>
            < li className="collection-item avatar" key={user.username} >
                <i className="material-icons circle green" > account_box </i>
                < span className="title" > {user.username} </span>
                {/*< p > {props.fs} < br />
                    {props.path}
                </p>*/}
                < a href="#!" className="secondary-content" > <i className="material-icons" > grade </i></a >
            </li>
        )}
    </ul>
}