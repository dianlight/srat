package converter

import (
	"fmt"
	"os"
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
