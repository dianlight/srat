package api

import (
	"context"
	"log/slog"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/srat/service"
	"github.com/shirou/gopsutil/v4/net"
)

type SystemHanler struct {
	host_service service.HostServiceInterface
}

func NewSystemHanler(host_service service.HostServiceInterface) *SystemHanler {
	p := new(SystemHanler)
	p.host_service = host_service
	return p
}

func (self *SystemHanler) RegisterSystemHanler(api huma.API) {
	huma.Get(api, "/nics", self.GetNICsHandler, huma.OperationTags("system"))
	huma.Get(api, "/hostname", self.GetHostnameHandler, huma.OperationTags("system"))
	huma.Get(api, "/capabilities", self.GetCapabilitiesHandler, huma.OperationTags("system"))
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
	sambaVersionSufficient, err := osutil.IsSambaVersionSufficient()
	if err != nil {
		slog.Warn("Failed to check Samba version", "error", err)
		capabilities.SambaVersion = "unknown"
		capabilities.SambaVersionSufficient = false
		reasons = append(reasons, "Samba version could not be determined")
	} else {
		version, _ := osutil.GetSambaVersion()
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

	return &struct{ Body dto.SystemCapabilities }{Body: capabilities}, nil
}
