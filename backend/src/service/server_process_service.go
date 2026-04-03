package service

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/homeassistant/mount"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/srat/tempio"
	"github.com/dianlight/srat/templates"
	"github.com/dianlight/srat/unixsamba"
	"github.com/dianlight/tlog"
	"github.com/lonegunmanb/go-defaults"
	cache "github.com/patrickmn/go-cache"
	"github.com/shirou/gopsutil/v4/process"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

// Mockable functions for testing
var (
	NetInterfaceByName = net.InterfaceByName
	NetInterfaceAddrs  = func(iface *net.Interface) ([]net.Addr, error) { return iface.Addrs() }
)

// SetNetInterfaceByName allows overriding net.InterfaceByName in tests.
func SetNetInterfaceByName(fn func(name string) (*net.Interface, error)) {
	NetInterfaceByName = fn
}

// SetNetInterfaceAddrs allows overriding iface.Addrs() in tests.
func SetNetInterfaceAddrs(fn func(iface *net.Interface) ([]net.Addr, error)) {
	NetInterfaceAddrs = fn
}

// ResolveInterfaceIPv4s converts interface names to their IPv4 addresses.
// Always includes 127.0.0.1 (loopback) regardless of the input names.
// For each interface, only IPv4 addresses are returned (IPv6 is ignored).
// Logs warnings for interfaces that cannot be resolved or have no IPv4 addresses.
func ResolveInterfaceIPv4s(names []string) []string {
	var ips []string
	seen := make(map[string]any)

	// Always include loopback
	//	ips = append(ips, "127.0.0.1")
	//	seen["127.0.0.1"] = struct{}{}

	for _, name := range names {
		iface, err := NetInterfaceByName(name)
		if err != nil {
			slog.Warn("Failed to resolve network interface", "interface", name, "error", err)
			continue
		}
		addrs, err := NetInterfaceAddrs(iface)
		if err != nil {
			slog.Warn("Failed to get addresses for interface", "interface", name, "error", err)
			continue
		}
		found := false
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLinkLocalUnicast() {
				continue
			}
			// Only allow IPv4
			if ip.To4() == nil {
				continue
			}
			ipStr := ip.String()
			if _, ok := seen[ipStr]; ok {
				continue
			}
			ips = append(ips, ipStr)
			seen[ipStr] = struct{}{}
			found = true
		}
		if !found {
			slog.Warn("Interface has no suitable IPv4 addresses", "interface", name)
		}
	}
	return ips
}

type ServerServiceInterface interface {
	CreateSambaConfigStream() (data *[]byte, err errors.E)
	CreateSambaUsersMapStream() (data *[]byte, err errors.E)
	GetServerProcesses() (*dto.ServerProcessStatus, errors.E)
	GetSambaStatus() (*dto.SambaStatus, errors.E)
	WriteConfigsAndRestartProcesses(ctx context.Context) errors.E
	SetState(state *dto.ContextState)
}

type ServerProcessStatus interface {
	GetProcessStatus(parentPid int32) *dto.ProcessStatus
}

type ServerService struct {
	ctx             context.Context
	ctxCancel       context.CancelFunc
	DockerInterface string
	DockerNet       string
	state           *dto.ContextState
	share_service   ShareServiceInterface
	user_service    UserServiceInterface
	host_service    HostServiceInterface
	setting_service SettingServiceInterface
	//prop_repo        repository.PropertyRepositoryInterface
	mount_client     mount.ClientWithResponsesInterface
	cache            *cache.Cache
	dbomConv         converter.DtoToDbomConverterImpl
	hdidle_service   HDIdleServiceInterface
	eventBus         events.EventBusInterface
	commandRunner    CommandExecutionServiceInterface
	status           dto.ServerProcessStatus
	internalServices []ServerProcessStatus
}

