package api

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/srat/service"
	"github.com/jpillora/overseer"
	"github.com/shirou/gopsutil/v4/net"
)

type SystemHanler struct {
	fs_service      service.FilesystemServiceInterface
	host_service    service.HostServiceInterface
	filesystemsPath string
}

func NewSystemHanler(fs_service service.FilesystemServiceInterface, host_service service.HostServiceInterface) *SystemHanler {
	p := new(SystemHanler)
	p.fs_service = fs_service
	p.host_service = host_service
	p.filesystemsPath = "/proc/filesystems"
	return p
}

func (self *SystemHanler) SetFilesystemsPath(path string) {
	self.filesystemsPath = path
}

func (self *SystemHanler) RegisterSystemHanler(api huma.API) {
	huma.Put(api, "/restart", self.RestartHandler, huma.OperationTags("system"))
	huma.Get(api, "/nics", self.GetNICsHandler, huma.OperationTags("system"))
	huma.Get(api, "/hostname", self.GetHostnameHandler, huma.OperationTags("system"))
	huma.Get(api, "/filesystems", self.GetFSHandler, huma.OperationTags("system"))
	huma.Get(api, "/capabilities", self.GetCapabilitiesHandler, huma.OperationTags("system"))
}

// RestartHandler handles the request to restart the server.
// It logs the restart action and calls the overseer to perform the restart.
//
// Parameters:
//   - ctx: The context for the request.
//   - input: An empty struct as input.
//
// Returns:
//   - An empty struct and an error, both of which are nil.
func (handler *SystemHanler) RestartHandler(ctx context.Context, input *struct{}) (*struct{}, error) {
	slog.Debug("Restarting server...")
	overseer.Restart()
	return nil, nil
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

// ReadLinesOffsetN reads contents from file and splits them by new line.
// The offset tells at which line number to start.
// The count determines the number of lines to read (starting from offset):
// n >= 0: at most n lines
// n < 0: whole file
// Source: https://github.com/shirou/gopsutil
func (handler *SystemHanler) readLinesOffsetN(filename string, offset uint, n int) ([]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return []string{""}, err
	}
	defer f.Close()

	var ret []string

	r := bufio.NewReader(f)
	for i := uint(0); i < uint(n)+offset || n < 0; i++ {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF && len(line) > 0 {
				ret = append(ret, strings.Trim(line, "\n"))
			}
			break
		}
		if i < offset {
			continue
		}
		ret = append(ret, strings.Trim(line, "\n"))
	}

	return ret, nil
}

// Source: https://github.com/shirou/gopsutil
func (handler *SystemHanler) getFileSystems() ([]string, error) {
	filename := handler.filesystemsPath
	lines, err := handler.readLinesOffsetN(filename, 0, -1)
	if err != nil {
		return nil, err
	}
	var ret []string
	seen := make(map[string]struct{})
	allowedNodev := map[string]struct{}{
		"zfs":     {},
		"fuse":    {},
		"fuse3":   {},
		"fuseblk": {},
	}
	for _, line := range lines {
		cleaned := strings.TrimSpace(line)
		if cleaned == "" {
			continue
		}
		if !strings.HasPrefix(cleaned, "nodev") {
			if _, exists := seen[cleaned]; !exists {
				ret = append(ret, cleaned)
				seen[cleaned] = struct{}{}
			}
			continue
		}
		fields := strings.Fields(cleaned)
		if len(fields) != 2 {
			continue
		}
		fsType := strings.TrimSpace(fields[1])
		if _, ok := allowedNodev[fsType]; ok {
			if _, exists := seen[fsType]; !exists {
				ret = append(ret, fsType)
				seen[fsType] = struct{}{}
			}
		}
	}

	return ret, nil
}

// GetFSHandler retrieves the filesystem types available on the system.
// It returns a struct containing a slice of FilesystemTypes and an error if any occurred during the retrieval process.
//
// Parameters:
//   - ctx: The context for the request, used for cancellation and deadlines.
//   - input: An empty struct, reserved for future use.
//
// Returns:
//   - A pointer to a struct containing a slice of FilesystemTypes in the Body field.
//   - An error if there was an issue retrieving the filesystem types.
func (handler *SystemHanler) GetFSHandler(ctx context.Context, input *struct{}) (*struct{ Body dto.FilesystemTypes }, error) {

	fs, err := handler.getFileSystems()
	if err != nil {
		return nil, err
	}
	xfs := make(dto.FilesystemTypes, len(fs))
	for i, fsi := range fs {
		flags, _ := handler.fs_service.GetStandardMountFlags()
		data, _ := handler.fs_service.GetFilesystemSpecificMountFlags(fsi)

		xfs[i] = dto.FilesystemType{
			Name:             fsi,
			Type:             fsi,
			MountFlags:       flags,
			CustomMountFlags: data,
		}
	}
	return &struct{ Body dto.FilesystemTypes }{Body: xfs}, nil
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

	return &struct{ Body dto.SystemCapabilities }{Body: capabilities}, nil
}
