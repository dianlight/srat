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
	WriteSambaConfig(ctx context.Context) errors.E
	RestartSambaService(ctx context.Context) errors.E
	TestSambaConfig(ctx context.Context) errors.E
	WriteAndRestartSambaConfig(ctx context.Context) errors.E
}

type SambaServiceProcessStatus interface {
	GetProcessStatus(parentPid int32) *dto.ProcessStatus
}

type SambaService struct {
	ctx              context.Context
	ctxCancel        context.CancelFunc
	DockerInterface  string
	DockerNet        string
	state            *dto.ContextState
	share_service    ShareServiceInterface
	prop_repo        repository.PropertyRepositoryInterface
	samba_user_repo  repository.SambaUserRepositoryInterface
	mount_client     mount.ClientWithResponsesInterface
	cache            *cache.Cache
	dbomConv         converter.DtoToDbomConverterImpl
	hdidle_service   HDIdleServiceInterface
	eventBus         events.EventBusInterface
	status           dto.SambaProcessStatus
	internalServices []SambaServiceProcessStatus
}

type SambaServiceParams struct {
	fx.In
	Ctx               context.Context
	CtxCancel         context.CancelFunc
	State             *dto.ContextState
	Share_service     ShareServiceInterface
	Prop_repo         repository.PropertyRepositoryInterface
	Samba_user_repo   repository.SambaUserRepositoryInterface
	Mount_client      mount.ClientWithResponsesInterface `optional:"true"`
	Hdidle_service    HDIdleServiceInterface
	EventBus          events.EventBusInterface
	InternalProcesses []SambaServiceProcessStatus `group:"internal_services"`
}

type serviceConfig struct {
	Name                 string
	SoftResetServiceMask dto.DataDirtyTracker
	HardResetServiceMask dto.DataDirtyTracker
	Managed              bool
	StartCommand         []string
	SoftResetCommand     []string
	HardResetCommand     []string
	StopCommand          []string
}

var (
	serviceConfigMap = map[string]serviceConfig{
		"smbd": {
			Name:                 "smbd",
			SoftResetServiceMask: dto.DataDirtyTracker{Users: true, Settings: false, Shares: true},
			HardResetServiceMask: dto.DataDirtyTracker{Users: false, Settings: true, Shares: false},
			StartCommand:         []string{"s6-svc", "-uwU", "/run/s6-rc/servicedirs/smbd"},
			SoftResetCommand:     []string{"smbcontrol", "smbd", "reload-config"},
			HardResetCommand:     []string{"s6-svc", "-rwR", "/run/s6-rc/servicedirs/smbd"},
			StopCommand:          []string{"s6-svc", "-dwd", "/run/s6-rc/servicedirs/smbd"},
			Managed:              true,
		},
		"nmbd": {
			Name:                 "nmbd",
			SoftResetServiceMask: dto.DataDirtyTracker{Users: true, Settings: false, Shares: true},
			HardResetServiceMask: dto.DataDirtyTracker{Users: false, Settings: true, Shares: false},
			StartCommand:         []string{"s6-svc", "-uwU", "/run/s6-rc/servicedirs/nmbd"},
			SoftResetCommand:     []string{"smbcontrol", "nmbd", "reload-config"},
			HardResetCommand:     []string{"s6-svc", "-rwR", "/run/s6-rc/servicedirs/nmbd"},
			StopCommand:          []string{"s6-svc", "-dwd", "/run/s6-rc/servicedirs/nmbd"},
			Managed:              true,
		},
		"wsddn": {
			Name:                 "wsddn",
			SoftResetServiceMask: dto.DataDirtyTracker{Users: false, Settings: false, Shares: false},
			HardResetServiceMask: dto.DataDirtyTracker{Users: false, Settings: true, Shares: false},
			StartCommand:         []string{"s6-svc", "-u", "/run/s6-rc/servicedirs/wsddn"},
			SoftResetCommand:     []string{"s6-svc", "-r", "/run/s6-rc/servicedirs/wsddn"},
			HardResetCommand:     []string{"s6-svc", "-r", "/run/s6-rc/servicedirs/wsddn"},
			StopCommand:          []string{"s6-svc", "-d", "/run/s6-rc/servicedirs/wsddn"},
			Managed:              true,
		},
		"srat-server": {
			Name:                 "srat-server",
			SoftResetServiceMask: dto.DataDirtyTracker{Users: false, Settings: false, Shares: false},
			HardResetServiceMask: dto.DataDirtyTracker{Users: false, Settings: false, Shares: false},
			StartCommand:         []string{"s6-svc", "-u", "/run/s6-rc/servicedirs/srat-server"},
			SoftResetCommand:     []string{"s6-svc", "-r", "/run/s6-rc/servicedirs/srat-server"},
			HardResetCommand:     []string{"s6-svc", "-r", "/run/s6-rc/servicedirs/srat-server"},
			StopCommand:          []string{"true"},
			Managed:              false,
		},
	}

	defaultDirtyMask = dto.DataDirtyTracker{Shares: true, Users: true, Settings: true}
)

