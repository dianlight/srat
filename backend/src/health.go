package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Health struct {
	Alive bool `json:"alive"`
}

// HealthCheckHandler godoc
//
//	@Summary		HealthCheck
//	@Description	HealthCheck
//	@Tags			system
//	@Produce		json
//	@Success		200 {object}	Health
//	@Failure		405	{object}	ResponseError
//	@Router			/health [get]
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// A very simple health check.
	w.Header().Set("Content-Type", "application/json")

	// In the future we could report back on the status of our DB, or our cache
	// (e.g. Redis) by performing a simple PING, and include them in the response.
	jsonResponse, jsonError := json.Marshal(&Health{
		Alive: true,
	})

	if jsonError != nil {
		fmt.Println("Unable to encode JSON")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(jsonError.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

func HealthCheckWsHandler(request WebSocketMessageEnvelope, c chan *WebSocketMessageEnvelope) {
	for {
		var message WebSocketMessageEnvelope = WebSocketMessageEnvelope{
			Event: "heartbeat",
			Uid:   request.Uid,
			Data:  Health{Alive: true},
		}
		c <- &message
		time.Sleep(5 * time.Second)
	}
}
