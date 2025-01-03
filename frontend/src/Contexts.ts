import { createContext } from 'react';
import { Api, type DtoDataDirtyTracker, type DtoHealthPing } from './srat';
import { WSRouter } from './WSRouter';

let APIURL = process.env.APIURL;
if (process.env.APIURL === "dynamic") {
    APIURL = window.location.href.substring(0, window.location.href.lastIndexOf('/static/') + 1);
    console.info(`Dynamic APIURL provided, using generated: ${APIURL}`)

}

export const apiContext = new Api({
    baseURL: APIURL
});
const wsUrl = new URL(APIURL || "")
wsUrl.protocol = window.location.protocol === 'https:' ? "wss:" : "ws:"
wsUrl.pathname += "ws"

export const wsContext = new WSRouter(wsUrl.href);
//export const AuthContext = createContext(null);

console.log("API URL", APIURL)
console.log("WS URL", wsUrl.href)

export const ModeContext = createContext<DtoHealthPing>({});

// Dirty data  state context
export type DirtyData = {
    shares: boolean,
    volumes: boolean,
    users: boolean,
    configs: boolean,
    //[key: string]: boolean
}
export const DirtyDataContext = createContext<DtoDataDirtyTracker>({});

