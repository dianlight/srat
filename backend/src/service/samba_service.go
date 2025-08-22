package service

import (
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/mount"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/tempio"
	"github.com/dianlight/srat/tlog"
	"github.com/lonegunmanb/go-defaults"
	cache "github.com/patrickmn/go-cache"
	"github.com/shirou/gopsutil/v4/process"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

type SambaServiceInterface interface {
	CreateConfigStream() (data *[]byte, err errors.E)
	GetSambaProcess() (*dto.SambaProcessStatus, errors.E)
	GetSambaStatus() (*dto.SambaStatus, errors.E)
	//StreamToFile(stream *[]byte, path string) error
	//StartSambaService(id uint) error
	//StopSambaService(id uint) error
	WriteSambaConfig() errors.E
	RestartSambaService() errors.E
	TestSambaConfig() errors.E
	WriteAndRestartSambaConfig() errors.E
}

type SambaService struct {
	DockerInterface    string
	DockerNet          string
	apictx             *dto.ContextState
	dirtyservice       DirtyDataServiceInterface
	supervisor_service SupervisorServiceInterface
	share_service      ShareServiceInterface
	prop_repo          repository.PropertyRepositoryInterface
	samba_user_repo    repository.SambaUserRepositoryInterface
	mount_client       mount.ClientWithResponsesInterface
	cache              *cache.Cache
}

type SambaServiceParams struct {
	fx.In

	Apictx          *dto.ContextState
	Dirtyservice    DirtyDataServiceInterface
	Share_service   ShareServiceInterface
	Prop_repo       repository.PropertyRepositoryInterface
	Samba_user_repo repository.SambaUserRepositoryInterface
	Mount_client    mount.ClientWithResponsesInterface `optional:"true"`
	Su              SupervisorServiceInterface
}

func NewSambaService(in SambaServiceParams) SambaServiceInterface {
	p := &SambaService{}
	p.apictx = in.Apictx
	p.dirtyservice = in.Dirtyservice
	p.share_service = in.Share_service
	p.prop_repo = in.Prop_repo
	p.samba_user_repo = in.Samba_user_repo
	p.mount_client = in.Mount_client
	p.supervisor_service = in.Su
	p.cache = cache.New(1*time.Minute, 10*time.Minute)
	in.Dirtyservice.AddRestartCallback(p.WriteAndRestartSambaConfig)
	return p
}

func (self *SambaService) GetSambaStatus() (*dto.SambaStatus, errors.E) {
	if x, found := self.cache.Get("samba_status"); found {
		return x.(*dto.SambaStatus), nil
	}

	cmd := exec.Command("smbstatus", "-j")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Errorf("Error executing smbstatus: %w \n %#v", err, map[string]any{"error": err, "output": string(out)})
	}

	var status dto.SambaStatus
	err = json.Unmarshal(out, &status)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	self.cache.Set("samba_status", &status, cache.DefaultExpiration)

	return &status, nil
}

func (self *SambaService) CreateConfigStream() (data *[]byte, err errors.E) {
	config, err := self.jSONFromDatabase()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	// End
	//ctsx := ctx.Value("context_state").(*dto.Status)
	config.DockerInterface = self.apictx.DockerInterface
	config.DockerNet = self.apictx.DockerNet
	config_2 := config.ConfigToMap()
	datar, err := tempio.RenderTemplateBuffer(config_2, self.apictx.Template)
	return &datar, errors.WithStack(err)
}

func (self *SambaService) jSONFromDatabase() (tconfig config.Config, err errors.E) {
	var conv converter.ConfigToDbomConverterImpl

	properties, err := self.prop_repo.All(true)
	if err != nil {
		return tconfig, errors.WithStack(err)
	}
	users, err := self.samba_user_repo.All()
	if err != nil {
		return tconfig, errors.WithStack(err)
	}
	shares, err := self.share_service.All()
	if err != nil {
		return tconfig, errors.WithStack(err)
	}

	tconfig = config.Config{}
	// set default values
	defaults.SetDefaults(&tconfig)
	// end
	err = conv.DbomObjectsToConfig(properties, users, *shares, &tconfig)
	if err != nil {
		return tconfig, errors.WithStack(err)
	}
	for _, cshare := range tconfig.Shares {
		if cshare.Usage == "media" {
			tconfig.Medialibrary.Enable = true
			break
		}
	}

	return tconfig, nil
}

