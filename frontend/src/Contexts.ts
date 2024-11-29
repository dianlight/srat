import { createContext } from 'react';
import { Api } from './srat';
import { WSRouter } from './WSRouter';

export const apiContext = createContext(new Api({
    baseURL: 'http://localhost:8080',
}));
export const wsContext = createContext(new WSRouter(new WebSocket('ws://localhost:8080/ws')))
export const AuthContext = createContext(null);