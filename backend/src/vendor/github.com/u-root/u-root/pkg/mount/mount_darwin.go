//go:build darwin

package mount

import "errors"

var errUnsupported = errors.New("mount operations not supported on this platform")

// Mounter is a device that can be attached at a file system path.
type Mounter interface {
	DevName() string
	Mount(path string, flags uintptr, opts ...func() error) (*MountPoint, error)
}

// MountPoint represents a mounted file system.
type MountPoint struct {
	Path   string
	Device string
	FSType string
	Flags  uintptr
	Data   string
}

// Mount mounts a filesystem.
func Mount(dev, path, fsType, data string, flags uintptr, opts ...func() error) (*MountPoint, error) {
	return nil, errUnsupported
}

// TryMount attempts to mount a filesystem, deriving fstype from the block device.
func TryMount(device, path, data string, flags uintptr, opts ...func() error) (*MountPoint, error) {
	return nil, errUnsupported
}

// Unmount unmounts the file system at path.
func Unmount(path string, force, lazy bool) error {
	return errUnsupported
}

// SameFilesystem checks if two paths are on the same filesystem.
func SameFilesystem(path1, path2 string) (bool, error) {
	return false, errUnsupported
}

// FSFromBlock returns the filesystem type and flags for a block device.
func FSFromBlock(n string) (string, uintptr, error) {
	return "", 0, errUnsupported
}
