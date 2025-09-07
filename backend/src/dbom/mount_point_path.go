package dbom

import (
	"regexp"
	"strings"
	"time"

	"gitlab.com/tozd/go/errors"
	"gorm.io/gorm"
)

type MountPointPath struct {
	Path               string `gorm:"primarykey"`
	Type               string `gorm:"not null;default:null"`
	DeviceId           string `gorm:"not null;default:null;index"` // Device ID (e.g., from /dev/disk/by-id/) associated with this mount point.
	FSType             string
	Flags              *MounDataFlags `gorm:"not null;default:''"`
	Data               *MounDataFlags `gorm:"not null;default:''"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          gorm.DeletedAt  `gorm:"index"`
	IsToMountAtStartup *bool           `gorm:"not null;default:false"` // If true, mount point should be mounted at startup.
	Shares             []ExportedShare `gorm:"foreignKey:MountPointDataPath;references:Path;"`
}

// invalidPathCharsRegex defines characters that are NOT allowed in a path.
// Allowed are: alphanumeric, forward slash, period, underscore, hyphen.
var invalidPathCharsRegex = regexp.MustCompile(`[^a-zA-Z0-9/._-]`)

func (u *MountPointPath) BeforeSave(tx *gorm.DB) (err error) {
	if u.Path == "" {
		return errors.Errorf("path cannot be empty")
	}

	// Validate path format
	// 1. Must start with a forward slash.
	if !strings.HasPrefix(u.Path, "/") {
		return errors.Errorf("path must start with '/': got '%s'", u.Path)
	}

	// 2. Cannot contain null characters.
	if strings.Contains(u.Path, "\x00") {
		return errors.Errorf("path cannot contain null characters: '%s'", u.Path)
	}

	// 3. Must only contain allowed characters.
	if invalidPathCharsRegex.MatchString(u.Path) {
		firstInvalidChar := invalidPathCharsRegex.FindString(u.Path)
		return errors.Errorf("path contains invalid characters (e.g., '%s'): '%s'. Allowed characters are alphanumeric, '/', '.', '_', '-'", firstInvalidChar, u.Path)
	}

	//u.Path = stringy.New(u.Path).SnakeCase().Get() // FIXME: Why snake case? Should we use kebab case or keep it as is?

	// check if u.Path exists and is a directory
	/*
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
				fs, flags, err := mount.FSFromBlock(u.Device) // FIXME: this is not a good way to get the filesystem type
				if err != nil {
					u.IsInvalid = true
					u.InvalidError = pointer.String(fmt.Sprintf("error: %#+v", err))
				}
				fmt.Printf("Flags %+v\n", flags)
				u.FSType = fs
			}
		}
	*/
	return nil
}