type ServerServiceParams struct {
	fx.In
	Ctx             context.Context
	CtxCancel       context.CancelFunc
	State           *dto.ContextState
	Share_service   ShareServiceInterface
	User_service    UserServiceInterface
	Host_service    HostServiceInterface
	Setting_service SettingServiceInterface
	//Samba_user_repo   repository.SambaUserRepositoryInterface
	Mount_client      mount.ClientWithResponsesInterface `optional:"true"`
	Hdidle_service    HDIdleServiceInterface
	EventBus          events.EventBusInterface
	CommandRunner     CommandExecutionServiceInterface
	InternalProcesses []ServerProcessStatus `group:"internal_services"`
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
	sambaUsersMapFile = "/etc/samba/smbusers"

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
		"nfsd": {
			Name:                 "nfsd",
			SoftResetServiceMask: dto.DataDirtyTracker{Users: false, Settings: false, Shares: true},
			HardResetServiceMask: dto.DataDirtyTracker{Users: false, Settings: true, Shares: false},
			StartCommand:         []string{"s6-svc", "-uwu", "/run/s6-rc/servicedirs/nfsd"},
			SoftResetCommand:     []string{"exportfs", "-rav"},
			HardResetCommand:     []string{"s6-svc", "-rwr", "/run/s6-rc/servicedirs/nfsd"},
			StopCommand:          []string{"s6-svc", "-dwd", "/run/s6-rc/servicedirs/nfsd"},
			Managed:              false,
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

func NewServerProcessesService(lc fx.Lifecycle, in ServerServiceParams) ServerServiceInterface {
	p := &ServerService{}
	p.ctx = in.Ctx
	p.ctxCancel = in.CtxCancel
	p.state = in.State
	p.share_service = in.Share_service
	//p.prop_repo = in.Prop_repo
	p.user_service = in.User_service
	p.setting_service = in.Setting_service
	p.host_service = in.Host_service

	//p.samba_user_repo = in.Samba_user_repo
	p.mount_client = in.Mount_client

	p.cache = cache.New(1*time.Minute, 10*time.Minute)
	p.eventBus = in.EventBus

	p.dbomConv = converter.DtoToDbomConverterImpl{}
	p.hdidle_service = in.Hdidle_service
	p.commandRunner = in.CommandRunner

	p.status = dto.ServerProcessStatus{}
	p.internalServices = in.InternalProcesses

	var unsubscribe [1]func()
	unsubscribe[0] = p.eventBus.OnDirtyData(func(ctx context.Context, event events.DirtyDataEvent) errors.E {
		if event.Type == events.EventTypes.RESTART {
			slog.InfoContext(ctx, "ServerProcesses received RESTART event, writing and restarting Samba configuration...")
			if event.DataDirtyTracker.Settings {
				if setting, err2 := p.setting_service.Load(); err2 != nil {
					slog.ErrorContext(ctx, "Error getting HAUseNFS setting", "error", err2)
					return err2

				} else {
					nfsdConfig, ok := serviceConfigMap["nfsd"]
					if !ok {
						slog.ErrorContext(ctx, "nfsd service config not found", "service_config_map", serviceConfigMap)
						return errors.New("nfsd service config not found")
					}
					nfsdConfig.Managed = *setting.HAUseNFS
					serviceConfigMap["nfsd"] = nfsdConfig
				}
			}
			if err := p.writeConfigsAndRestartServers(ctx, event.DataDirtyTracker); err != nil {
				slog.ErrorContext(ctx, "Error writing and restarting Samba configuration", "error", err)
				return err
			}
		}
		return nil
	})
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			serviceStart := time.Now()
			tlog.TraceContext(ctx, "=== SERVICE INIT: ServerProcesses Starting ===")
			if err := p.writeSambaUsersMapConfig(ctx); err != nil {
				return err
			}
			if err := p.recoverMediaUsageSymlinks(ctx, "/media"); err != nil {
				slog.WarnContext(ctx, "Media symlink startup recovery completed with warnings", "err", err)
			}
			defer func() {
				tlog.TraceContext(ctx, "=== SERVICE INIT: ServerProcesses Complete ===", "duration", time.Since(serviceStart))
			}()
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
				outStop, err := p.runCommandWithRunner(p.ctx, "service-stop-"+processName, "Stop "+processName, processConfig.StopCommand)
				if err != nil {
					slog.ErrorContext(p.ctx, "Error stopping service", "service", processName, "error", err, "output", outStop)
				}
			}
			return nil
		},
	})

	return p
}

func (self *ServerService) commandOutputFromSnapshot(snapshot dto.CommandExecutionSnapshot) string {
	if len(snapshot.Lines) == 0 {
		return ""
	}
	builder := strings.Builder{}
	for i, line := range snapshot.Lines {
		if i > 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString(line.Line)
	}
	return builder.String()
}

