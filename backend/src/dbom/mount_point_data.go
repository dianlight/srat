package dbom

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/snapcore/snapd/osutil"
	"github.com/u-root/u-root/pkg/mount"
	"github.com/xorcare/pointer"
	"github.com/ztrue/tracerr"
	"gorm.io/gorm"
)

type MountPointData struct {
	ID          uint `gorm:"primarykey"`
	DeviceId    uint64
	Source      string
	Path        string `gorm:"uniqueIndex"`
	PrimaryPath string
	FSType      string
	Flags       dto.MounDataFlags `gorm:"type:mount_data_flags"`
	//Data         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
	Invalid      bool
	InvalidError *string
	Warnings     *string
}

// BeforeSave is a GORM callback function that sets the DefaultPath to the Path
// if it is currently empty. This function is intended to be used with GORM's
// BeforeSave hook to ensure that the DefaultPath is always populated.
//
// Parameters:
// - tx: A pointer to a gorm.DB instance representing the database transaction.
//
// Return Value:
//   - err: An error value that will be returned by GORM if the callback function
//     returns an error. If no error occurs, this value will be nil.

func (u *MountPointData) BeforeSave(tx *gorm.DB) error {
	if u.Path == "" {
		return tracerr.Errorf("path cannot be empty")
	}
	// check if u.Path exists and is a directory
	sstat := syscall.Stat_t{}
	err := syscall.Stat(u.Path, &sstat)
	if os.IsNotExist(err) {
		u.Invalid = true
		u.InvalidError = pointer.String(tracerr.Sprint(err))
	} else if !strings.HasPrefix(u.Path, "/") {
		return tracerr.Errorf("path %s is not a valid mountpoint", u.Path)
	} else if err != nil {
		return tracerr.Wrap(err)
	}
	if u.DeviceId == 0 || u.DeviceId != sstat.Dev {
		u.DeviceId = sstat.Dev
	}
	if !u.Invalid {
		stat := syscall.Statfs_t{}
		err = syscall.Statfs(u.Path, &stat)
		if err != nil {
			return tracerr.Wrap(err)
		}
		if len(u.Flags) == 0 {
			u.Flags.Scan(stat.Flags)
		}
		if u.Source == "" {
			u.Invalid = true
			u.InvalidError = pointer.String("Unknown device source for " + u.Path)
			info, err := osutil.LoadMountInfo()
			if err != nil {
				return tracerr.Wrap(err)
			}
			for _, m := range info {

				if m.MountDir == u.Path {
					u.Source = m.MountSource
					u.PrimaryPath = m.MountDir
					u.FSType = m.FsType
					//u.Data = m.
					u.Invalid = false
					u.InvalidError = nil
					break
				} else {
					same, _ := mount.SameFilesystem(u.Path, m.MountDir)
					if same {
						u.PrimaryPath = m.MountDir
						u.Source = m.MountSource
						u.FSType = m.FsType
						//u.Data = m.
						u.Invalid = false
						u.InvalidError = nil
						u.Warnings = pointer.String("Mount point is not the same as the primary path")
						break
					}
				}
			}
		}
		if u.FSType == "" && u.Source != "" {
			fs, flags, err := mount.FSFromBlock(u.Source)
			if err != nil {
				u.Invalid = true
				u.InvalidError = pointer.String(tracerr.Sprint(err))
			}
			fmt.Printf("Flags %+v\n", flags)
			u.FSType = fs
		}
	}
	return nil
}

/*
func (u *MountPointData) AfterFind(tx *gorm.DB) (err error) {
	// Validate teh mount data flags

	return
  }
*/

// All retrieves all MountPointData entries from the database.
//
// This method uses the global 'db' variable, which should be a properly
// initialized GORM database connection.
//
// Returns:
//   - []MountPointData: A slice containing all MountPointData entries found in the database.
//   - error: An error if the retrieval operation fails, or nil if successful.
//     Possible errors include database connection issues or other database-related errors.
func (_ MountPointData) All() ([]MountPointData, error) {
	var mountPoints []MountPointData
	err := db.Find(&mountPoints).Error
	return mountPoints, err
}

// Save persists the current MountPointData instance to the database.
// If the instance already exists in the database, it will be updated;
// otherwise, a new record will be created.
//
// This method uses the global 'db' variable, which should be a properly
// initialized GORM database connection.
//
// Returns:
//   - error: An error if the save operation fails, or nil if successful.
//     Possible errors include database connection issues, constraint violations,
//     or other database-related errors.
func (mp *MountPointData) Save() error {
	return db.Save(mp).Error
}

func (mp *MountPointData) FromPath(path string) error {
	if path == "" {
		return tracerr.Errorf("path cannot be empty")
	}
	//log.Printf("FromName \n%s \n%v \n%v", name, db, &mp)
	return db.Limit(1).Find(&mp, "path = ?", path).Error
}
