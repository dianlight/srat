import { createContext } from 'react';
import { Api, type MainHealth } from './srat';
import { WSRouter } from './WSRouter';

let APIURL = process.env.APIURL;
if (process.env.APIURL === "dynamic") {
    APIURL = window.location.href.substring(0, window.location.href.lastIndexOf('/static/') + 1);
    console.info(`Dynamic APIURL provided, using generated: ${APIURL}`)

}

export const apiContext = createContext(new Api({
    baseURL: APIURL
}));
const wsUrl = new URL(APIURL || "")
wsUrl.protocol = window.location.protocol === 'https:' ? "wss:" : "ws:"
wsUrl.pathname += "ws"

export const wsContext = createContext(new WSRouter(wsUrl.href));
//export const AuthContext = createContext(null);

console.log("API URL", APIURL)
console.log("WS URL", wsUrl.href)

export const ModeContext = createContext<MainHealth>({});
