package api

import (
	"context"
	"log/slog"
	stdnet "net"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/commandexec"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/srat/unixsamba"
	"github.com/shirou/gopsutil/v4/net"
	"go.uber.org/fx"
)

type SystemHanlerParams struct {
	fx.In
	HostService     service.HostServiceInterface
	CommandExecutor commandexec.Executor
	ApiCtx          *dto.ContextState `optional:"true"`
}

type SystemHanler struct {
	host_service     service.HostServiceInterface
	command_executor commandexec.Executor
	apiCtx           *dto.ContextState
}

func NewSystemHanler(in SystemHanlerParams) *SystemHanler {
	p := new(SystemHanler)
	p.host_service = in.HostService
	p.command_executor = in.CommandExecutor
	p.apiCtx = in.ApiCtx
	return p
}

func (self *SystemHanler) RegisterSystemHanler(api huma.API) {
	huma.Get(api, "/welcome", self.HandleWelcome, huma.OperationTags("system", "internal"))
	huma.Get(api, "/appconfig", self.HandleAppConfig, huma.OperationTags("system", "internal"))
	huma.Get(api, "/command_output", self.HandleCommandOutput, huma.OperationTags("system", "internal"))
	huma.Get(api, "/command_events", self.HandleCommandEvents, huma.OperationTags("system", "internal"))
	huma.Get(api, "/mdns_events", self.HandleMdnsEvents, huma.OperationTags("system", "internal"))
	huma.Get(api, "/nics", self.GetNICsHandler, huma.OperationTags("system"))
	huma.Get(api, "/hostname", self.GetHostnameHandler, huma.OperationTags("system"))
	huma.Get(api, "/capabilities", self.GetCapabilitiesHandler, huma.OperationTags("system"))
}

func (self *SystemHanler) HandleWelcome(ctx context.Context, input *struct{}) (*struct{ Body dto.Welcome }, error) {
	return nil, huma.Error500InternalServerError("You are not welcome!", nil)
}

func (self *SystemHanler) HandleAppConfig(ctx context.Context, input *struct{}) (*struct {
	Body dto.AppConfigChangedNotification
}, error) {
	return nil, huma.Error500InternalServerError("App configuration not available", nil)
}

// HandleCommandEvents is a documentation-only stub that anchors the WebSocket command event
// schemas (CommandStartedNotification, CommandOutputNotification, CommandTerminatedNotification)
// into the OpenAPI spec so they are code-generated into the frontend TypeScript types.
// Actual command events are delivered over the WebSocket connection.
func (self *SystemHanler) HandleCommandEvents(ctx context.Context, input *struct{}) (*struct {
	Body struct {
		Started    *dto.CommandStartedNotification    `json:"started,omitempty"`
		Output     *dto.CommandOutputNotification     `json:"output,omitempty"`
		Terminated *dto.CommandTerminatedNotification `json:"terminated,omitempty"`
		Problem    *dto.Problem                       `json:"problem,omitempty"`
	}
}, error) {
	return nil, huma.Error500InternalServerError("Use WebSocket for command events", nil)
}

// HandleMdnsEvents is a documentation-only stub that anchors the MdnsRegisterNotification
// WebSocket event schema into the OpenAPI spec so it is code-generated into frontend TypeScript types.
// Actual mDNS events are delivered over the WebSocket connection.
func (self *SystemHanler) HandleMdnsEvents(ctx context.Context, input *struct{}) (*struct {
	Body dto.MdnsRegisterNotification
}, error) {
	return nil, huma.Error500InternalServerError("Use WebSocket for mDNS events", nil)
}

func (self *SystemHanler) HandleCommandOutput(ctx context.Context, input *struct {
	ExecutionID string `query:"execution_id" doc:"Command execution ID to inspect"`
}) (*struct{ Body dto.CommandExecutionSnapshot }, error) {
	executionID := strings.TrimSpace(input.ExecutionID)
	if executionID == "" {
		return nil, huma.Error400BadRequest("execution_id is required", nil)
	}
	if self.command_executor == nil {
		return nil, huma.Error500InternalServerError("Command output service is not available", nil)
	}

	snapshot, ok := self.command_executor.GetSnapshot(executionID)
	if !ok {
		return nil, huma.Error404NotFound("Command output not available", nil)
	}

	return &struct{ Body dto.CommandExecutionSnapshot }{Body: snapshot}, nil
}

// GetNICsHandler handles the request to retrieve network interface card (NIC) information.
// It uses the ghw library to get the network information and converts it to a DTO (Data Transfer Object).
//
// Parameters:
//   - ctx: The context for the request, which can be used for cancellation and deadlines.
//   - input: An empty struct as input, as no specific input is required for this handler.
//
// Returns:
//   - A struct containing the network information in the Body field.
//   - An error if there is any issue retrieving or converting the network information.
func (handler *SystemHanler) GetNICsHandler(ctx context.Context, input *struct{}) (*struct{ Body net.InterfaceStatList }, error) {

	//	net, err := ghw.Network()
	//	if err != nil {
	//		return nil, err
	//	}

	nics, err := net.InterfacesWithContext(ctx)
	if err != nil {
		return nil, err
	}

	// Filter out veth* (virtual ethernet) interfaces used by containers
	filteredNics := make(net.InterfaceStatList, 0, len(nics))
	for _, nic := range nics {
		if !strings.HasPrefix(nic.Name, "veth") {
			filteredNics = append(filteredNics, nic)
		}
	}

	return &struct{ Body net.InterfaceStatList }{Body: filteredNics}, nil
}

