//go:build darwin

package mount

import "errors"

var errUnsupported = errors.New("mount operations not supported on this platform")

type Mounter interface {
	DevName() string
	Mount(path string, flags uintptr, opts ...func() error) (*MountPoint, error)
}

type MountPoint struct {
	Path   string
	Device string
	FSType string
	Flags  uintptr
	Data   string
}

func Mount(dev, path, fsType, data string, flags uintptr, opts ...func() error) (*MountPoint, error) {
	return nil, errUnsupported
}

func TryMount(device, path, data string, flags uintptr, opts ...func() error) (*MountPoint, error) {
	return nil, errUnsupported
}

func Unmount(path string, force, lazy bool) error {
	return errUnsupported
}

func SameFilesystem(path1, path2 string) (bool, error) {
	return false, errUnsupported
}

func FSFromBlock(n string) (string, uintptr, error) {
	return "", 0, errUnsupported
}
