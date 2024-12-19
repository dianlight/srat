import logo from "./img/logo.png"
import github from "./img/github.svg"
import { SocialIcon } from 'react-social-icons'
import pkg from '../package.json'
import { useContext, useEffect, useRef, useState } from "react"
import { apiContext, GithubContext, wsContext } from "./Contexts"
import type { MainHealth } from "./srat"
import { useMediaQuery } from "react-responsive"


export function NavBar() {
    const api = useContext(apiContext);
    const ws = useContext(wsContext);
    const octokit = useContext(GithubContext);
    const [status, setStatus] = useState(false)
    const [errorInfo, setErrorInfo] = useState<string>('')
    const [isDark, setIsDark] = useState(false)
    const [isUpdate, setIsUpdate] = useState(false)

    const systemPrefersDark = useMediaQuery(
        {
            query: "(prefers-color-scheme: dark)",
        },
        undefined,
        (isSystemDark) => setIsDark(isSystemDark)
    );


    useEffect(() => {
        document.documentElement.setAttribute('theme', isDark ? 'dark' : 'light')

        octokit.rest.repos.getLatestRelease({
            owner: pkg.repository.owner,
            repo: pkg.repository.name
        }).then((res) => {
            const latest = res.data.tag_name;
            const current = pkg.version;
            console.log("Latest version", latest, "Current version", current)
            if (latest !== current) {
                setIsUpdate(true)
            }
        }).catch(err => {
            console.error("Error checking for updates", err)
        })
        /*
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
        */
    }, [isDark])


    function onLoadHandler() {
        ws.subscribe<MainHealth>('heartbeat', (data) => {
            //console.log("Got heartbeat", data)
            setStatus(data.alive || false);
        })
        ws.onError((event) => {
            //  console.error("WS error2", event.type, JSON.stringify(event))
            setErrorInfo(JSON.stringify(event));
            setStatus(false);
        })
    }
    /*
        useEffect(() => {
            ws.subscribe<MainHealth>('heartbeat', (data) => {
                //console.log("Got heartbeat", data)
                setStatus(data.alive || false);
            })
    
            ws.onError((event) => {
                //  console.error("WS error2", event.type, JSON.stringify(event))
                setErrorInfo(JSON.stringify(event));
                setStatus(false);
            })
            // const instance = Tabs.init(tabRef.current, {
            //     swipeable: true,
            // });
    
        }, []);
    */

    return <header onLoad={onLoadHandler}>
        <div className="navbar-fixed">
            <nav className="primary nav-extended">
                <div className="nav-wrapper">
                    {/*
                    <a className="sidenav-trigger" href="#" data-target="nav-mobile"><i className="material-icons">menu</i></a>
                    */}
                    <ul>
                        <li><img id="logo-container" className="brand-logo" alt="SRAT -- Samba Rest Adminitration Tool" src={logo} /></li>
                    </ul>
                    <ul className="right">
                        <li className={isUpdate ? "pulse" : "hide"}>
                            <a id="do_update" href="#"><i className="material-icons tooltipped" data-position="bottom" data-tooltip={errorInfo}>system_update_alt</i></a>
                        </li>
                        <li>
                            <i className="material-icons tooltipped" data-position="bottom" data-tooltip={errorInfo}>{status ? "mood" : "error"}</i>
                        </li>
                        <li>
                            <a onClick={() => { setIsDark(!isDark) }} id="theme-switch" href="#"><i className="material-icons">{isDark ? "light_mode" : "dark_mode"}</i></a>
                        </li>
                        <li>
                            <a className="_ext-link" href={pkg.repository.url} target="_blank" title="Check out our GitHub"><img src={github} style={{ height: "20px" }} /></a>
                        </li>
                    </ul>
                </div>
                {/* TABS */}
                <div className="nav-content">
                    <ul className="tabs tabs-transparent">
                        <li className="tab"><a className="active" href="#shares">Shares</a></li>
                        <li className="tab"><a href="#volumes">Volumes</a></li>
                        <li className="tab"><a href="#users">Users</a></li>
                        <li className="tab"><a href="#settings">Settings</a></li>
                        <li className="tab"><a href="#smbconf">smb.conf (ro)</a></li>
                    </ul>
                </div>
            </nav>
        </div>
    </header>
    {/*

    <div className="nav-wrapper container"><img id="logo-container" className="brand-logo" alt="SRAT -- Samba Rest
        Adminitration Tool" src={logo} />
        <ul className="right _hide-on-med-and-down">
            <li><SocialIcon network="github" url={pkg.repository.url} /></li>
            <li>
                <i className="material-icons tooltipped" data-position="bottom" data-tooltip={errorInfo}>{status ? "mood" : "error"}</i>
            </li>
        </ul>
    </div >
    */}
}