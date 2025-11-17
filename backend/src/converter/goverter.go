package converter

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/dianlight/srat/internal/osutil"
	"github.com/u-root/u-root/pkg/mount"
	"github.com/xorcare/pointer"
)

//go:generate go tool goverter gen ./...

var osStat = os.Stat

func MockFuncOsStat(fn func(name string) (os.FileInfo, error)) {
	osStat = fn
}

// isPathDirNotExists checks if a given path string points to an existing directory.
// It returns true if the path exists and is a directory, false otherwise.
// An error is returned if there's an issue with os.Stat (other than os.IsNotExist).
func isPathDirNotExists(path string) (bool, error) {
	fi, err := osStat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Path does not exist.
			return true, nil
		}
		// Another error occurred while stating the path.
		return true, fmt.Errorf("error stating path %s: %w", path, err)
	}

	// Path exists, check if it's a directory.
	return !fi.IsDir(), nil
}

func deviceToDeviceId(source string) (string, error) {
	deviceID := source
	entries, err := os.ReadDir("/dev/disk/by-id/")
	if err == nil {
		for _, entry := range entries {
			if entry.Type()&os.ModeSymlink != 0 {
				linkPath := filepath.Join("/dev/disk/by-id/", entry.Name())
				resolved, err := filepath.EvalSymlinks(linkPath)
				if err != nil {
					continue
				}
				//slog.Debug("Resolved symlink", "link", linkPath, "resolved", resolved, "source", source)
				if resolved == source || linkPath == source {
					deviceID = "by-id-" + entry.Name()
					break
				}
			}
		}
	}
	return deviceID, nil
}

func mountPathToDeviceId(mountPath string) (string, error) {
	slog.Debug("Resolving device ID for mount path", "mountPath", mountPath)
	info, err := osutil.LoadMountInfo()
	if err != nil {
		slog.Warn("Error loading mount info", "err", err)
		return "", nil
	}
	slog.Info("Loaded mount info", "count", len(info), "all", info)
	for _, m := range info {
		slog.Info("Mount info", "mount_dir", m.MountDir, "mount_source", m.MountSource, "mountPath", mountPath, "all", m)
		if m.MountDir == mountPath {
			return deviceToDeviceId(m.MountSource)
		} else {
			same, _ := mount.SameFilesystem(mountPath, m.MountDir)
			if same {
				slog.Info("Same filesystem detected", "mountPath", mountPath, "mountDir", m.MountDir)
				return mountPathToDeviceId(m.MountDir)
			}
		}
	}
	return "", nil
}

func falsePConst() *bool {
	return pointer.Bool(false)
}

func falseConst() bool {
	return false
}

/*
func truePConst() *bool {
	return pointer.Bool(true)
}

func trueConst() bool {
	return true
}
*/
