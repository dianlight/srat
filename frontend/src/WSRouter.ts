import { v4 as uuidv4 } from 'uuid';

export interface WSMessage<T> {
    event: string
    data: T
    uid: string
}

export interface WSSubscriber<T> extends Omit<WSMessage<T>, 'data'> {
    dataType: T,
    cb: (data: T) => void
}

export class WSRouter {
    WebSocket: WebSocket

    subcribers: Map<string, WSSubscriber<any>> = new Map()

    lastError: string = ''

    constructor(ws: WebSocket) {
        this.WebSocket = ws
        ws.onopen = () => {
            console.log("WS opened")
        }

        ws.onclose = () => {
            console.log("WS closed")
        }

        ws.onerror = (event) => {
            console.error("WS error", event)
        }

        ws.onmessage = (event) => {
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
            } else
                console.error(`No subscriber for ${message.event} ${message.uid}`)

        }
    }

    send(data: any) {
        this.WebSocket.send(data)
    }

    subscribe<T>(event: 'heartbeat', cb: (data: T) => void) {
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
            uid: uuid
        } as WSMessage<T>))
    }

    onError(cb: (data: Event) => void) {
        this.WebSocket.addEventListener('error', cb)
    }

}