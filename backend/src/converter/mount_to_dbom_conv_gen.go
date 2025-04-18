// Code generated by github.com/jmattheis/goverter, DO NOT EDIT.
//go:build !goverter

package converter

import (
	dbom "github.com/dianlight/srat/dbom"
	mount "github.com/u-root/u-root/pkg/mount"
)

type MountToDbomImpl struct{}

func (c *MountToDbomImpl) MountToMountPointPath(source *mount.MountPoint, target *dbom.MountPointPath) error {
	if source != nil {
		if source.Path != "" {
			target.Path = source.Path
		}
		if source.Device != "" {
			xstring, err := removeDevPrefix(source.Device)
			if err != nil {
				return err
			}
			target.Device = xstring
		}
		if source.FSType != "" {
			target.FSType = source.FSType
		}
		if source.Flags != 0 {
			dbomMounDataFlags, err := uintptrToMounDataFlags(source.Flags)
			if err != nil {
				return err
			}
			target.Flags = dbomMounDataFlags
		}
	}
	return nil
}
