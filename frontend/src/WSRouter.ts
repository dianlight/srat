import { v4 as uuidv4 } from 'uuid';
import type { MainEventType } from './srat';
import { toast } from 'react-toastify';

/** Message structure for WebSocket communication */
export interface WSMessage<T> {
    /** Event type identifier */
    event: string
    /** Payload data */
    data: T
    /** Unique identifier for the message */
    uid: string
    /** Action to perform */
    action: 'subscribe' | 'unsubscribe' | 'error'
}

/** Subscriber information for WebSocket events */
export interface WSSubscriber<T> extends Omit<WSMessage<T>, 'data' | 'action'> {
    /** Expected data type */
    dataType: T,
    /** Callback function to handle received data */
    cb: (data: T) => void
}

/**
 * WebSocket Router class for managing WebSocket connections and subscriptions
 * Handles automatic reconnection and message routing to subscribers
 */
export class WSRouter {
    /** WebSocket instance */
    WebSocket!: WebSocket
    /** Map of subscribers indexed by UUID */
    subcribers: Map<string, WSSubscriber<any>> = new Map()
    /** Map of error handlers indexed by UUID */
    errorSubscribers: Map<string, (data: Event) => void> = new Map()
    /** Last error message */
    lastError: string = ''
    /** Pid for the riconnect Tiemout */
    timeoutpids: ReturnType<typeof setTimeout>[] = []
    /**
     * Create a new WebSocketRouter instance
     * @param url WebSocket URL
     */
    constructor(url: string) {
        const startWebsocket = () => {
            this.timeoutpids = this.timeoutpids.filter((pid) => { clearTimeout(pid); return false });
            try {
                console.log("WS connecting", url)
                this.WebSocket = new WebSocket(url)
                this.WebSocket.addEventListener('open', () => {
                    console.log("WS opened", this.WebSocket)
                    toast.dismiss(0xFFFFFFFF)
                    toast.dismiss(0xFFFFFFFE)
                    for (const subscriber of this.subcribers.values()) {
                        this.send(JSON.stringify({
                            event: subscriber.event,
                            uid: subscriber.uid,
                            action: 'subscribe',
                        } as WSMessage<any>))
                    }
                    for (const subscriber of this.errorSubscribers.values()) {
                        this.WebSocket.addEventListener('error', subscriber)
                    }
                })

                this.WebSocket.onclose = () => {
                    console.log("WS closed")
                    toast.warn(`Server Connection closed!`, { toastId: 0xFFFFFFFF });
                    this.timeoutpids = this.timeoutpids.filter((pid) => { clearTimeout(pid); return false });
                    this.timeoutpids.push(setTimeout(startWebsocket, 1000))
                }

                this.WebSocket.addEventListener('error', (event) => {
                    console.error("WS error", event)
                    toast.error(`Server Connection error! Retry in 5s`, { toastId: 0xFFFFFFFE, data: { error: event } });
                    this.lastError = JSON.stringify(event)
                    this.timeoutpids = this.timeoutpids.filter((pid) => { clearTimeout(pid); return false });
                    this.timeoutpids.push(setTimeout(startWebsocket, 5000))
                })

                this.WebSocket.onmessage = (event) => {
                    //console.log(`WS message ${event.data}`)
                    const message = JSON.parse(event.data) as WSMessage<any>
                    const subscriber = this.subcribers.get(message.uid)
                    if (subscriber) {
                        if (subscriber.dataType === typeof message.data) {
                            subscriber.cb(message.data)
                            this.lastError = ''
                        } else {
                            console.error(`Data type mismatch ${subscriber.dataType} !== ${typeof message.data}`)
                        }
                    } else {
                        console.error(`No subscriber for ${message.event} ${message.uid}`)
                    }
                }
            } catch (e) {
                console.error(e)
                setTimeout(startWebsocket, 5000)
            }
        }

        startWebsocket();
    }

    /**
     * Sends data through the WebSocket connection
     * @param data - Data to send
     */
    send(data: any) {
        try {
            if (this.WebSocket.readyState !== WebSocket.OPEN) {
                return
            }
            this.WebSocket.send(data)
        } catch (e) {
            console.error(e)
        }
    }


    /**
     * Subscribes to a specific event type
     * @param event - Event type to subscribe to
     * @param cb - Callback function to handle received data
     * @returns UUID of the subscription
     */
    subscribe<T>(event: MainEventType, cb: (data: T) => void): string {
        const type: T = {} as T;
        const uuid = uuidv4();
        this.subcribers.set(uuid, {
            event,
            uid: uuid,
            dataType: typeof type,
            cb
        });
        this.send(JSON.stringify({
            event,
            uid: uuid,
            action: 'subscribe',
        } as WSMessage<T>))
        return uuid;
    }

    /**
     * Unsubscribes from a specific subscription
     * @param uid - UUID of the subscription to remove
     */
    unsubscribe(uid: string) {
        const subscriber = this.subcribers.get(uid)
        if (subscriber) {
            this.send(JSON.stringify({
                event: subscriber.event,
                uid: uid,
                action: 'unsubscribe',
            } as WSMessage<any>))
            this.subcribers.delete(uid)
        }
    }

    /**
      * Registers an error handler for WebSocket errors
      * @param cb - Callback function to handle error events
      */
    onError(cb: (data: Event) => void) {
        this.WebSocket.addEventListener('error', cb)
        this.errorSubscribers.set(uuidv4(), cb)
    }

}