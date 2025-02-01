package service

import (
	"os"
	"strconv"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/snapcore/snapd/osutil"
	"github.com/u-root/u-root/pkg/mount"
	"github.com/ztrue/tracerr"
)

type VolumeServiceInterface interface {
	MountVolume(id uint) error
	UnmountVolume(id uint, force bool, lazy bool) error
}

type VolumeService struct {
}

func NewVolumeService() VolumeServiceInterface {
	return &VolumeService{}
}

func (ms *VolumeService) MountVolume(id uint) error {
	var dbom_mount_data dbom.MountPointPath
	err := dbom_mount_data.FromID(uint(id))
	if err != nil {
		return tracerr.Wrap(err)
	}

	if dbom_mount_data.Source == "" {
		return tracerr.New("Source path is empty")
	}

	if dbom_mount_data.Path == "" {
		return tracerr.New("Mount point path is empty")
	}

	ok, err := osutil.IsMounted(dbom_mount_data.Path)
	if err != nil {
		return tracerr.Wrap(err)
	}

	if dbom_mount_data.IsMounted && ok {
		return tracerr.New("Volume is already mounted")
	}

	orgPath := dbom_mount_data.Path
	for i := 1; ok; i++ {
		dbom_mount_data.Path = orgPath + "_(" + strconv.Itoa(i) + ")"
		ok, err = osutil.IsMounted(dbom_mount_data.Path)
		if err != nil {
			return tracerr.Wrap(err)
		}
	}

	flags, err := dbom_mount_data.Flags.Value()
	if err != nil {
		return tracerr.Wrap(err)
	}
	var mp *mount.MountPoint
	if dbom_mount_data.FSType == "" {
		mp, err = mount.TryMount(dbom_mount_data.Source, dbom_mount_data.Path, "" /*mount_data.Data*/, uintptr(flags.(int64)), func() error { return os.MkdirAll(dbom_mount_data.Path, 0o666) })
	} else {
		mp, err = mount.Mount(dbom_mount_data.Source, dbom_mount_data.Path, dbom_mount_data.FSType, "" /*mount_data.Data*/, uintptr(flags.(int64)), func() error { return os.MkdirAll(dbom_mount_data.Path, 0o666) })
	}
	if err != nil {
		return tracerr.Wrap(err)
	} else {
		var convm converter.MountToDbomImpl
		err = convm.MountToMountPointPath(mp, &dbom_mount_data)
		if err != nil {
			return tracerr.Wrap(err)
		}
		dbom_mount_data.IsMounted = true
		err = dbom_mount_data.Save()
		if err != nil {
			return tracerr.Wrap(err)
		}
	}
	return nil
}

func (ms *VolumeService) UnmountVolume(id uint, force bool, lazy bool) error {
	var dbom_mount_data dbom.MountPointPath
	err := dbom_mount_data.FromID(uint(id))
	if err != nil {
		return tracerr.Wrap(err)
	}
	err = mount.Unmount(dbom_mount_data.Path, force, lazy)
	if err != nil {
		return tracerr.Wrap(err)
	}
	dbom_mount_data.IsMounted = false
	err = dbom_mount_data.Save()
	if err != nil {
		return tracerr.Wrap(err)
	}
	return nil

}
