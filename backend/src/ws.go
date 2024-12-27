package main

import (
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

type WebSocketMessageEnvelope struct {
	Event EventType `json:"event"`
	Uid   string    `json:"uid"`
	Data  any       `json:"data"`
}

var upgrader = websocket.Upgrader{} // use default options

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

	for {
		var message WebSocketMessageEnvelope
		err := c.ReadJSON(&message)
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s %s", message.Event, message)
		// Dispatcher

		switch message.Event {
		case EventHeartbeat:
			go HealthCheckWsHandler(message, outchan)
		case EventShare:
			go SharesWsHandler(message, outchan)
		case EventVolumes:
			go VolumesWsHandler(message, outchan)
		case EventUpdate:
			go UpdateWsHandler(message, outchan)
		case EventDirty:
			go DirtyWsHandler(message, outchan)
		default:
			log.Printf("Unknown event: %s", message.Event)
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
