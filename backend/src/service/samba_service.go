package service

import (
	"log"
	"os"
	"os/exec"

	"github.com/dianlight/srat/dbutil"
	"github.com/dianlight/srat/dto"
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
	exported_share_repo repository.ExportedShareRepositoryInterface
}

func NewSambaService(apictx *dto.ContextState, dirtyservice DirtyDataServiceInterface, exported_share_repo repository.ExportedShareRepositoryInterface) SambaServiceInterface {
	p := &SambaService{}
	p.apictx = apictx
	p.dirtyservice = dirtyservice
	p.exported_share_repo = exported_share_repo
	dirtyservice.AddRestartCallback(p.WriteAndRestartSambaConfig)
	return p
}

func (self *SambaService) CreateConfigStream() (data *[]byte, err error) {
	config, err := dbutil.JSONFromDatabase(self.exported_share_repo)
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
