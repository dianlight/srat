package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dianlight/srat/data"
)

type Health struct {
	Alive    bool  `json:"alive"`
	ReadOnly bool  `json:"read_only"`
	Samba    int32 `json:"samba_pid"`
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

	sambaProcess, err := GetSambaProcess()
	var sambaPID int32
	if err == nil && sambaProcess != nil {
		sambaPID = int32(sambaProcess.Pid)
	}

	jsonResponse, jsonError := json.Marshal(&Health{
		Alive:    true,
		ReadOnly: *data.ROMode,
		Samba:    sambaPID,
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
		sambaProcess, err := GetSambaProcess()
		var sambaPID int32
		if err == nil && sambaProcess != nil {
			sambaPID = int32(sambaProcess.Pid)
		}

		var message WebSocketMessageEnvelope = WebSocketMessageEnvelope{
			Event: "heartbeat",
			Uid:   request.Uid,
			Data:  Health{Alive: true, ReadOnly: *data.ROMode, Samba: sambaPID},
		}
		c <- &message
		time.Sleep(5 * time.Second)
	}
}
