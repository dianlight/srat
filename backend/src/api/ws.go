package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/dianlight/srat/dto"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{} // use default options

var activeContexts = make(map[string]context.CancelFunc)

// WSChannel godoc
//
//	@Summary		WSChannel
//	@Description	Open the WSChannel
//	@Tags			system
//	@Produce		json
//	@Success		200 {object}	dto.DataDirtyTracker
//	@Failure		405	{object}	dto.ResponseError
//	@Router			/ws [get]
func WSChannelHandler(w http.ResponseWriter, rq *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	c, err := upgrader.Upgrade(w, rq, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	outchan := make(chan *dto.WebSocketMessageEnvelope, 10)

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
		var message dto.WebSocketMessageEnvelope
		err := c.ReadJSON(&message)
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("ws: %s %s", message.Event, message)
		// Dispatcher

		if message.Action == dto.ActionSubscribe {
			var ctx context.Context
			ctx, activeContexts[message.Uid] = context.WithCancel(rq.Context())
			//log.Printf("Subscribed: %s %v\n", message.Uid, activeContexts)
			switch message.Event {
			case dto.EventHeartbeat:
				go HealthCheckWsHandler(ctx, message, outchan)
			case dto.EventShare:
				go SharesWsHandler(ctx, message, outchan)
			case dto.EventVolumes:
				go VolumesWsHandler(ctx, message, outchan)
			case dto.EventUpdate:
				go UpdateWsHandler(ctx, message, outchan)
			case dto.EventDirty:
				go DirtyWsHandler(ctx, message, outchan)
			default:
				log.Printf("Unknown event: %s", message.Event)
			}
		} else if message.Action == dto.ActionUnsubscribe {
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
//	@Success		200	{object}	[]dto.EventType
//	@Failure		500	{object}	string
//	@Router			/events [get]
func WSChannelEventsList(w http.ResponseWriter, rq *http.Request) {
	HttpJSONReponse(w, dto.EventTypes, nil)
}
