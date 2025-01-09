package api

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	tempiogo "github.com/dianlight/srat/tempio"
	"github.com/icza/gog"
	"github.com/shirou/gopsutil/v4/process"
)

func createConfigStream(ctx context.Context) (*[]byte, error) {

	var config config.Config
	// Settings
	var properties dbom.Properties
	err := properties.Load()
	if err != nil {
		return nil, err
	}
	var settings dto.Settings
	err = settings.FromArray(&properties)
	if err != nil {
		return nil, err
	}
	settings.ToIgnoreEmpty(&config)
	// Users
	var sambaUsers dbom.SambaUsers
	err = sambaUsers.Load()
	if err != nil {
		return nil, err
	}
	var users dto.Users
	err = users.From(&sambaUsers) // FIXME: Tha admin user?!?!
	if err != nil {
		return nil, err
	}
	users.ToIgnoreEmpty(&config.OtherUsers)
	// Shares
	var sambaShares dbom.ExportedShares
	err = sambaShares.Load()
	if err != nil {
		return nil, err
	}
	var shares dto.SharedResources
	err = shares.FromArray(&sambaShares, "Name")
	if err != nil {
		return nil, err
	}
	shares.ToIgnoreEmpty(&config.Shares)
	// End
	config.DockerInterface = *ctx.Value("docker_interface").(*string)
	config.DockerNet = *ctx.Value("docker_network").(*string)

	config_2 := config.ConfigToMap()
	templateData := ctx.Value("template_data").([]byte)
	data, err := tempiogo.RenderTemplateBuffer(config_2, templateData)
	return &data, err
}

// GetSambaProcess retrieves the Samba process (smbd) if it's running.
//
// This function searches through all running processes to find the Samba
// daemon process named "smbd".
//
// Returns:
//   - *process.Process: A pointer to the Samba process if found, or nil if not found.
//   - error: An error if one occurred during the process search, or nil if successful.
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
//	@Failure		400	{object}	dto.ResponseError
//	@Failure		500	{object}	dto.ResponseError
//	@Router			/samba/apply [put]
func ApplySamba(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	stream, err := createConfigStream(r.Context())
	if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Error creating config stream", err.Error())
		return
	}
	smbConfigFile := r.Context().Value("samba_config_file").(*string)
	if *smbConfigFile == "" {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "No file to write", nil)
	} else {
		err := os.WriteFile(*smbConfigFile, *stream, 0644)
		if err != nil {
			dto.ResponseError{}.ToResponseError(http.StatusAccepted, w, "Error writing file", err)
		}

		// Check samba configuration with exec testparm -s
		cmd := exec.Command("testparm", "-s", *smbConfigFile)
		out, err := cmd.CombinedOutput()
		if err != nil {
			dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Error executing testparm", map[string]any{"error": err, "output": string(out)})
			return
		}

		process, err := GetSambaProcess()
		if err != nil {
			dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Error getting Samba process", err)
			return
		}

		// Exec smbcontrol smbd reload-config
		if process != nil {
			cmdr := exec.Command("smbcontrol", "smbd", "reload-config")
			out, err = cmdr.CombinedOutput()
			if err != nil {
				dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Error executing smbcontrol", map[string]any{"error": err, "output": string(out)})
				return
			}
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
//	@Produce		json
//	@Success		200	{object}	dto.SmbConf
//	@Failure		500	{object}	dto.ResponseError
//	@Router			/samba [get]
func GetSambaConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var smbConf dto.SmbConf

	stream, err := createConfigStream(r.Context())
	if err != nil {
		smbConf.ToResponseError(http.StatusInternalServerError, w, "Error creating config stream", err)
		return
	}

	smbConf.Data = string(*stream)
	smbConf.ToResponse(http.StatusOK, w)
}

// GetSambaPrecessStatus godoc
//
//	@Summary		Get the current samba process status
//	@Description	Get the current samba process status
//	@Tags			samba
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	dto.SambaProcessStatus
//	@Failure		400	{object}	dto.ResponseError
//	@Failure		500	{object}	dto.ResponseError
//	@Router			/samba/status [get]
func GetSambaProcessStatus(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var sambaP dto.SambaProcessStatus

	var spid, err = GetSambaProcess()
	if err != nil {
		sambaP.ToResponseError(http.StatusInternalServerError, w, "Error getting samba process", err)
		return
	}

	if spid == nil {
		sambaP.ToResponseError(http.StatusNotFound, w, "Samba process not found", nil)
		return
	}

	createTime, _ := spid.CreateTime()

	sambaP.PID = spid.Pid
	sambaP.Name = gog.Must(spid.Name())
	sambaP.CreateTime = time.Unix(createTime/1000, 0)
	sambaP.CPUPercent = gog.Must(spid.CPUPercent())
	sambaP.MemoryPercent = gog.Must(spid.MemoryPercent())
	sambaP.OpenFiles = int32(len(gog.Must(spid.OpenFiles())))
	sambaP.Connections = int32(len(gog.Must(spid.Connections())))
	sambaP.Status = gog.Must(spid.Status())
	sambaP.IsRunning = gog.Must(spid.IsRunning())

	sambaP.ToResponse(http.StatusOK, w)
}
