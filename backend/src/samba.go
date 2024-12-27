package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
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

		// Check samba configuration with exec testparm -s
		cmd := exec.Command("testparm", "-s", *smbConfigFile)
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Error executing testparm: %v\nOutput: %s", err, out)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		// Exec smbcontrol smbd reload-config
		cmdr := exec.Command("smbcontrol", "smbd", "reload-config")
		err = cmdr.Run()
		if err != nil {
			log.Printf("Error executing smbcontrol: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

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
//	@Router			/samba/status [get]
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

// PersistConfig godoc
//
//	@Summary		Persiste the current samba config
//	@Description	Save dirty changes to the disk
//	@Tags			samba
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	config.Config
//	@Failure		400	{object}	ResponseError
//	@Failure		500	{object}	ResponseError
//	@Router			/config [put]
//	@Router			/config [patch]
func persistConfig(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	jsonResponse, jsonError := json.Marshal(data.Config)
	if jsonError != nil {
		fmt.Println("Unable to encode JSON")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(jsonError.Error()))
		return
	}

	config.SaveConfig(data.Config)
	data.DirtySectionState.Settings = false
	data.DirtySectionState.Users = false
	data.DirtySectionState.Shares = false
	data.DirtySectionState.Volumes = false

	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

// RollbackConfig godoc
//
//	@Summary		Rollback the current samba config
//	@Description    Revert to the last saved samba config
//	@Tags			samba
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	config.Config
//	@Failure		400	{object}	ResponseError
//	@Failure		500	{object}	ResponseError
//	@Router			/config [delete]
func rollbackConfig(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	config, err := config.RollbackConfig(data.Config)
	if err != nil {
		fmt.Printf("Error rolling back config: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	jsonResponse, jsonError := json.Marshal(config)
	if jsonError != nil {
		fmt.Println("Unable to encode JSON")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(jsonError.Error()))
		return
	}

	data.Config = config
	data.DirtySectionState.Settings = false
	data.DirtySectionState.Users = false
	data.DirtySectionState.Shares = false
	data.DirtySectionState.Volumes = false

	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}
