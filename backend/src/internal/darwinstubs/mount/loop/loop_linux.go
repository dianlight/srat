//go:build linux

package loop

import (
	"github.com/dianlight/srat/internal/darwinstubs/mount"
	ureloop "github.com/u-root/u-root/pkg/mount/loop"
)

type Loop = ureloop.Loop

func New(source, fstype string, data string) (*Loop, error) {
	return ureloop.New(source, fstype, data)
}

func FindDevice() (string, error) {
	return ureloop.FindDevice()
}

func SetFile(devicename, filename string) error {
	return ureloop.SetFile(devicename, filename)
}

func ClearFile(devicename string) error {
	return ureloop.ClearFile(devicename)
}

var _ mount.Mounter = &Loop{}
