package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/data"
	tempiogo "github.com/dianlight/srat/tempio"
	"github.com/icza/gog"
	"github.com/shirou/gopsutil/v4/process"
)

func createConfigStream() (*[]byte, error) {
	config_2 := config.ConfigToMap(data.Config)
	//log.Printf("New Config:\n\t%s", config_2)
	data, err := tempiogo.RenderTemplateBuffer(config_2, templateData)
	return &data, err
}

func GetSambaProcess() (*process.Process, error) {
	var allProcess, err = process.Processes()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	for _, p := range allProcess {
		var name, err = p.Name()
		if err != nil {
			continue
		}
		if name == "smbd" {
			return p, nil
		}
	}
	return nil, nil
}

// ApplySamba godoc
//
//	@Summary		Write the samba config and send signal ro restart
//	@Description	Write the samba config and send signal ro restart
//	@Tags			samba
//	@Accept			json
//	@Produce		json
//	@Success		204
//	@Failure		400	{object}	ResponseError
//	@Failure		500	{object}	ResponseError
//	@Router			/samba/apply [put]
func applySamba(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "plain/text")

	stream, err := createConfigStream()
	if err != nil {
		log.Fatal(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	if *smbConfigFile == "" {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("No file to write"))
	} else {
		err := os.WriteFile(*smbConfigFile, *stream, 0644)
		if err != nil {
			log.Fatal(err)
		}
		// TODO: Send signal to restart samba
		w.WriteHeader(http.StatusNoContent)
	}
}

// GetSambaConfig godoc
//
//	@Summary		Get the generated samba config
//	@Description	Get the generated samba config
//	@Tags			samba
//	@Accept			json
//	@Produce		plain/text
//	@Success		200	{object}	string
//	@Failure		400	{object}	ResponseError
//	@Failure		500	{object}	ResponseError
//	@Router			/samba [get]
func getSambaConfig(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "plain/text")

	var pid, err = GetSambaProcess()
	if err != nil {
		log.Fatal(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	if pid != nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Samba is running"))
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Samba is not running"))
	}

	stream, err := createConfigStream()
	if err != nil {
		log.Fatal(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	w.WriteHeader(http.StatusOK)
	w.Write(*stream)
}

type SambaProcessStatus struct {
	PID           int32     `json:"pid"`
	Name          string    `json:"name"`
	CreateTime    time.Time `json:"create_time"`
	CPUPercent    float64   `json:"cpu_percent"`
	MemoryPercent float32   `json:"memory_percent"`
	OpenFiles     int32     `json:"open_files"`
	Connections   int32     `json:"connections"`
	Status        []string  `json:"status"`
	IsRunning     bool      `json:"is_running"`
}

// GetSambaPrecessStatus godoc
//
//	@Summary		Get the current samba process status
//	@Description	Get the current samba process status
//	@Tags			samba
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	SambaProcessStatus
//	@Failure		400	{object}	ResponseError
//	@Failure		500	{object}	ResponseError
//	@Router			/samba [get]
func getSambaProcessStatus(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var spid, err = GetSambaProcess()
	if err != nil {
		log.Fatal(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	if spid == nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Samba process not found"))
		return
	}

	createTime, _ := spid.CreateTime()

	jsonResponse, jsonError := json.Marshal(&SambaProcessStatus{
		PID:           spid.Pid,
		Name:          gog.Must(spid.Name()),
		CreateTime:    time.Unix(createTime/1000, 0),
		CPUPercent:    gog.Must(spid.CPUPercent()),
		MemoryPercent: gog.Must(spid.MemoryPercent()),
		OpenFiles:     int32(len(gog.Must(spid.OpenFiles()))),
		Connections:   int32(len(gog.Must(spid.Connections()))),
		Status:        gog.Must(spid.Status()),
		IsRunning:     gog.Must(spid.IsRunning()),
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
