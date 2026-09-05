//go:build darwin

package loop

import (
	"fmt"

	"github.com/u-root/u-root/pkg/mount"
)

var errUnsupported = fmt.Errorf("loop device operations not supported on this platform")

// Loop represents a regular file exposed as a loop block device.
type Loop struct {
	Dev    string
	Source string
	FSType string
	Data   string
}

var _ mount.Mounter = &Loop{}

// New initializes a Loop struct.
func New(source, fstype string, data string) (*Loop, error) {
	return nil, errUnsupported
}

// DevName implements mount.Mounter.
func (l *Loop) DevName() string {
	return l.Dev
}

// Mount mounts the provided source file using the allocated loop device.
func (l *Loop) Mount(path string, flags uintptr, opts ...func() error) (*mount.MountPoint, error) {
	return nil, errUnsupported
}

// Free frees the loop device.
func (l *Loop) Free() error {
	return errUnsupported
}

// FindDevice finds an unused loop device.
func FindDevice() (string, error) {
	return "", errUnsupported
}

// SetFile associates loop device with regular file.
func SetFile(devicename, filename string) error {
	return errUnsupported
}

// ClearFile clears the fd association of the loop device.
func ClearFile(devicename string) error {
	return errUnsupported
}
