package converter

import (
	"fmt"
	"os"
	"path/filepath"
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
	deviceID := ""
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
