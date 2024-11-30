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
    WebSocket!: WebSocket

    subcribers: Map<string, WSSubscriber<any>> = new Map()
    errorSubscribers: Map<string, (data: Event) => void> = new Map()

    lastError: string = ''

    constructor(url: string) {

        const startWebsocket = () => {

            this.WebSocket = new WebSocket(url)
            this.WebSocket.addEventListener('open', () => {
                console.log("WS opened", this.WebSocket)
                for (const subscriber of this.subcribers.values()) {
                    this.send(JSON.stringify({
                        event: subscriber.event,
                        uid: subscriber.uid
                    } as WSMessage<any>))
                }
                for (const subscriber of this.errorSubscribers.values()) {
                    this.WebSocket.addEventListener('error', subscriber)
                }
            })

            this.WebSocket.onclose = () => {
                console.log("WS closed")
            }

            this.WebSocket.addEventListener('error', (event) => {
                console.error("WS error", event)
                this.lastError = JSON.stringify(event)
                setTimeout(startWebsocket, 5000)
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
        }

        startWebsocket();
    }

    send(data: any) {
        this.WebSocket.send(data)
    }

    subscribe<T>(event: 'heartbeat' | 'shares' | 'users', cb: (data: T) => void) {
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
        this.errorSubscribers.set(uuidv4(), cb)
    }

}