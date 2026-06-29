//go:build linux

package sysfs

import (
	uresysfs "github.com/prometheus/procfs/sysfs"
)

type FS = uresysfs.FS
type NetClassIface = uresysfs.NetClassIface
type NetClass = uresysfs.NetClass

func NewFS(mountPoint string) (FS, error) {
	return uresysfs.NewFS(mountPoint)
}
