//go:build linux

package mount

import uremount "github.com/u-root/u-root/pkg/mount"

type Mounter = uremount.Mounter
type MountPoint = uremount.MountPoint

func Mount(dev, path, fsType, data string, flags uintptr, opts ...func() error) (*MountPoint, error) {
	return uremount.Mount(dev, path, fsType, data, flags, opts...)
}

func TryMount(device, path, data string, flags uintptr, opts ...func() error) (*MountPoint, error) {
	return uremount.TryMount(device, path, data, flags, opts...)
}

func Unmount(path string, force, lazy bool) error {
	return uremount.Unmount(path, force, lazy)
}

func SameFilesystem(path1, path2 string) (bool, error) {
	return uremount.SameFilesystem(path1, path2)
}

func FSFromBlock(n string) (string, uintptr, error) {
	return uremount.FSFromBlock(n)
}
