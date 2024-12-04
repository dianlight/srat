import { createContext } from 'react';
import { Api } from './srat';
import { WSRouter } from './WSRouter';

export const apiContext = createContext(new Api({
    baseURL: process.env.APIURL
}));
const wsUrl = new URL(process.env.APIURL || "")
wsUrl.protocol = "ws"
wsUrl.pathname = "/ws"

export const wsContext = createContext(new WSRouter(wsUrl.href));
export const AuthContext = createContext(null);

console.log("API URL", process.env.APIURL)
console.log("WS URL", wsUrl.href)
