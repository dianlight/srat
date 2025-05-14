package service

import (
	"log"
	"log/slog"
	"os"
	"os/exec"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/mount"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/tempio"
	"github.com/shirou/gopsutil/v4/process"
	"gitlab.com/tozd/go/errors"
)

type SambaServiceInterface interface {
	CreateConfigStream() (data *[]byte, err error)
	GetSambaProcess() (*process.Process, error)
	//StreamToFile(stream *[]byte, path string) error
	//StartSambaService(id uint) error
	//StopSambaService(id uint) error
	WriteSambaConfig() error
	RestartSambaService() error
	TestSambaConfig() error
	WriteAndRestartSambaConfig() error
}

type SambaService struct {
	DockerInterface     string
	DockerNet           string
	apictx              *dto.ContextState
	dirtyservice        DirtyDataServiceInterface
	supervisor_service  SupervisorServiceInterface
	exported_share_repo repository.ExportedShareRepositoryInterface
	prop_repo           repository.PropertyRepositoryInterface
	samba_user_repo     repository.SambaUserRepositoryInterface
	mount_client        mount.ClientWithResponsesInterface
}

func NewSambaService(apictx *dto.ContextState,
	dirtyservice DirtyDataServiceInterface,
	exported_share_repo repository.ExportedShareRepositoryInterface,
	prop_repo repository.PropertyRepositoryInterface,
	samba_user_repo repository.SambaUserRepositoryInterface,
	mount_client mount.ClientWithResponsesInterface,
	su SupervisorServiceInterface,
) SambaServiceInterface {
	p := &SambaService{}
	p.apictx = apictx
	p.dirtyservice = dirtyservice
	p.exported_share_repo = exported_share_repo
	p.prop_repo = prop_repo
	p.samba_user_repo = samba_user_repo
	p.mount_client = mount_client
	p.supervisor_service = su
	dirtyservice.AddRestartCallback(p.WriteAndRestartSambaConfig)
	return p
}

func (self *SambaService) CreateConfigStream() (data *[]byte, err error) {
	config, err := self._JSONFromDatabase()
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

func (self *SambaService) _JSONFromDatabase() (tconfig config.Config, err error) {
	var conv converter.ConfigToDbomConverterImpl

	properties, err := self.prop_repo.All(true)
	if err != nil {
		return tconfig, errors.WithStack(err)
	}
	users, err := self.samba_user_repo.All()
	if err != nil {
		return tconfig, errors.WithStack(err)
	}
	shares, err := self.exported_share_repo.All()
	if err != nil {
		return tconfig, errors.WithStack(err)
	}

	tconfig = config.Config{}
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
func (self *SambaService) GetSambaProcess() (*process.Process, error) {
	var allProcess, err = process.Processes()
	if err != nil {
		log.Fatal(err)
		return nil, errors.WithStack(err)
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

func (self *SambaService) WriteSambaConfig() error {
	stream, err := self.CreateConfigStream()
	if err != nil {
		return errors.WithStack(err)
	}

	err = os.WriteFile(self.apictx.SambaConfigFile, *stream, 0644)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (self *SambaService) TestSambaConfig() error {

	// Check samba configuration with exec testparm -s
	cmd := exec.Command("testparm", "-s", self.apictx.SambaConfigFile)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Errorf("Error executing testparm: %w \n %#v", err, map[string]any{"error": err, "output": string(out)})
	}
	return nil
}

func (self *SambaService) RestartSambaService() error {
	process, err := self.GetSambaProcess()
	if err != nil {
		return errors.WithStack(err)
	}

	// Exec smbcontrol smbd reload-config
	if process != nil {
		cmdr := exec.Command("smbcontrol", "smbd", "reload-config")
		out, err := cmdr.CombinedOutput()
		if err != nil {
			return errors.Errorf("Error executing smbcontrol: %w \n %#v", err, map[string]any{"error": err, "output": string(out)})
		}
	}
	// remount network share on ha_core
	shares, err := self.exported_share_repo.All()
	if err != nil {
		return errors.WithStack(err)
	}

	for _, share := range *shares {
		switch share.Usage {
		case "media", "share", "backup":
			err = self.supervisor_service.NetworkMountShare(share)
			if err != nil {
				slog.Error("Mounting error", "err", err)
			}
		}
	}

	return nil
}

// WriteSambaConfig Test and Restart
func (self *SambaService) WriteAndRestartSambaConfig() error {
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
