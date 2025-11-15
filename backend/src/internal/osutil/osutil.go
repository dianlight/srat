package osutil

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
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
	versionOverride   string
	versionOverrideMu sync.RWMutex
)

type mountInfoSource func() (io.ReadCloser, error)

// MockMountInfo replaces the mountinfo reader with the provided content until the
// returned restore function is called.
func MockMountInfo(content string) (restore func()) {
	return setMountInfoSource(func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(content)), nil
	})
}

// MockSambaVersion replaces the Samba version for testing purposes until the
// returned restore function is called.
func MockSambaVersion(version string) (restore func()) {
	versionOverrideMu.Lock()
	previousVersion := versionOverride
	versionOverride = version
	versionOverrideMu.Unlock()

	return func() {
		versionOverrideMu.Lock()
		versionOverride = previousVersion
		versionOverrideMu.Unlock()
	}
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
		infos, err := mountinfo.GetMounts(func(m *mountinfo.Info) (skip, stop bool) {
			if info, err := os.Stat(m.Mountpoint); err == nil && info.IsDir() {
				return false, false
			} else {
				return true, false
			}
		})
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

// GetSambaVersion retrieves the installed Samba version.
// Returns version string (e.g., "4.23.0") or empty string if not found.
func GetSambaVersion() (string, error) {
	// Check if version is mocked (for testing)
	versionOverrideMu.RLock()
	if versionOverride != "" {
		defer versionOverrideMu.RUnlock()
		return versionOverride, nil
	}
	versionOverrideMu.RUnlock()

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

// IsSambaVersionAtLeast checks if the installed Samba version meets the specified minimum version.
// Example: IsSambaVersionAtLeast(4, 21) returns true if version >= 4.21.0
func IsSambaVersionAtLeast(majorRequired, minorRequired int) (bool, error) {
	version, err := GetSambaVersion()
	if err != nil || version == "" {
		return false, err
	}

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

	// Check if version >= required
	if major > majorRequired || (major == majorRequired && minor >= minorRequired) {
		return true, nil
	}

	return false, nil
}

// GenerateSecurePassword generates a cryptographically secure random password.
// It uses crypto/rand to generate random bytes and encodes them in base64.
// The resulting password will be approximately 22 characters long (16 bytes * 4/3).
func GenerateSecurePassword() (string, error) {
	// Generate 16 random bytes (128 bits of entropy)
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	// Encode to base64 for a safe password string
	// Using RawURLEncoding to avoid padding and make it URL-safe
	password := base64.RawURLEncoding.EncodeToString(randomBytes)
	return password, nil
}
