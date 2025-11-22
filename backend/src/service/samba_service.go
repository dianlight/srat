package service

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/homeassistant/mount"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/tempio"
	"github.com/dianlight/tlog"
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
	WriteSambaConfig() errors.E
	RestartSambaService() errors.E
	TestSambaConfig() errors.E
	WriteAndRestartSambaConfig() errors.E
}

type SambaService struct {
	DockerInterface string
	DockerNet       string
	state           *dto.ContextState
	//	supervisor_service SupervisorServiceInterface
	share_service ShareServiceInterface
	share_repo    repository.ExportedShareRepositoryInterface
	//	ha_ws_service      HaWsServiceInterface
	prop_repo       repository.PropertyRepositoryInterface
	samba_user_repo repository.SambaUserRepositoryInterface
	mount_client    mount.ClientWithResponsesInterface
	cache           *cache.Cache
	dbomConv        converter.DtoToDbomConverterImpl
	hdidle_service  HDIdleServiceInterface
	eventBus        events.EventBusInterface
}

type SambaServiceParams struct {
	fx.In

	State               *dto.ContextState
	Share_service       ShareServiceInterface
	Prop_repo           repository.PropertyRepositoryInterface
	Exported_share_repo repository.ExportedShareRepositoryInterface
	Samba_user_repo     repository.SambaUserRepositoryInterface
	//	HA_ws_service       HaWsServiceInterface
	Mount_client mount.ClientWithResponsesInterface `optional:"true"`
	//	Su                  SupervisorServiceInterface
	Hdidle_service HDIdleServiceInterface
	EventBus       events.EventBusInterface
}

func NewSambaService(lc fx.Lifecycle, in SambaServiceParams) SambaServiceInterface {
	p := &SambaService{}
	p.state = in.State
	p.share_service = in.Share_service
	p.prop_repo = in.Prop_repo
	p.share_repo = in.Exported_share_repo
	p.samba_user_repo = in.Samba_user_repo
	p.mount_client = in.Mount_client
	//	p.supervisor_service = in.Su
	p.cache = cache.New(1*time.Minute, 10*time.Minute)
	p.eventBus = in.EventBus
	//in.Dirtyservice.AddRestartCallback(p.WriteAndRestartSambaConfig)
	//	p.ha_ws_service = in.HA_ws_service
	p.dbomConv = converter.DtoToDbomConverterImpl{}
	p.hdidle_service = in.Hdidle_service

	var unsubscribe [1]func()
	unsubscribe[0] = p.eventBus.OnSamba(func(ctx context.Context, event events.SambaEvent) {
		if event.Type == events.EventTypes.RESTART {
			slog.InfoContext(ctx, "SambaService received RESTART event, writing and restarting Samba configuration...")
			if err := p.WriteAndRestartSambaConfig(); err != nil {
				slog.ErrorContext(ctx, "Error writing and restarting Samba configuration", "error", err)
				p.eventBus.EmitDirtyData(events.DirtyDataEvent{
					Event:            events.Event{Type: events.EventTypes.RESTART},
					DataDirtyTracker: event.DataDirtyTracker,
				})
			}
		}
	})
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			return nil
		},
		OnStop: func(context.Context) error {
			for _, unsub := range unsubscribe {
				if unsub != nil {
					unsub()
				}
			}
			return nil
		},
	})

	return p
}

func (self *SambaService) GetSambaStatus() (*dto.SambaStatus, errors.E) {
	if x, found := self.cache.Get("samba_status"); found {
		return x.(*dto.SambaStatus), nil
	}

	cmd := exec.Command("smbstatus", "-j")
	out, err := cmd.Output()
	if err != nil {
		outErr, _ := cmd.CombinedOutput()
		return nil, errors.Errorf("Error executing smbstatus: %w \n %#v", err, map[string]any{"error": err, "output": string(out), "errout": string(outErr), "cmd": cmd.String()})
	}

	// Validate that output is valid JSON before unmarshaling
	outStr := strings.TrimSpace(string(out))
	if outStr == "" {
		return nil, errors.New("smbstatus returned empty output")
	}
	if outStr[0] != '{' && outStr[0] != '[' {
		return nil, errors.Errorf("smbstatus returned non-JSON output: %s", outStr)
	}

	var status dto.SambaStatus
	err = json.Unmarshal(out, &status)
	if err != nil {
		return nil, errors.Errorf("failed to parse smbstatus output as JSON: %w (output: %s)", err, outStr)
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
	config.DockerInterface = self.state.DockerInterface
	config.DockerNet = self.state.DockerNet
	config_2 := config.ConfigToMap()

	// Add Samba version information to the template context
	sambaVersion, _ := osutil.GetSambaVersion()
	isSambaVersionSufficient, _ := osutil.IsSambaVersionSufficient()
	(*config_2)["samba_version"] = sambaVersion
	(*config_2)["samba_version_sufficient"] = isSambaVersionSufficient

	datar, err := tempio.RenderTemplateBuffer(config_2, self.state.Template)
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
	shares, err := self.share_repo.All()
	if err != nil {
		return tconfig, errors.WithStack(err)
	}

	sr, errS := self.dbomConv.ExportedSharesToSharedResources(shares)
	if errS != nil {
		return tconfig, errors.WithStack(errS)
	}

	nshare := make([]dbom.ExportedShare, 0, len(*sr))
	for _, share := range *sr {
		if share.Disabled != nil && *share.Disabled {
			continue
		}
		if share.Status != nil && !share.Status.IsValid {
			continue
		}
		if share.MountPointData != nil && share.MountPointData.IsInvalid {
			continue
		}
		dbs := dbom.ExportedShare{}
		err = self.dbomConv.SharedResourceToExportedShare(share, &dbs)
		if err != nil {
			return tconfig, errors.WithStack(err)
		}
		nshare = append(nshare, dbs)
	}

	tconfig = config.Config{}
	// set default values
	defaults.SetDefaults(&tconfig)
	// end
	err = conv.DbomObjectsToConfig(properties, users, nshare, &tconfig)
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

// GetSambaProcess retrieves the status of all Samba-related processes and subprocesses.
//
// This function searches through all running processes to find:
//   - smbd: Samba daemon (real OS process)
//   - nmbd: NetBIOS name server (real OS process)
//   - wsdd2: Web Services Discovery daemon (real OS process)
//   - srat-server: SRAT main process (real OS process)
//   - hdidle: HDIdle power-save monitoring (virtual subprocess)
//
// For virtual subprocesses (like HDIdle), the PID is set to the negative value
// of the parent process PID. This convention allows distinguishing between real
// OS processes and internal monitoring threads in the UI.
//
// Returns:
//   - *dto.SambaProcessStatus: Status of all processes, with PIDs set to -1 if not found
//   - errors.E: An error if one occurred during the process search
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
		Srat: dto.ProcessStatus{
			Pid: -1,
		},
		Hdidle: dto.ProcessStatus{
			Pid: -1,
		},
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
		case "srat-server":
			conv.ProcessToProcessStatus(p, &spc.Srat)
		}
	}

	// Get HDIdle monitoring status as a subprocess
	// Convention: Subprocesses use negative parent PIDs to indicate they are
	// virtual monitoring threads rather than real OS processes
	if self.hdidle_service != nil {
		hdidleStatus := self.hdidle_service.GetProcessStatus(spc.Srat.Pid)
		if hdidleStatus != nil {
			spc.Hdidle = *hdidleStatus
		}
	}

	return &spc, nil
}

