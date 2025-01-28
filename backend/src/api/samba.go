package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/dianlight/srat/dbutil"
	"github.com/dianlight/srat/dto"
	tempiogo "github.com/dianlight/srat/tempio"
	"github.com/shirou/gopsutil/v4/process"
	"github.com/ztrue/tracerr"
)

func createConfigStream(ctx context.Context) (data *[]byte, err error) {

	//var config config.Config
	// Settings
	/*
		var properties dbom.Properties
		err := properties.Load()
		if err != nil {
			return nil, err
		}
		var settings dto.Settings
		err = mapper.Map(context.Background(), &settings, properties)
		if err != nil {
			return nil, tracerr.Wrap(err)
		}
		err = mapper.Map(context.Background(), &config, settings)
		if err != nil {
			return nil, tracerr.Wrap(err)
		}
		//err = mapper.Map(context.Background(), &config.Options, settings)
		//if err != nil {
		//	return nil, tracerr.Wrap(err)
		//}
		// Users
		var sambaUsers dbom.SambaUsers
		err = sambaUsers.Load()
		if err != nil {
			return nil, tracerr.Wrap(err)
		}
		var users []dto.User
		err = mapper.Map(context.Background(), &users, sambaUsers)
		if err != nil {
			return nil, tracerr.Wrap(err)
		}
		normalUsers := funk.Filter(users, func(u dto.User) bool { return !u.IsAdmin }).([]dto.User)
		err = mapper.Map(context.Background(), &config.OtherUsers, normalUsers)
		if err != nil {
			return nil, tracerr.Wrap(err)
		}
		// Admin user
		admin, ok := funk.Find(users, func(u dto.User) bool { return u.IsAdmin }).(dto.User)
		if !ok {
			return nil, errors.New("No admin user found")
		}
		config.Username = admin.Username
		config.Password = admin.Password

		// Shares
		var sambaShares dbom.ExportedShares
		err = sambaShares.Load()
		if err != nil {
			return nil, tracerr.Wrap(err)
		}
		var shares []dto.SharedResource
		err = mapper.Map(context.Background(), &shares, sambaShares) //.FromArray(&sambaShares, "Name")
		if err != nil {
			return nil, tracerr.Wrap(err)
		}
		err = mapper.Map(context.Background(), &config.Shares, shares)
		if err != nil {
			return nil, tracerr.Wrap(err)
		}
		// Special setting parameters to remove after upgrade
	*/
	config, err := dbutil.JSONFromDatabase()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	// End
	//ctsx := ctx.Value("context_state").(*dto.Status)
	ctsx := StateFromContext(ctx)
	config.DockerInterface = ctsx.DockerInterface
	config.DockerNet = ctsx.DockerNet
	config_2 := config.ConfigToMap()
	datar, err := tempiogo.RenderTemplateBuffer(config_2, ctsx.Template)
	return &datar, tracerr.Wrap(err)
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
		return nil, tracerr.Wrap(err)
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
//	@Failure		400	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/samba/apply [put]
func ApplySamba(w http.ResponseWriter, r *http.Request) {

	stream, err := createConfigStream(r.Context())
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	//ctx := r.Context().Value("context_state").(*dto.Status)
	ctx := StateFromContext(r.Context())
	if ctx.SambaConfigFile == "" {
		HttpJSONReponse(w, fmt.Errorf("No samba config file provided"), nil)
	} else {
		err := os.WriteFile(ctx.SambaConfigFile, *stream, 0644)
		if err != nil {
			HttpJSONReponse(w, err, nil)
			return
		}

		// Check samba configuration with exec testparm -s
		cmd := exec.Command("testparm", "-s", ctx.SambaConfigFile)
		out, err := cmd.CombinedOutput()
		if err != nil {
			err = fmt.Errorf("Error executing testparm: %w \n %#v", err, map[string]any{"error": err, "output": string(out)})
			HttpJSONReponse(w, err, nil)
			return
		}

		process, err := GetSambaProcess()
		if err != nil {
			HttpJSONReponse(w, err, nil)
			return
		}

		// Exec smbcontrol smbd reload-config
		if process != nil {
			cmdr := exec.Command("smbcontrol", "smbd", "reload-config")
			out, err = cmdr.CombinedOutput()
			if err != nil {
				err = fmt.Errorf("Error executing testparm: %w \n %#v", err, map[string]any{"error": err, "output": string(out)})
				HttpJSONReponse(w, err, nil)
				return
			}
		}

		HttpJSONReponse(w, nil, nil)
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
//	@Failure		500	{object}	ErrorResponse
//	@Router			/samba [get]
func GetSambaConfig(w http.ResponseWriter, r *http.Request) {
	var smbConf dto.SmbConf

	stream, err := createConfigStream(r.Context())
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	smbConf.Data = string(*stream)
	HttpJSONReponse(w, smbConf, nil)
}

// GetSambaPrecessStatus godoc
//
//	@Summary		Get the current samba process status
//	@Description	Get the current samba process status
//	@Tags			samba
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	dto.SambaProcessStatus
//	@Failure		404	{object}	ErrorResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/samba/status [get]
/*
func GetSambaProcessStatus(w http.ResponseWriter, _ *http.Request) {

	var sambaP dto.SambaProcessStatus

	var spid, err = GetSambaProcess()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	if spid == nil {
		HttpJSONReponse(w, fmt.Errorf("Samba process not found"), &Options{
			Code: http.StatusNotFound,
		})
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

	HttpJSONReponse(w, sambaP, nil)
}
*/