func (self *ServerService) runCommandWithRunner(ctx context.Context, commandID, label string, args []string) (string, errors.E) {
	if len(args) == 0 {
		return "", errors.New("missing command")
	}

	if self.commandRunner == nil {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return string(out), errors.WithStack(err)
		}
		return string(out), nil
	}

	snapshot, err := self.commandRunner.Execute(ctx, commandID, label, args[0], args[1:]...)
	output := self.commandOutputFromSnapshot(snapshot)
	if err != nil {
		return output, errors.WithStack(err)
	}

	return output, nil
}

func (self *ServerService) GetSambaStatus() (*dto.SambaStatus, errors.E) {
	if x, found := self.cache.Get("samba_status"); found {
		return x.(*dto.SambaStatus), nil
	}

	ctx, cancel := context.WithTimeout(self.ctx, 30*time.Second)
	defer cancel()

	out, err := self.runCommandWithRunner(ctx, "samba-status", "Samba status", []string{"smbstatus", "-j"})
	if err != nil {
		return nil, errors.Errorf("Error executing smbstatus: %w \n %#v", err, map[string]any{"error": err, "output": out, "cmd": "smbstatus -j"})
	}

	// Validate that output is valid JSON before unmarshaling
	outStr := strings.TrimSpace(out)
	if outStr == "" {
		return nil, errors.New("smbstatus returned empty output")
	}
	if outStr[0] != '{' && outStr[0] != '[' {
		return nil, errors.Errorf("smbstatus returned non-JSON output: %s", outStr)
	}

	var status dto.SambaStatus
	unmarshalErr := json.Unmarshal([]byte(out), &status)
	if unmarshalErr != nil {
		return nil, errors.Errorf("failed to parse smbstatus output as JSON: %w (output: %s)", unmarshalErr, outStr)
	}

	self.cache.Set("samba_status", &status, cache.DefaultExpiration)

	return &status, nil
}

func (self *ServerService) CreateSambaConfigStream() (data *[]byte, err errors.E) {
	config, err := self.jSONFromDatabase()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	// End
	//ctsx := ctx.Value("context_state").(*dto.Status)
	config.DockerInterface = self.state.DockerInterface
	config.DockerNet = self.state.DockerNet

	// If config.Intrfaces don't contain "lo" (loopback), add it automatically to ensure local connectivity
	if !slices.Contains(config.Interfaces, "lo") {
		config.Interfaces = append([]string{"lo"}, config.Interfaces...)
	}
	// Resolve interface names to IP addresses only if I need only IPv4
	if self.state.DisableIPv6 {
		config.Interfaces = ResolveInterfaceIPv4s(config.Interfaces)
		config.DockerInterface = ResolveInterfaceIPv4s([]string{self.state.DockerInterface})[0]
	}

	config_2 := config.ConfigToMap()

	// Add Samba version information to the template context
	sambaVersion, _ := osutil.GetSambaVersion()
	isSambaVersionSufficient, _ := osutil.IsSambaVersionSufficient()
	(*config_2)["samba_version"] = sambaVersion
	(*config_2)["samba_version_sufficient"] = isSambaVersionSufficient

	datar, err := tempio.RenderTemplateBuffer(config_2, self.state.Template)
	return &datar, errors.WithStack(err)
}

func (self *ServerService) CreateSambaUsersMapStream() (data *[]byte, err errors.E) {
	type sambaUserMapping struct {
		UnixUsername   string
		SambaUsernames []string
	}

	users, listErr := self.user_service.ListUsers()
	if listErr != nil {
		return nil, errors.WithStack(listErr)
	}

	aliasesByUnixUser := make(map[string][]string)
	for _, user := range users {
		normalizedUsername := unixsamba.NormalizeUsernameForUnixSamba(user.Username)
		if normalizedUsername == "" || normalizedUsername == user.Username {
			continue
		}

		aliasesByUnixUser[normalizedUsername] = append(aliasesByUnixUser[normalizedUsername], user.Username)
	}

	sortedUnixUsers := make([]string, 0, len(aliasesByUnixUser))
	for unixUsername := range aliasesByUnixUser {
		sortedUnixUsers = append(sortedUnixUsers, unixUsername)
	}
	sort.Strings(sortedUnixUsers)

	mappings := make([]sambaUserMapping, 0, len(sortedUnixUsers))
	for _, unixUsername := range sortedUnixUsers {
		aliases := aliasesByUnixUser[unixUsername]
		sort.Strings(aliases)
		mappings = append(mappings, sambaUserMapping{
			UnixUsername:   unixUsername,
			SambaUsernames: aliases,
		})
	}

	templateData, templateErr := templates.Template_content.ReadFile("smbusers.gtpl")
	if templateErr != nil {
		return nil, errors.WithStack(templateErr)
	}

	renderData := map[string]any{
		"mappings": mappings,
	}

	rendered, renderErr := tempio.RenderTemplateBuffer(&renderData, templateData)
	if renderErr != nil {
		return nil, errors.WithStack(renderErr)
	}

	return &rendered, nil
}

