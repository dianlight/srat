package service

import (
	"log"
	"os"
	"os/exec"

	"github.com/dianlight/srat/dbutil"
	"github.com/dianlight/srat/tempio"
	"github.com/shirou/gopsutil/v4/process"
	"github.com/ztrue/tracerr"
)

type SambaServiceInterface interface {
	CreateConfigStream(dockerInterface string, dockerNet string, templateData []byte) (data *[]byte, err error)
	GetSambaProcess() (*process.Process, error)
	StreamToFile(stream *[]byte, path string) error
	//StartSambaService(id uint) error
	//StopSambaService(id uint) error
	RestartSambaService() error
	TestSambaConfig(path string) error
}

type SambaService struct {
	DockerInterface string
	DockerNet       string
}

func NewSambaService() SambaServiceInterface {
	return &SambaService{}
}

func (self *SambaService) CreateConfigStream(dockerInterface string, dockerNet string, templateData []byte) (data *[]byte, err error) {
	config, err := dbutil.JSONFromDatabase()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	// End
	//ctsx := ctx.Value("context_state").(*dto.Status)
	config.DockerInterface = dockerInterface
	config.DockerNet = dockerNet
	config_2 := config.ConfigToMap()
	datar, err := tempio.RenderTemplateBuffer(config_2, templateData)
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
func (self *SambaService) GetSambaProcess() (*process.Process, error) {
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

func (self *SambaService) StreamToFile(stream *[]byte, path string) error {

	err := os.WriteFile(path, *stream, 0644)
	if err != nil {
		return tracerr.Wrap(err)
	}
	return nil
}

func (self *SambaService) TestSambaConfig(path string) error {

	// Check samba configuration with exec testparm -s
	cmd := exec.Command("testparm", "-s", path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return tracerr.Errorf("Error executing testparm: %w \n %#v", err, map[string]any{"error": err, "output": string(out)})
	}
	return nil
}

func (self *SambaService) RestartSambaService() error {
	process, err := self.GetSambaProcess()
	if err != nil {
		return tracerr.Wrap(err)
	}

	// Exec smbcontrol smbd reload-config
	if process != nil {
		cmdr := exec.Command("smbcontrol", "smbd", "reload-config")
		out, err := cmdr.CombinedOutput()
		if err != nil {
			return tracerr.Errorf("Error executing testparm: %w \n %#v", err, map[string]any{"error": err, "output": string(out)})
		}
	}

	return nil
}