func NewSambaService(lc fx.Lifecycle, in SambaServiceParams) SambaServiceInterface {
	p := &SambaService{}
	p.ctx = in.Ctx
	p.ctxCancel = in.CtxCancel
	p.state = in.State
	p.share_service = in.Share_service
	p.prop_repo = in.Prop_repo

	p.samba_user_repo = in.Samba_user_repo
	p.mount_client = in.Mount_client

	p.cache = cache.New(1*time.Minute, 10*time.Minute)
	p.eventBus = in.EventBus

	p.dbomConv = converter.DtoToDbomConverterImpl{}
	p.hdidle_service = in.Hdidle_service

	p.status = dto.SambaProcessStatus{}
	p.internalServices = in.InternalProcesses

	var unsubscribe [1]func()
	unsubscribe[0] = p.eventBus.OnDirtyData(func(ctx context.Context, event events.DirtyDataEvent) errors.E {
		if event.Type == events.EventTypes.RESTART {
			slog.InfoContext(ctx, "SambaService received RESTART event, writing and restarting Samba configuration...")
			if err := p.writeAndRestartSambaConfig(ctx, event.DataDirtyTracker); err != nil {
				slog.ErrorContext(ctx, "Error writing and restarting Samba configuration", "error", err)
				return err
			}
		}
		return nil
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
			// stop all process with Managed=true
			for processName, processConfig := range serviceConfigMap {
				if p.status[processName] == nil {
					continue
				}
				if !processConfig.Managed {
					continue
				}
				slog.InfoContext(p.ctx, "Stopping service", "service", processName)
				cmdStop := exec.CommandContext(p.ctx, processConfig.StopCommand[0], processConfig.StopCommand[1:]...)
				outStop, err := cmdStop.CombinedOutput()
				if err != nil {
					slog.ErrorContext(p.ctx, "Error stopping service", "service", processName, "error", err, "output", string(outStop))
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
	sr, err := self.share_service.ListShares()
	if err != nil {
		return tconfig, errors.WithStack(err)
	}

	nshare := make([]dbom.ExportedShare, 0, len(sr))
	for _, share := range sr {
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

func (self *SambaService) GetSambaProcess() (*dto.SambaProcessStatus, errors.E) {
	var conv converter.ProcessToDtoImpl
	var allProcess, err = process.ProcessesWithContext(self.ctx)
	if err != nil {
		log.Fatal(err)
		return &self.status, errors.WithStack(err)
	}

	// Get current process PID for subprocess detection
	currentPid := int32(os.Getpid())

	for _, p := range allProcess {
		var name, err = p.Name()
		if err != nil {
			continue
		}
		for processName := range serviceConfigMap {
			if name == processName {
				if _, ok := self.status[processName]; !ok {
					self.status[processName] = &dto.ProcessStatus{}
				}

				if pp, err := p.Parent(); err == nil {
					if ppName, err := pp.Name(); err == nil && ppName == processName {
						continue
					}
				}
				processStatus, err := conv.ProcessToProcessStatus(p)
				if err != nil {
					tlog.TraceContext(self.ctx, "Error converting process to DTO", "process", processName, "pid", p.Pid, "error", err)
					continue
				}

				// If this is the current process (srat-server), find all virtual subprocesses
				if processStatus.Pid == currentPid {
					processStatus.Children = self.findChildProcesses(currentPid)
				}

				self.status[processName] = processStatus

			}
		}
	}

	return &self.status, nil
}

// findChildProcesses collects virtual subprocesses from internal services (like hdidle)
// that run as goroutines within the current process. These are not OS-level processes
// but internal monitoring threads represented with negative PIDs.
func (self *SambaService) findChildProcesses(parentPid int32) []*dto.ProcessStatus {
	var children []*dto.ProcessStatus

	for _, service := range self.internalServices {
		if procStatus := service.GetProcessStatus(parentPid); procStatus != nil && procStatus.IsRunning {
			children = append(children, procStatus)
		}
	}

	return children
}

// WriteSambaConfig writes the Samba configuration to disk using the default dirty mask.
// Exported to satisfy SambaServiceInterface and enable API callers.
func (self *SambaService) WriteSambaConfig(ctx context.Context) errors.E {
	return self.writeSambaConfig(ctx)
}

// TestSambaConfig validates the Samba configuration using testparm.
func (self *SambaService) TestSambaConfig(ctx context.Context) errors.E {
	return self.testSambaConfig(ctx)
}

// RestartSambaService restarts/reloads Samba services using the default dirty mask (all dirty).
func (self *SambaService) RestartSambaService(ctx context.Context) errors.E {
	return self.restartSambaService(ctx, defaultDirtyMask)
}

// WriteAndRestartSambaConfig writes, tests, and restarts Samba using the default dirty mask.
func (self *SambaService) WriteAndRestartSambaConfig(ctx context.Context) errors.E {
	return self.writeAndRestartSambaConfig(ctx, defaultDirtyMask)
}

func (self *SambaService) writeSambaConfig(ctx context.Context) errors.E {
	tlog.TraceContext(ctx, "Writing Samba configuration file", "file", self.state.SambaConfigFile)
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

func (self *SambaService) testSambaConfig(ctx context.Context) errors.E {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
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

func (self *SambaService) restartSambaService(ctx context.Context, dirty dto.DataDirtyTracker) errors.E {
	process, err := self.GetSambaProcess()
	if err != nil {
		return errors.WithStack(err)
	}
	// Exec smbcontrol smbd reload-config
	if process != nil {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		for processName, processConfig := range serviceConfigMap {
			tlog.TraceContext(ctx, "Restarting service", "service", processName)
			if procStatus, ok := (*process)[processName]; ok {
				if procStatus.Pid <= 0 || dirty.AndMask(processConfig.HardResetServiceMask) {
					slog.InfoContext(ctx, "Performing hard restart of service...", "service", processName)
					cmdHardRestart := exec.CommandContext(ctx, processConfig.HardResetCommand[0], processConfig.HardResetCommand[1:]...)
					outHardRestart, err := cmdHardRestart.CombinedOutput()
					if err != nil {
						return errors.Errorf("Error performing hard restart of service %s: %w \n %#v", processName, err, map[string]any{"error": err, "output": string(outHardRestart)})
					}
				} else if dirty.AndMask(processConfig.SoftResetServiceMask) {
					slog.InfoContext(ctx, "Performing soft restart of service...", "service", processName)
					cmdSoftRestart := exec.CommandContext(ctx, processConfig.SoftResetCommand[0], processConfig.SoftResetCommand[1:]...)
					outSoftRestart, err := cmdSoftRestart.CombinedOutput()
					if err != nil {
						return errors.Errorf("Error performing soft restart of service %s: %w \n %#v", processName, err, map[string]any{"error": err, "output": string(outSoftRestart)})
					}
				} else {
					slog.InfoContext(ctx, "No restart needed for service.", "service", processName)
				}
			} else {
				slog.WarnContext(ctx, "Samba process not found, perform start command if exists.", "process", processName)
				if len(processConfig.StartCommand) > 0 && self.CommandExists(processConfig.StartCommand) {
					slog.InfoContext(ctx, "Starting service...", "service", processName)
					cmdStart := exec.CommandContext(ctx, processConfig.StartCommand[0], processConfig.StartCommand[1:]...)
					outStart, err := cmdStart.CombinedOutput()
					if err != nil {
						return errors.Errorf("Error starting service %s: %w \n %#v", processName, err, map[string]any{"error": err, "output": string(outStart)})
					}
				} else {
					slog.InfoContext(ctx, "No start command defined for service or command does not exist, skipping.", "service", processName)
				}
				continue
			}
		}

		self.eventBus.EmitSamba(events.SambaEvent{
			Event:            events.Event{Type: events.EventTypes.CLEAN},
			DataDirtyTracker: dto.DataDirtyTracker{},
		})
	} else {
		slog.WarnContext(ctx, "Samba processes not found, skipping reload commands.")
	}
	return nil
}

// CommandExists checks if a command is available and executable.
// For s6-* commands, it validates that the service directory path exists.
// For other commands, it checks if the command is in PATH and is executable.
func (self *SambaService) CommandExists(cmd []string) bool {
	if len(cmd) == 0 {
		return false
	}

	cmdName := cmd[0]

	// For s6-* commands, check if the last element (service directory path) exists
	if strings.HasPrefix(cmdName, "s6-") {
		if len(cmd) < 2 {
			return false
		}
		servicePath := cmd[len(cmd)-1]
		info, err := os.Stat(servicePath)
		return err == nil && info.IsDir()
	}

	// For other commands, check if executable exists in PATH
	_, err := exec.LookPath(cmdName)
	return err == nil
}

// WriteSambaConfig Test and Restart
func (self *SambaService) writeAndRestartSambaConfig(ctx context.Context, dirty dto.DataDirtyTracker) errors.E {
	err := self.writeSambaConfig(ctx)
	if err != nil {
		return errors.WithStack(err)
	}
	err = self.testSambaConfig(ctx)
	if err != nil {
		return errors.WithStack(err)
	}
	err = self.restartSambaService(ctx, dirty)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