// GetHostnameHandler handles the request to retrieve the system's hostname.
// It uses the HostService to get the hostname and returns it in a DTO.
//
// Parameters:
//   - ctx: The context for the request.
//   - input: An empty struct as input.
//
// Returns:
//   - A struct containing the hostname in the Body field.
//   - An error if there is any issue retrieving the hostname.
func (handler *SystemHanler) GetHostnameHandler(ctx context.Context, input *struct{}) (*struct{ Body string }, error) {
	hostname, err := handler.host_service.GetHostName()
	if err != nil {
		return nil, err // Error is already descriptive from the service
	}
	return &struct{ Body string }{Body: hostname}, nil
}

// GetCapabilitiesHandler retrieves the system capabilities.
// It checks for various system features like QUIC support.
//
// Parameters:
//   - ctx: The context for the request.
//   - input: An empty struct as input.
//
// Returns:
//   - A struct containing the system capabilities in the Body field.
//   - An error if there is any issue checking the capabilities.
func (handler *SystemHanler) GetCapabilitiesHandler(ctx context.Context, input *struct{}) (*struct{ Body dto.SystemCapabilities }, error) {
	capabilities := dto.SystemCapabilities{}
	var reasons []string

	// Check Samba version
	sambaVersionSufficient, err := unixsamba.IsSambaVersionSufficient()
	if err != nil {
		slog.Warn("Failed to check Samba version", "error", err)
		capabilities.SambaVersion = "unknown"
		capabilities.SambaVersionSufficient = false
		reasons = append(reasons, "Samba version could not be determined")
	} else {
		version, _ := unixsamba.GetSambaVersion()
		capabilities.SambaVersion = version
		capabilities.SambaVersionSufficient = sambaVersionSufficient
		if !sambaVersionSufficient {
			reasons = append(reasons, "Samba version must be >= 4.23.0")
		}
	}

	// Check if QUIC kernel module is loaded
	// The quic module might be named differently on different systems
	// Common names: quic, net_quic, or built into the kernel
	quicLoaded, err := osutil.IsKernelModuleLoaded("quic")
	if err != nil {
		slog.Warn("Failed to check QUIC kernel module", "error", err)
		capabilities.HasKernelModule = false
	} else {
		capabilities.HasKernelModule = quicLoaded
	}

	// If "quic" module not found, try "net_quic"
	if !capabilities.HasKernelModule {
		netQuicLoaded, err := osutil.IsKernelModuleLoaded("net_quic")
		if err != nil {
			slog.Warn("Failed to check net_quic kernel module", "error", err)
		} else {
			capabilities.HasKernelModule = netQuicLoaded
		}
	}

	// QUIC is supported if Samba version is sufficient AND kernel module is loaded
	if !capabilities.HasKernelModule {
		reasons = append(reasons, "QUIC kernel module (quic or net_quic) not loaded")
	}

	capabilities.SupportsQUIC = capabilities.SambaVersionSufficient && capabilities.HasKernelModule

	if !capabilities.SupportsQUIC && len(reasons) > 0 {
		capabilities.UnsupportedReason = strings.Join(reasons, "; ")
	}

	// Check NFS support
	supportNFS := osutil.CommandExists([]string{"exportfs"})
	capabilities.SupportNFS = supportNFS

	// Report lib SMART backend availability from runtime context
	if handler.apiCtx != nil {
		capabilities.LibSmartAvailable = handler.apiCtx.LibSmartAvailable
	}

	// Report network interfaces eligible for addon-side direct mDNS registration.
	capabilities.AvailableMDNSInterfaces = availableMDNSInterfaces()

	return &struct{ Body dto.SystemCapabilities }{Body: capabilities}, nil
}

// availableMDNSInterfaces returns the names of network interfaces that are
// eligible for addon-side direct mDNS registration. Loopback, down, and
// container/virtual interfaces (docker*, veth*, hassio*, br-*) are excluded.
func availableMDNSInterfaces() []string {
	ifaces, err := stdnet.Interfaces()
	if err != nil {
		slog.Warn("Failed to list network interfaces for mDNS capabilities", "error", err)
		return nil
	}

	var names []string
	for _, iface := range ifaces {
		if iface.Flags&stdnet.FlagUp == 0 {
			continue
		}
		if iface.Flags&stdnet.FlagLoopback != 0 {
			continue
		}
		name := iface.Name
		if strings.HasPrefix(name, "lo") ||
			strings.HasPrefix(name, "docker") ||
			strings.HasPrefix(name, "veth") ||
			strings.HasPrefix(name, "hassio") ||
			strings.HasPrefix(name, "br-") {
			continue
		}
		names = append(names, name)
	}
	return names
}