// GetSambaProcess retrieves the Samba process (smbd) if it's running.
//
// This function searches through all running processes to find the Samba
// daemon process named "smbd".
//
// Returns:
//   - *process.Process: A pointer to the Samba process if found, or nil if not found.
//   - error: An error if one occurred during the process search, or nil if successful.
func (self *SambaService) GetSambaProcess() (*dto.SambaProcessStatus, errors.E) {
	spc := dto.SambaProcessStatus{
		Smbd: dto.ProcessStatus{
			Pid: -1,
		},
		Nmbd: dto.ProcessStatus{
			Pid: -1,
		},
		Wsdd2: dto.ProcessStatus{
			Pid: -1,
		},
		//Avahi: dto.ProcessStatus{
		//	Pid: -1,
		//},
	}
	var conv converter.ProcessToDtoImpl
	var allProcess, err = process.Processes()
	if err != nil {
		log.Fatal(err)
		return &spc, errors.WithStack(err)
	}
	for _, p := range allProcess {
		var name, err = p.Name()
		if err != nil {
			continue
		}
		switch name {
		case "smbd":
			conv.ProcessToProcessStatus(p, &spc.Smbd)
		case "nmbd":
			conv.ProcessToProcessStatus(p, &spc.Nmbd)
		case "wsdd2":
			conv.ProcessToProcessStatus(p, &spc.Wsdd2)
			//case "avahi-daemon":
			//	conv.ProcessToProcessStatus(p, &spc.Avahi)
		}
	}
	return &spc, nil
}

func (self *SambaService) WriteSambaConfig() errors.E {
	stream, errE := self.CreateConfigStream()
	if errE != nil {
		return errors.WithStack(errE)
	}

	// Restrict permissions on config file
	err := os.WriteFile(self.apictx.SambaConfigFile, *stream, 0o600)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (self *SambaService) TestSambaConfig() errors.E {

	// Check samba configuration with exec testparm -s
	cmd := exec.Command("testparm", "-s", self.apictx.SambaConfigFile)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Errorf("Error executing testparm: %w \n %#v", err, map[string]any{"error": err, "output": string(out)})
	}
	return nil
}

func (self *SambaService) RestartSambaService() errors.E {
	process, err := self.GetSambaProcess()
	if err != nil {
		return errors.WithStack(err)
	}

	// Exec smbcontrol smbd reload-config
	if process != nil {
		slog.Info("Reloading smbd configuration...")
		cmdSmbdReload := exec.Command("smbcontrol", "smbd", "reload-config")
		outSmbd, err := cmdSmbdReload.CombinedOutput()
		if err != nil {
			slog.Error("Error reloading smbd config", "error", err, "output", string(outSmbd))
			// Decide if this is a fatal error or if we should continue
		}

		slog.Info("Reloading nmbd configuration...")
		cmdNmbdReload := exec.Command("smbcontrol", "nmbd", "reload-config")
		outNmbd, err := cmdNmbdReload.CombinedOutput()
		if err != nil {
			slog.Error("Error reloading nmbd config", "error", err, "output", string(outNmbd))
			// Decide if this is a fatal error or if we should continue
		}

		// Remount network shares on ha_core
		// This logic might be better placed after confirming all local services are stable
		// or if it's specifically tied to smbd/nmbd reloads.
		shares, err := self.share_service.All()
		if err != nil {
			return errors.WithStack(err)
		}

		for _, share := range *shares {
			if share.Disabled != nil && *share.Disabled {
				continue
			}
			switch share.Usage {
			case "media", "share", "backup":
				err = self.supervisor_service.NetworkMountShare(share)
				if err != nil {
					slog.Error("Mounting error", "share", share, "err", err)
				}
			}
		}
	} else {
		slog.Warn("Samba process (smbd) not found, skipping reload commands.")
	}

	// Restart wsdd2 service using s6
	wsdd2ServicePath := "/run/s6-rc/servicedirs/wsdd2"
	if _, statErr := os.Stat(wsdd2ServicePath); statErr == nil {
		slog.Info("Restarting wsdd2 service...")
		cmdWsdd2Restart := exec.Command("s6-svc", "-r", wsdd2ServicePath)
		outWsdd2, cmdErr := cmdWsdd2Restart.CombinedOutput()
		if cmdErr != nil {
			return errors.Errorf("Error restarting wsdd2 service: %w \n %#v", cmdErr, map[string]any{"error": cmdErr, "output": string(outWsdd2)})
		}
	} else if os.IsNotExist(statErr) {
		tlog.Warn("wsdd2 service path not found, skipping restart.", "path", wsdd2ServicePath)
	} else {
		tlog.Error("Error checking wsdd2 service path, skipping restart.", "path", wsdd2ServicePath, "error", statErr)
	}
	return nil
}

// WriteSambaConfig Test and Restart
func (self *SambaService) WriteAndRestartSambaConfig() errors.E {
	err := self.WriteSambaConfig()
	if err != nil {
		return errors.WithStack(err)
	}
	err = self.TestSambaConfig()
	if err != nil {
		return errors.WithStack(err)
	}
	err = self.RestartSambaService()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
