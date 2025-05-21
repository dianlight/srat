package dbom

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/gobeam/stringy"
	"github.com/snapcore/snapd/osutil"
	"github.com/u-root/u-root/pkg/mount"
	"github.com/xorcare/pointer"
	"gitlab.com/tozd/go/errors"
	"gorm.io/gorm"
)

type MountPointPath struct {
	Path         string `gorm:"primarykey"`
	Type         string `gorm:"not null;default:null"`
	DeviceId     uint64
	Device       string
	FSType       string
	Flags        MounDataFlags `g_orm:"type:text"`
	Data         MounDataFlags `g_orm:"type:text"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
	IsInvalid    bool
	IsMounted    bool
	InvalidError *string
	Warnings     *string
}

func (u *MountPointPath) BeforeSave(tx *gorm.DB) (err error) {
	if u.Path == "" {
		return errors.Errorf("path cannot be empty")
	}
	u.Path = stringy.New(u.Path).SnakeCase().Get()
	u.Warnings = nil
	u.IsInvalid = false
	u.InvalidError = nil

	// check if u.Path exists and is a directory
	sstat := syscall.Stat_t{}
	err = syscall.Stat(u.Path, &sstat)
	if os.IsNotExist(err) {
		u.IsInvalid = true
		u.InvalidError = pointer.String(fmt.Sprintf("error: %#+v", err))
	} else if !strings.HasPrefix(u.Path, "/") {
		return errors.Errorf("path %s is not a valid mountpoint", u.Path)
	} else if err != nil {
		return errors.WithStack(err)
	}
	if u.DeviceId == 0 || u.DeviceId != sstat.Dev {
		u.DeviceId = sstat.Dev
	}
	if !u.IsInvalid {
		stat := syscall.Statfs_t{}
		err = syscall.Statfs(u.Path, &stat)
		if err != nil {
			return errors.WithStack(err)
		}
		if len(u.Flags) == 0 {
			u.Flags.Scan(stat.Flags)
		}
		if u.Device == "" {
			u.IsInvalid = true
			u.InvalidError = pointer.String("Unknown device source for " + u.Path)
			info, err := osutil.LoadMountInfo()
			if err != nil {
				return errors.WithStack(err)
			}
			for _, m := range info {

				if m.MountDir == u.Path {
					u.Device = m.MountSource
					//u.PrimaryPath = m.MountDir
					u.FSType = m.FsType
					//u.Data = m.
					u.IsInvalid = false
					u.InvalidError = nil
					break
				} else {
					same, _ := mount.SameFilesystem(u.Path, m.MountDir)
					if same {
						//u.PrimaryPath = m.MountDir
						u.Device = m.MountSource
						u.FSType = m.FsType
						//u.Data = m.
						u.IsInvalid = false
						u.InvalidError = nil
						u.Warnings = pointer.String("Mount point is not the same as the primary path")
						break
					}
				}
			}
		}
		if u.FSType == "" && u.Device != "" {
			fs, flags, err := mount.FSFromBlock(u.Device)
			if err != nil {
				u.IsInvalid = true
				u.InvalidError = pointer.String(fmt.Sprintf("error: %#+v", err))
			}
			fmt.Printf("Flags %+v\n", flags)
			u.FSType = fs
		}
	}
	return nil
}