func (self *ServerService) jSONFromDatabase() (tconfig config.Config, err errors.E) {
	var conv converter.ConfigToDbomConverterImpl

	settings, err := self.setting_service.Load()
	if err != nil {
		return tconfig, errors.WithStack(err)
	}

	properties := dbom.Properties{}
	err = self.dbomConv.SettingsToProperties(*settings, &properties)
	if err != nil {
		return tconfig, errors.WithStack(err)
	}

	users, errS := self.user_service.ListUsers()
	if errS != nil {
		return tconfig, errors.WithStack(errS)
	}
	smbus, errS := self.dbomConv.UsersToSambaUsers(users)
	if errS != nil {
		return tconfig, errors.WithStack(errS)
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
	err = conv.DbomObjectsToConfig(properties, smbus, nshare, &tconfig)
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

func (self *ServerService) GetServerProcesses() (*dto.ServerProcessStatus, errors.E) {
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
func (self *ServerService) findChildProcesses(parentPid int32) []*dto.ProcessStatus {
	var children []*dto.ProcessStatus

	for _, service := range self.internalServices {
		if procStatus := service.GetProcessStatus(parentPid); procStatus != nil && procStatus.IsRunning {
			children = append(children, procStatus)
		}
	}

	return children
}

// WriteConfigsAndRestartProcesses writes, tests, and restarts Samba using the default dirty mask.
func (self *ServerService) WriteConfigsAndRestartProcesses(ctx context.Context) errors.E {
	return self.writeConfigsAndRestartServers(ctx, defaultDirtyMask)
}

func (self *ServerService) writeSambaConfig(ctx context.Context) errors.E {
	tlog.TraceContext(ctx, "Writing Samba configuration file", "file", self.state.SambaConfigFile)
	stream, errE := self.CreateSambaConfigStream()
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

func (self *ServerService) writeSambaUsersMapConfig(ctx context.Context) errors.E {
	tlog.TraceContext(ctx, "Writing Samba username map file", "file", sambaUsersMapFile)

	// Create directory if it doesn't exist
	dirPath := filepath.Dir(sambaUsersMapFile)
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return errors.WithStack(err)
	}

	stream, err := self.CreateSambaUsersMapStream()
	if err != nil {
		return errors.WithStack(err)
	}

	errWrite := os.WriteFile(sambaUsersMapFile, *stream, 0o644)
	if errWrite != nil {
		return errors.WithStack(errWrite)
	}

	return nil
}

func (self *ServerService) recoverMediaUsageSymlinks(ctx context.Context, linkRoot string) errors.E {
	if strings.TrimSpace(linkRoot) == "" {
		return nil
	}

	shares, err := self.share_service.ListShares()
	if err != nil {
		return errors.WithStack(err)
	}

	if err := os.MkdirAll(linkRoot, 0o755); err != nil {
		return errors.WithStack(err)
	}

	for _, share := range shares {
		if share.Usage != dto.UsageAsMedia {
			continue
		}
		if share.Disabled != nil && *share.Disabled {
			continue
		}
		if share.MountPointData == nil || share.MountPointData.Path == "" {
			continue
		}
		if share.Status != nil && !share.Status.IsValid {
			continue
		}

		targetPath := share.MountPointData.Path
		if !filepath.IsAbs(targetPath) {
			slog.WarnContext(ctx, "Skipping media symlink recovery for non-absolute mount path", "share", share.Name, "target", targetPath)
			continue
		}

		if _, statErr := os.Stat(targetPath); statErr != nil {
			slog.WarnContext(ctx, "Skipping media symlink recovery because target path is unavailable", "share", share.Name, "target", targetPath, "err", statErr)
			continue
		}

		linkPath := filepath.Join(linkRoot, share.Name)
		if info, lstatErr := os.Lstat(linkPath); lstatErr == nil {
			if info.Mode()&os.ModeSymlink == 0 {
				slog.WarnContext(ctx, "Skipping media symlink recovery because destination exists and is not a symlink", "share", share.Name, "link", linkPath)
				continue
			}

			currentTarget, readErr := os.Readlink(linkPath)
			if readErr == nil && currentTarget == targetPath {
				continue
			}

			if removeErr := os.Remove(linkPath); removeErr != nil {
				slog.WarnContext(ctx, "Failed to remove stale media symlink", "share", share.Name, "link", linkPath, "err", removeErr)
				continue
			}
		} else if !os.IsNotExist(lstatErr) {
			slog.WarnContext(ctx, "Failed to inspect media symlink destination", "share", share.Name, "link", linkPath, "err", lstatErr)
			continue
		}

		if symlinkErr := os.Symlink(targetPath, linkPath); symlinkErr != nil {
			slog.WarnContext(ctx, "Failed to create media symlink", "share", share.Name, "target", targetPath, "link", linkPath, "err", symlinkErr)
			continue
		}

		slog.InfoContext(ctx, "Recovered media symlink", "share", share.Name, "target", targetPath, "link", linkPath)
	}

	return nil
}

func (self *ServerService) testSambaConfig(ctx context.Context) errors.E {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	tlog.TraceContext(ctx, "Testing Samba configuration file", "file", self.state.SambaConfigFile)

	out, err := self.runCommandWithRunner(ctx, "samba-testparm", "Validate samba config", []string{"testparm", "-s", self.state.SambaConfigFile})
	if err != nil {
		return errors.Errorf("Error executing testparm: %w \n %#v", err, map[string]any{"error": err, "output": out})
	}
	return nil
}

func (self *ServerService) restartServerServices(ctx context.Context, dirty dto.DataDirtyTracker) errors.E {
	process, err := self.GetServerProcesses()
	if err != nil {
		return errors.WithStack(err)
	}
	// Exec smbcontrol smbd reload-config
	if process != nil {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		for processName, processConfig := range serviceConfigMap {
			if !processConfig.Managed {
				slog.InfoContext(ctx, "Skipping unmanaged service", "service", processName)
				continue
			}
			tlog.TraceContext(ctx, "Restarting service", "service", processName)
			if procStatus, ok := (*process)[processName]; ok {
				if procStatus.Pid <= 0 || dirty.AndMask(processConfig.HardResetServiceMask) {
					slog.InfoContext(ctx, "Performing hard restart of service...", "service", processName)
					outHardRestart, restartErr := self.runCommandWithRunner(ctx, "service-hard-restart-"+processName, "Hard restart "+processName, processConfig.HardResetCommand)
					if restartErr != nil {
						return errors.Errorf("Error performing hard restart of service %s: %w \n %#v", processName, restartErr, map[string]any{"error": restartErr, "output": outHardRestart})
					}
				} else if dirty.AndMask(processConfig.SoftResetServiceMask) {
					slog.InfoContext(ctx, "Performing soft restart of service...", "service", processName)
					outSoftRestart, restartErr := self.runCommandWithRunner(ctx, "service-soft-restart-"+processName, "Soft restart "+processName, processConfig.SoftResetCommand)
					if restartErr != nil {
						return errors.Errorf("Error performing soft restart of service %s: %w \n %#v", processName, restartErr, map[string]any{"error": restartErr, "output": outSoftRestart})
					}
				} else {
					slog.InfoContext(ctx, "No restart needed for service.", "service", processName)
				}
			} else {
				slog.WarnContext(ctx, "Samba process not found, perform start command if exists.", "process", processName)
				if len(processConfig.StartCommand) > 0 && osutil.CommandExists(processConfig.StartCommand) {
					slog.InfoContext(ctx, "Starting service...", "service", processName)
					outStart, startErr := self.runCommandWithRunner(ctx, "service-start-"+processName, "Start "+processName, processConfig.StartCommand)
					if startErr != nil {
						return errors.Errorf("Error starting service %s: %w \n %#v", processName, startErr, map[string]any{"error": startErr, "output": outStart})
					}
				} else {
					slog.InfoContext(ctx, "No start command defined for service or command does not exist, skipping.", "service", processName)
				}
				continue
			}
		}

		self.eventBus.EmitServerProcess(events.ServerProcessEvent{
			Event:            events.Event{Type: events.EventTypes.CLEAN},
			DataDirtyTracker: dto.DataDirtyTracker{},
		})
	} else {
		slog.WarnContext(ctx, "Samba processes not found, skipping reload commands.")
	}
	return nil
}

// WriteSambaConfig Test and Restart
func (self *ServerService) writeConfigsAndRestartServers(ctx context.Context, dirty dto.DataDirtyTracker) errors.E {
	err := self.writeSambaConfig(ctx)
	if err != nil {
		return errors.WithStack(err)
	}
	err = self.writeSambaUsersMapConfig(ctx)
	if err != nil {
		return errors.WithStack(err)
	}
	err = self.testSambaConfig(ctx)
	if err != nil {
		return errors.WithStack(err)
	}

	if setting, err2 := self.setting_service.Load(); err2 == nil && setting.HAUseNFS != nil && *setting.HAUseNFS {
		err = self.writeNFSConfig(ctx)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	err = self.restartServerServices(ctx, dirty)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// writeNFSConfig writes the NFS exports configuration to /etc/exports
func (self *ServerService) writeNFSConfig(ctx context.Context) errors.E {
	nfsExportsFile := "/etc/exports"
	tlog.TraceContext(ctx, "Writing NFS exports configuration file", "file", nfsExportsFile)

	hostname, err := self.host_service.GetHostName()
	if err != nil {
		return errors.WithStack(err)
	}

	// Get all shares from the database
	shares, err := self.share_service.ListShares()
	if err != nil {
		return errors.WithStack(err)
	}

	// Build NFS exports content
	var exportsContent strings.Builder
	exportsContent.WriteString("# NFS exports generated by SRAT\n")
	exportsContent.WriteString("# Do not edit this file manually - changes will be overwritten\n\n")

	exportCount := 0
	for _, share := range shares {
		// Skip disabled shares
		if share.Disabled != nil && *share.Disabled {
			continue
		}

		// Skip shares with invalid status
		if share.Status != nil && !share.Status.IsValid {
			continue
		}

		// Skip shares with invalid mount point
		if share.MountPointData != nil && share.MountPointData.IsInvalid {
			continue
		}

		// Skip shares without mount point data
		if share.MountPointData == nil {
			continue
		}

		// Only export shares with usage type: media, share, or backup
		usage := string(share.Usage)
		if usage != "media" && usage != "share" && usage != "backup" {
			continue
		}

		// Get the share path from mount point data
		path := share.MountPointData.Path
		if path == "" {
			tlog.WarnContext(ctx, "Skipping share with empty path", "name", share.Name)
			continue
		}

		// Skip if mount point data fs is not exportable (e.g. apfs)
		if isShareNFSExportable(ctx, share) {
			// Generate NFS export entry
			// Format: /path/to/share *(rw,sync,no_subtree_check,no_root_squash,fsid=X)
			// Using fsid based on share index to ensure unique identification
			exportsContent.WriteString(path)
			exportsContent.WriteString(" ")
			exportsContent.WriteString(hostname)
			exportsContent.WriteString("(rw,sync,mp,no_subtree_check,no_root_squash")
			// fsid need to be a UUID valid format
			//exportsContent.WriteString(",fsid=")
			//exportsContent.WriteString(strings.ReplaceAll(share.MountPointData.DeviceId, "-", ""))
			exportsContent.WriteString(")\n")

			exportCount++
			tlog.DebugContext(ctx, "Added NFS export", "name", share.Name, "path", path, "usage", usage)
		} else {
			tlog.DebugContext(ctx, "Skipping share with non-exportable filesystem", "name", share.Name, "filesystem", spew.Sdump(share.MountPointData))
			continue
		}

	}

	slog.InfoContext(ctx, "Generated NFS exports configuration", "exportCount", exportCount)

	// Write the exports file
	err2 := os.WriteFile(nfsExportsFile, []byte(exportsContent.String()), 0o644)
	if err2 != nil {
		return errors.WithStack(err2)
	}

	return nil
}

func (self *ServerService) SetState(state *dto.ContextState) {
	self.state = state
}
