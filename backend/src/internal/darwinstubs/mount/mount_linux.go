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

// FSFromBlock determines the file system type of a block device. It uses
// u-root's magic probe first and falls back to the boot sector OEM ID for
// NTFS and exFAT, which u-root does not detect (srat#1513).
func FSFromBlock(n string) (string, uintptr, error) {
	fs, flags, err := uremount.FSFromBlock(n)
	if err == nil {
		return fs, flags, nil
	}
	if bootFs, bootErr := probeBootSectorFS(n); bootErr == nil {
		return bootFs, 0, nil
	}
	return "", 0, err
}
