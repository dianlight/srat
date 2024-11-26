import { createContext } from 'react';
import { Api } from './srat';

export const apiContext = createContext(new Api({
    baseURL: 'http://localhost:8080',
}));
export const AuthContext = createContext(null);