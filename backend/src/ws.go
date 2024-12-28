package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type EventType string

const (
	EventUpdate    EventType = "update"
	EventHeartbeat EventType = "heartbeat"
	EventShare     EventType = "share"
	EventVolumes   EventType = "volumes"
	EventDirty     EventType = "dirty"
)

var EventTypes = []string{
	string(EventUpdate),
	string(EventHeartbeat),
	string(EventShare),
	string(EventVolumes),
}

type WebSocketMessageEnvelopeAction string

const (
	ActionSubscribe   WebSocketMessageEnvelopeAction = "subscribe"
	ActionUnsubscribe WebSocketMessageEnvelopeAction = "unsubscribe"
	ActionError       WebSocketMessageEnvelopeAction = "error"
)

type WebSocketMessageEnvelope struct {
	Event  EventType                      `json:"event"`
	Uid    string                         `json:"uid"`
	Data   any                            `json:"data"`
	Action WebSocketMessageEnvelopeAction `json:"action,omitempty"`
}

var upgrader = websocket.Upgrader{} // use default options

var activeContexts = make(map[string]context.CancelFunc)

// WSChannel godoc
//
//	@Summary		WSChannel
//	@Description	Open the WSChannel
//	@Tags			system
//	@Produce		json
//	@Success		200 {object}    config.ConfigSectionDirtySate
//	@Failure		405	{object}	ResponseError
//	@Router			/ws [get]
func WSChannelHandler(w http.ResponseWriter, rq *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	c, err := upgrader.Upgrade(w, rq, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	outchan := make(chan *WebSocketMessageEnvelope, 10)

	// Output channel for sending messages to the client
	go func() {
		for {
			outmessage := <-outchan
			jsonResponse, jsonError := json.Marshal(&outmessage)

			if jsonError != nil {
				log.Printf("Unable to encode JSON")
				continue
			}
			//if outmessage.Event != EventHeartbeat {
			//	log.Printf("send: %s %s", outmessage.Event, string(jsonResponse))
			//}
			c.WriteMessage(websocket.TextMessage, []byte(jsonResponse))
		}
	}()

	// Input channel for receiving messages from the client
	for {
		var message WebSocketMessageEnvelope
		err := c.ReadJSON(&message)
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("ws: %s %s", message.Event, message)
		// Dispatcher

		if message.Action == ActionSubscribe {
			ctx, activeContexts[message.Uid] = context.WithCancel(context.Background())
			//log.Printf("Subscribed: %s %v\n", message.Uid, activeContexts)
			switch message.Event {
			case EventHeartbeat:
				go HealthCheckWsHandler(ctx, message, outchan)
			case EventShare:
				go SharesWsHandler(ctx, message, outchan)
			case EventVolumes:
				go VolumesWsHandler(ctx, message, outchan)
			case EventUpdate:
				go UpdateWsHandler(ctx, message, outchan)
			case EventDirty:
				go DirtyWsHandler(ctx, message, outchan)
			default:
				log.Printf("Unknown event: %s", message.Event)
			}
		} else if message.Action == ActionUnsubscribe {
			activeContexts[message.Uid]()
			delete(activeContexts, message.Uid)
		} else {
			log.Printf("Unknown action: %s", message.Action)
		}
	}
}

// WSChannelEventsList godoc
//
//	@Summary		WSChannelEventsList
//	@Description	Return a list of available WSChannel events
//	@Tags			system
//	@Produce		json
//	@Success		200 {object}	[]EventType
//	@Failure		500	{object}	string
//	@Router			/events [get]
func WSChannelEventsList(w http.ResponseWriter, rq *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	jsonResponse, jsonError := json.Marshal(EventTypes)
	if jsonError != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(jsonError.Error()))
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}
}
