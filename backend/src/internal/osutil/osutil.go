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

	"github.com/prometheus/procfs"
	"golang.org/x/sys/unix"
)

// MountInfoEntry contains data from /proc/self/mountinfo.
// Ex: 1546 1508 8:8 /supervisor/media /media rw,relatime master:112 - ext4 /dev/sda8 rw,commit=30
type MountInfoEntry struct {
	MountID        int               // 1546
	ParentID       int               // 1508
	DevMajor       int               // 8
	DevMinor       int               // 8
	Root           string            // /supervisor/media
	MountDir       string            // /media
	MountOptions   map[string]string // rw,relatime
	OptionalFields []string          // master:112
	FsType         string            // ext4
	MountSource    string            // /dev/sda8
	SuperOptions   map[string]string // rw,commit=30
}

var (
	overrideMu            sync.RWMutex
	mountsOverride        func() ([]*procfs.MountInfo, error)
	filesystemsOverride   func() ([]string, error)
	filesystemsOverrideMu sync.RWMutex
)

// MockMountInfo replaces the mount info reader with the provided mount info string content
// until the returned restore function is called. This is useful for testing.
// The format should be mount info strings as they appear in /proc/self/mountinfo, one per line.
func MockMountInfo(content string) (restore func()) {
	mounts, _ := parseMountInfoString(content)
	return setMountsOverride(func() ([]*procfs.MountInfo, error) {
		return convertToProcs(mounts), nil
	})
}

// MockFileSystems replaces GetFileSystems with the provided filesystem list
// until the returned restore function is called. This is useful for testing.
func MockFileSystems(filesystems []string) (restore func()) {
	filesystemsOverrideMu.Lock()
	defer filesystemsOverrideMu.Unlock()

	oldOverride := filesystemsOverride
	filesystemsOverride = func() ([]string, error) {
		return filesystems, nil
	}

	return func() {
		filesystemsOverrideMu.Lock()
		defer filesystemsOverrideMu.Unlock()
		filesystemsOverride = oldOverride
	}
}

// parseMountInfoString parses mount info text format into MountInfoEntry structs.
func parseMountInfoString(content string) ([]*MountInfoEntry, error) {
	entries := []*MountInfoEntry{}
	if content == "" {
		return entries, nil
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		entry, err := parseMountLine(line)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

// parseMountLine parses a single mount info line.
func parseMountLine(line string) (*MountInfoEntry, error) {
	fields := strings.Fields(line)
	if len(fields) < 10 {
		return nil, nil // Skip invalid lines
	}

	// Find the separator "-"
	separatorIdx := -1
	for i := 6; i < len(fields)-3; i++ {
		if fields[i] == "-" {
			separatorIdx = i
			break
		}
	}

	if separatorIdx == -1 {
		return nil, nil
	}

	mountID, _ := strconv.Atoi(fields[0])
	parentID, _ := strconv.Atoi(fields[1])

	// Parse major:minor
	devParts := strings.Split(fields[2], ":")
	devMajor, _ := strconv.Atoi(devParts[0])
	devMinor := 0
	if len(devParts) > 1 {
		devMinor, _ = strconv.Atoi(devParts[1])
	}

	entry := &MountInfoEntry{
		MountID:        mountID,
		ParentID:       parentID,
		DevMajor:       devMajor,
		DevMinor:       devMinor,
		Root:           fields[3],
		MountDir:       fields[4],
		MountOptions:   parseOptions(fields[5]),
		OptionalFields: fields[6:separatorIdx],
		FsType:         fields[separatorIdx+1],
		MountSource:    fields[separatorIdx+2],
		SuperOptions:   parseOptions(fields[separatorIdx+3]),
	}

	return entry, nil
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

// convertInfos is a wrapper for backward compatibility with tests.
func convertInfos(entries []*MountInfoEntry) []*MountInfoEntry {
	if entries == nil {
		return make([]*MountInfoEntry, 0)
	}
	return entries
}

// convertToProcs converts MountInfoEntry slice to procfs.MountInfo slice.
func convertToProcs(entries []*MountInfoEntry) []*procfs.MountInfo {
	if entries == nil {
		return nil
	}
	result := make([]*procfs.MountInfo, len(entries))
	for i, entry := range entries {
		result[i] = &procfs.MountInfo{
			MountID:        entry.MountID,
			ParentID:       entry.ParentID,
			MajorMinorVer:  strconv.Itoa(entry.DevMajor) + ":" + strconv.Itoa(entry.DevMinor),
			Root:           entry.Root,
			MountPoint:     entry.MountDir,
			Options:        entry.MountOptions,
			OptionalFields: convertToMap(entry.OptionalFields),
			FSType:         entry.FsType,
			Source:         entry.MountSource,
			SuperOptions:   entry.SuperOptions,
		}
	}
	return result
}

// convertToMap converts optional fields slice to map.
func convertToMap(fields []string) map[string]string {
	result := make(map[string]string)
	for _, field := range fields {
		parts := strings.SplitN(field, ":", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		} else {
			result[field] = ""
		}
	}
	return result
}

func setMountsOverride(fn func() ([]*procfs.MountInfo, error)) (restore func()) {
	overrideMu.Lock()
	previous := mountsOverride
	mountsOverride = fn
	overrideMu.Unlock()

	return func() {
		overrideMu.Lock()
		mountsOverride = previous
		overrideMu.Unlock()
	}
}

// LoadMountInfo is implemented in platform-specific files:
//   - loadmountinfo_linux.go: reads /proc/self/mountinfo via procfs
//   - osutil_darwin.go: returns empty list (no /proc on macOS)

// convertFromProcs converts procfs.MountInfo slice to MountInfoEntry slice.
func convertFromProcs(mounts []*procfs.MountInfo) []*MountInfoEntry {
	if mounts == nil {
		return nil
	}
	result := make([]*MountInfoEntry, len(mounts))
	for i, mount := range mounts {
		devMajor, devMinor := 0, 0
		parts := strings.Split(mount.MajorMinorVer, ":")
		if len(parts) > 0 {
			devMajor, _ = strconv.Atoi(parts[0])
		}
		if len(parts) > 1 {
			devMinor, _ = strconv.Atoi(parts[1])
		}

		result[i] = &MountInfoEntry{
			MountID:        mount.MountID,
			ParentID:       mount.ParentID,
			DevMajor:       devMajor,
			DevMinor:       devMinor,
			Root:           mount.Root,
			MountDir:       mount.MountPoint,
			MountOptions:   mount.Options,
			OptionalFields: convertMapToSlice(mount.OptionalFields),
			FsType:         mount.FSType,
			MountSource:    mount.Source,
			SuperOptions:   mount.SuperOptions,
		}
	}
	return result
}

// convertMapToSlice converts optional fields map back to slice format.
func convertMapToSlice(fields map[string]string) []string {
	result := make([]string, 0, len(fields))
	for k, v := range fields {
		if v != "" {
			result = append(result, k+":"+v)
		} else {
			result = append(result, k)
		}
	}
	return result
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

// CommandExists checks if a command is available and executable.
// For s6-* commands, it validates that the service directory path exists.
// For other commands, it checks if the command is in PATH and is executable.
func CommandExists(cmd []string) bool {
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

// ReadLinesOffsetN reads contents from file and splits them by new line.
// The offset tells at which line number to start.
// The count determines the number of lines to read (starting from offset):
// n >= 0: at most n lines
// n < 0: whole file
// Source: https://github.com/shirou/gopsutil
func readLinesOffsetN(filename string, offset uint, n int) ([]string, error) {
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
