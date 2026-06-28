//go:build darwin

package loop

import (
	"fmt"

	"github.com/dianlight/srat/internal/darwinstubs/mount"
)

var errUnsupported = fmt.Errorf("loop device operations not supported on this platform")

type Loop struct {
	Dev    string
	Source string
	FSType string
	Data   string
}

var _ mount.Mounter = &Loop{}

func New(source, fstype string, data string) (*Loop, error) {
	return nil, errUnsupported
}

func (l *Loop) DevName() string {
	return l.Dev
}

func (l *Loop) Mount(path string, flags uintptr, opts ...func() error) (*mount.MountPoint, error) {
	return nil, errUnsupported
}

func (l *Loop) Free() error {
	return errUnsupported
}

func FindDevice() (string, error) {
	return "", errUnsupported
}

func SetFile(devicename, filename string) error {
	return errUnsupported
}

func ClearFile(devicename string) error {
	return errUnsupported
}
