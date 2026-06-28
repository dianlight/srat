#!/bin/sh
# Regenerate Darwin vendor stubs after `go mod vendor` deletes them.
# These stubs allow the SRAT backend to compile and run tests on macOS.
# Canonical stubs live in backend/src/internal/darwinstubs/.
# Called by the `patch` mise task after `go -C src mod vendor`.
set -e

SRCDIR="${1:-src}"

mkdir -p "$SRCDIR/vendor/github.com/u-root/u-root/pkg/mount/loop"
mkdir -p "$SRCDIR/vendor/github.com/prometheus/procfs/sysfs"

cat >"$SRCDIR/vendor/github.com/u-root/u-root/pkg/mount/mount_darwin.go" <<'GOEOF'
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
GOEOF

cat >"$SRCDIR/vendor/github.com/u-root/u-root/pkg/mount/loop/loop_darwin.go" <<'GOEOF'
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
GOEOF

cat >"$SRCDIR/vendor/github.com/prometheus/procfs/sysfs/fs_darwin.go" <<'GOEOF'
//go:build darwin

package sysfs

// FS represents a path on the sysfs filesystem.
type FS struct{}

// NewFS returns a new FS mounted under the given mountPoint.
func NewFS(mountPoint string) (FS, error) {
	return FS{}, nil
}

// NetClassIface contains information about a network interface.
type NetClassIface struct {
	Name             string
	AddrAssignType   *int64
	AddrLen          *int64
	Address          string
	Broadcast        string
	Carrier          *int64
	CarrierChanges   *int64
	CarrierUpCount   *int64
	CarrierDownCount *int64
	DevID            *int64
	Dormant          *int64
	Duplex           string
	Flags            *int64
	IfAlias          string
	IfIndex          *int64
	IfLink           *int64
	LinkMode         *int64
	MTU              *int64
	NameAssignType   *int64
	NetDevGroup      *int64
	OperState        string
	PhysPortID       string
	PhysPortName     string
	PhysSwitchID     string
	Speed            *int64
	TxQueueLen       *int64
	Type             *int64
}

// NetClass is a collection of info for every interface.
type NetClass map[string]NetClassIface

// NetClassByIface returns network interface stats for the given interface.
func (fs FS) NetClassByIface(devicePath string) (*NetClassIface, error) {
	return nil, nil
}

// NetClassDevices returns a list of network device names.
func (fs FS) NetClassDevices() ([]string, error) {
	return nil, nil
}

// NetClass returns network interface stats for all interfaces.
func (fs FS) NetClass() (NetClass, error) {
	return nil, nil
}
GOEOF

echo "Darwin vendor stubs regenerated."
