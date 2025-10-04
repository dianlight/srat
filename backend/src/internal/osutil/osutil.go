package osutil

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/moby/sys/mountinfo"
	"golang.org/x/sys/unix"
)

// MountInfoEntry contains data from /proc/<pid>/mountinfo.
type MountInfoEntry struct {
	MountID        int
	ParentID       int
	DevMajor       int
	DevMinor       int
	Root           string
	MountDir       string
	MountOptions   map[string]string
	OptionalFields []string
	FsType         string
	MountSource    string
	SuperOptions   map[string]string
}

var (
	overrideMu        sync.RWMutex
	mountInfoOverride mountInfoSource
)

type mountInfoSource func() (io.ReadCloser, error)

// MockMountInfo replaces the mountinfo reader with the provided content until the
// returned restore function is called.
func MockMountInfo(content string) (restore func()) {
	return setMountInfoSource(func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(content)), nil
	})
}

func setMountInfoSource(fn mountInfoSource) (restore func()) {
	overrideMu.Lock()
	previous := mountInfoOverride
	mountInfoOverride = fn
	overrideMu.Unlock()

	return func() {
		overrideMu.Lock()
		mountInfoOverride = previous
		overrideMu.Unlock()
	}
}

// LoadMountInfo loads the current mount information for the running process.
func LoadMountInfo() ([]*MountInfoEntry, error) {
	overrideMu.RLock()
	override := mountInfoOverride
	overrideMu.RUnlock()

	if override == nil {
		infos, err := mountinfo.GetMounts(nil)
		if err != nil {
			return nil, err
		}
		return convertInfos(infos), nil
	}

	reader, err := override()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	infos, err := mountinfo.GetMountsFromReader(reader, nil)
	if err != nil {
		return nil, err
	}
	return convertInfos(infos), nil
}

func convertInfos(infos []*mountinfo.Info) []*MountInfoEntry {
	entries := make([]*MountInfoEntry, 0, len(infos))
	for _, info := range infos {
		entries = append(entries, &MountInfoEntry{
			MountID:        info.ID,
			ParentID:       info.Parent,
			DevMajor:       info.Major,
			DevMinor:       info.Minor,
			Root:           info.Root,
			MountDir:       info.Mountpoint,
			MountOptions:   parseOptions(info.Options),
			OptionalFields: parseOptional(info.Optional),
			FsType:         info.FSType,
			MountSource:    info.Source,
			SuperOptions:   parseOptions(info.VFSOptions),
		})
	}
	return entries
}

func parseOptions(opts string) map[string]string {
	result := make(map[string]string)
	if opts == "" {
		return result
	}
	for _, opt := range strings.Split(opts, ",") {
		keyValue := strings.SplitN(opt, "=", 2)
		key := keyValue[0]
		if len(keyValue) == 2 {
			result[key] = keyValue[1]
		} else {
			result[key] = ""
		}
	}
	return result
}

func parseOptional(optional string) []string {
	optional = strings.TrimSpace(optional)
	if optional == "" {
		return nil
	}
	fields := strings.Fields(optional)
	out := make([]string, len(fields))
	copy(out, fields)
	return out
}

// IsMounted checks whether the provided path is present in the mount table.
func IsMounted(path string) (bool, error) {
	entries, err := LoadMountInfo()
	if err != nil {
		return false, err
	}
	target := filepath.Clean(path)
	for _, entry := range entries {
		if filepath.Clean(entry.MountDir) == target {
			return true, nil
		}
	}
	return false, nil
}

// IsWritable returns true when the current user has write access to the path.
func IsWritable(path string) bool {
	return unix.Access(path, unix.W_OK) == nil
}

// IsKernelModuleLoaded checks if a kernel module is currently loaded.
// It reads /proc/modules to determine if the module exists.
func IsKernelModuleLoaded(moduleName string) (bool, error) {
	file, err := os.Open("/proc/modules")
	if err != nil {
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Module format: modulename size refcount dependencies state offset
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == moduleName {
			return true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	return false, nil
}

// IsLibraryAvailable checks if a shared library is available on the system.
// It uses ldconfig to check for the library.
func IsLibraryAvailable(libraryName string) (bool, error) {
	// Try using ldconfig -p to list all cached libraries
	cmd := exec.Command("ldconfig", "-p")
	output, err := cmd.Output()
	if err != nil {
		// If ldconfig fails, try pkg-config as fallback
		return isLibraryAvailableViaPkgConfig(libraryName)
	}

	// Search for the library in ldconfig output
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, libraryName) {
			return true, nil
		}
	}

	return false, scanner.Err()
}

// isLibraryAvailableViaPkgConfig checks library availability using pkg-config
func isLibraryAvailableViaPkgConfig(libraryName string) (bool, error) {
	// Remove lib prefix and .so suffix if present
	pkgName := strings.TrimPrefix(libraryName, "lib")
	pkgName = strings.TrimSuffix(pkgName, ".so")
	
	cmd := exec.Command("pkg-config", "--exists", pkgName)
	err := cmd.Run()
	return err == nil, nil
}

// GetSambaVersion retrieves the installed Samba version.
// Returns version string (e.g., "4.23.0") or empty string if not found.
func GetSambaVersion() (string, error) {
	cmd := exec.Command("smbd", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse version from output like "Version 4.23.0"
	versionLine := strings.TrimSpace(string(output))
	parts := strings.Fields(versionLine)
	if len(parts) >= 2 && strings.ToLower(parts[0]) == "version" {
		return parts[1], nil
	}

	return "", nil
}

// IsSambaVersionSufficient checks if Samba version meets minimum requirement.
// Returns true if version >= 4.23.0
func IsSambaVersionSufficient() (bool, error) {
	version, err := GetSambaVersion()
	if err != nil || version == "" {
		return false, err
	}

	// Parse version string
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false, nil
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return false, err
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return false, err
	}

	// Check if version >= 4.23
	if major > 4 || (major == 4 && minor >= 23) {
		return true, nil
	}

	return false, nil
}
