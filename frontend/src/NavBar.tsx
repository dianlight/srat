import logo from "./img/logo.png"
import { SocialIcon } from 'react-social-icons'
import pkg from '../package.json'
import { useContext, useEffect, useState } from "react"
import { apiContext } from "./Contexts"


export function NavBar() {
    const api = useContext(apiContext);
    const [status, setStatus] = useState(false)
    const [errorInfo, setErrorInfo] = useState<string>('')

    useEffect(() => {
        async function getOnlineStatus() {
            api.health.healthList().then((res) => {
                setStatus(res.data.alive || false);
                setErrorInfo('')
                setTimeout(getOnlineStatus, 5000);
            }).catch(err => {
                setErrorInfo(JSON.stringify(err));
                setStatus(false);
                setTimeout(getOnlineStatus, 5000);
            })
        }

        setTimeout(getOnlineStatus, 1000);
    })


    return <div className="nav-wrapper container"><img id="logo-container" className="brand-logo" alt="SRAT -- Samba Rest
        Adminitration Tool" src={logo} />
        <ul className="right hide-on-med-and-down">
            <li><SocialIcon network="github" url={pkg.repository.url} /></li>
            <li>
                <i className="material-icons tooltipped" data-position="bottom" data-tooltip={errorInfo}>{status ? "mood" : "error"}</i>
            </li>
        </ul>
        <ul id="nav-mobile" className="sidenav">
            <li><a href="#">Navbar Link2</a></li>
        </ul>
        <a href="#" data-target="nav-mobile" className="sidenav-trigger"><i className="material-icons">menu</i></a>
    </div >
}