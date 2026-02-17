import { useEffect, useState } from 'react';


export class LogEntry {
    source: string;
    level: string;
    args: string[];
    time: string;

    constructor(source: string, level: string, args: any[]) {
        this.source = source;
        this.level = level;
        this.args = args.map(arg => {
            if (arg instanceof Error) return arg.message;
            return typeof arg === 'object' ? JSON.stringify(arg) : String(arg);
        });
        this.time = new Date().toLocaleTimeString();
    }

    String() {
        return `[${this.time}] [${this.source}] [${this.level}] ${this.args?.join(' ')}`;
    }
}

/**
 * GlobalEventTracker: Classe Singleton per la gestione degli eventi.
 * Può essere utilizzata anche al di fuori di React se necessario.
 */
class GlobalEventTracker {
    static instance: GlobalEventTracker;
    history: LogEntry[] = [];
    maxSize: number = 200;
    listeners: Set<Function> = new Set();
    isInitialized: boolean = false;

    constructor(maxSize: number = 200) {
        if (GlobalEventTracker.instance) return GlobalEventTracker.instance;
        this.maxSize = maxSize;
        GlobalEventTracker.instance = this;
    }

    init() {
        if (this.isInitialized) return;

        this.initConsole();
        this.initGlobalListeners();
        this.initFetchInterceptor();
        this.isInitialized = true;
    }

    initConsole() {
        ['log', 'info', 'warn', 'error'].forEach(level => {
            const method = console[level as keyof Console] as (...args: any[]) => void;
            (console as any)[level] = ((...args: any[]) => {
                this.addEntry('JS_CONSOLE', level, args);
                method.apply(console, args);
            });
        });
    }

    initGlobalListeners() {
        window.addEventListener('error', (event) => {
            let message = 'Errore sconosciuto';
            let source = 'Sconosciuta';

            if (event.target && (event.target as HTMLElement).tagName === 'IMG' || (event.target as HTMLElement).tagName === 'SCRIPT') {
                message = `Fallimento caricamento risorsa: ${(event.target as HTMLElement).tagName}`;
                source = event.target as HTMLImageElement).src;
            } else {
                message = event.error ? event.error.message : event.message;
                source = event.filename || 'runtime';
            }

            this.addEntry('BROWSER_RUNTIME', 'error', [message, `Origine: ${source}`]);
        }, true);

        window.addEventListener('unhandledrejection', (event) => {
            this.addEntry('UNHANDLED_PROMISE', 'error', [event.reason || 'Nessuna ragione fornita']);
        });
    }

    initFetchInterceptor() {
        const originalFetch = window.fetch;
        const wrappedFetch = async (...args: Parameters<typeof fetch>) => {
            try {
                const response = await originalFetch(...args);
                if (!response.ok) {
                    this.addEntry('NETWORK_FETCH', 'error', [`HTTP ${response.status} su ${args[0]}`]);
                }
                return response;
            } catch (err) {
                const errorMessage = err instanceof Error ? err.message : String(err);
                this.addEntry('NETWORK_FETCH', 'error', [`Fallimento connessione: ${errorMessage}`]);
                throw err;
            }
        };
        window.fetch = Object.assign(wrappedFetch, originalFetch) as typeof fetch;
    }

    addEntry(source: string, level: string, args: any[]) {
        const entry = new LogEntry(source,level,args)
        this.history.push(entry);
        if (this.history.length > this.maxSize) this.history.shift();
        this.notify();
    }

    clearHistory() {
        this.history = [];
        this.notify();
    }

    notify() {
        this.listeners.forEach(l => l([...this.history]));
    }

    subscribe(l: (history: LogEntry[]) => void) {
        this.listeners.add(l);
        l([...this.history]);
        return () => {
            this.listeners.delete(l);
        };
    }
}

// Istanza esportata per uso programmatico
export const tracker = new GlobalEventTracker();

/**
 * Hook personalizzato per accedere ai log in qualsiasi componente
 */
export function useSystemLogs() {
    const [logs, setLogs] = useState<LogEntry[]>([]);

    useEffect(() => {
        tracker.init(); // Assicura l'inizializzazione al primo utilizzo
        return tracker.subscribe(setLogs);
    }, []);

    return {
        logs,
        clearLogs: () => tracker.clearHistory()
    };
}

/**
 * Componente GlobalEventMonitor
 * Inseriscilo nel root della tua app. Non renderizza nulla (null).
 */
export const GlobalEventMonitor = () => {
    useEffect(() => {
        tracker.init();
    }, []);

    return null;
};

export default GlobalEventMonitor;