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
	"go.uber.org/fx"
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

type SambaServiceParams struct {
	fx.In

	Apictx              *dto.ContextState
	Dirtyservice        DirtyDataServiceInterface
	Exported_share_repo repository.ExportedShareRepositoryInterface
	Prop_repo           repository.PropertyRepositoryInterface
	Samba_user_repo     repository.SambaUserRepositoryInterface
	Mount_client        mount.ClientWithResponsesInterface `optional:"true"`
	Su                  SupervisorServiceInterface
}

func NewSambaService(in SambaServiceParams) SambaServiceInterface {
	p := &SambaService{}
	p.apictx = in.Apictx
	p.dirtyservice = in.Dirtyservice
	p.exported_share_repo = in.Exported_share_repo
	p.prop_repo = in.Prop_repo
	p.samba_user_repo = in.Samba_user_repo
	p.mount_client = in.Mount_client
	p.supervisor_service = in.Su
	in.Dirtyservice.AddRestartCallback(p.WriteAndRestartSambaConfig)
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
		// remount network share on ha_core
		shares, err := self.exported_share_repo.All()
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