func (self *SambaService) WriteSambaConfig() errors.E {
	tlog.Trace("Writing Samba configuration file", "file", self.state.SambaConfigFile)
	stream, errE := self.CreateConfigStream()
	if errE != nil {
		return errors.WithStack(errE)
	}

	// Restrict permissions on config file
	err := os.WriteFile(self.state.SambaConfigFile, *stream, 0o600)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (self *SambaService) TestSambaConfig() errors.E {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tlog.TraceContext(ctx, "Testing Samba configuration file", "file", self.state.SambaConfigFile)

	// Check samba configuration with exec testparm -s
	cmd := exec.CommandContext(ctx, "testparm", "-s", self.state.SambaConfigFile)
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
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		tlog.TraceContext(ctx, "Restarting Samba service")
		if process.Smbd.Pid != -1 {
			slog.InfoContext(ctx, "Reloading smbd configuration...")
			cmdSmbdReload := exec.CommandContext(ctx, "smbcontrol", "smbd", "reload-config")
			outSmbd, err := cmdSmbdReload.CombinedOutput()
			if err != nil {
				if strings.Contains(string(outSmbd), "Can't find pid for destination") {
					slog.WarnContext(ctx, "Samba process (smbd) not found, skipping reload command.")
				} else {
					slog.ErrorContext(ctx, "Error reloading smbd config", "error", err, "output", string(outSmbd))
				}
			}
			self.eventBus.EmitDirtyData(events.DirtyDataEvent{
				Event:            events.Event{Type: events.EventTypes.CLEAN},
				DataDirtyTracker: dto.DataDirtyTracker{},
			})
		}

		if process.Nmbd.Pid != -1 {
			slog.InfoContext(ctx, "Reloading nmbd configuration...")
			cmdNmbdReload := exec.CommandContext(ctx, "smbcontrol", "nmbd", "reload-config")
			outNmbd, err := cmdNmbdReload.CombinedOutput()
			if err != nil {
				if strings.Contains(string(outNmbd), "Can't find pid for destination") {
					slog.WarnContext(ctx, "Samba process (nmbd) not found, skipping reload command.")
				} else {
					slog.ErrorContext(ctx, "Error reloading nmbd config", "error", err, "output", string(outNmbd))
				}
			}
		}

		if process.Wsdd2.Pid != -1 {
			// Restart wsdd2 service using s6
			wsdd2ServicePath := "/run/s6-rc/servicedirs/wsdd2"
			if _, statErr := os.Stat(wsdd2ServicePath); statErr == nil {
				slog.InfoContext(ctx, "Restarting wsdd2 service...")
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				cmdWsdd2Restart := exec.CommandContext(ctx, "s6-svc", "-r", wsdd2ServicePath)
				outWsdd2, cmdErr := cmdWsdd2Restart.CombinedOutput()
				if cmdErr != nil {
					return errors.Errorf("Error restarting wsdd2 service: %w \n %#v", cmdErr, map[string]any{"error": cmdErr, "output": string(outWsdd2)})
				}
			} else if os.IsNotExist(statErr) {
				tlog.WarnContext(ctx, "wsdd2 service path not found, skipping restart.", "path", wsdd2ServicePath)
			} else {
				tlog.ErrorContext(ctx, "Error checking wsdd2 service path, skipping restart.", "path", wsdd2ServicePath, "error", statErr)
			}
		}
	} else {
		slog.Warn("Samba process (smbd) not found, skipping reload commands.")
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
