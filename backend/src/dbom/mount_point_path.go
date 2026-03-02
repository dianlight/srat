package dbom

import (
	"regexp"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"gitlab.com/tozd/go/errors"
	"gorm.io/gorm"
)

type MountPointPath struct {
	Path               string  `gorm:"primarykey"`
	Root               *string `gorm:"primarykey;default:'/'"`
	Type               string  `gorm:"not null;default:null"`
	DeviceId           string  `gorm:"not null;default:null;index"` // Device ID (e.g., from /dev/disk/by-id/) associated with this mount point.
	FSType             string
	Flags              *MounDataFlags `gorm:"not null;default:''"`
	Data               *MounDataFlags `gorm:"not null;default:''"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          gorm.DeletedAt `gorm:"index"`
	IsToMountAtStartup *bool          `gorm:"not null;default:false"` // If true, mount point should be mounted at startup.
	ExportedShare      *ExportedShare `gorm:"foreignKey:MountPointDataPath,MountPointDataRoot;references:Path,Root"`
}

// invalidPathCharsRegex defines characters that are NOT allowed in a path.
// Allowed are: alphanumeric, forward slash, period, underscore, hyphen.
var invalidPathCharsRegex = regexp.MustCompile(`[^a-zA-Z0-9/._-]`)

func (u *MountPointPath) BeforeSave(tx *gorm.DB) (err error) {
	if u.Path == "" {
		return errors.Errorf("path cannot be empty (%s)", spew.Sdump(u))
	}

	// Validate path format
	// 1. Must start with a forward slash.
	if !strings.HasPrefix(u.Path, "/") {
		return errors.Errorf("path must start with '/': got '%s' (%s)", u.Path, spew.Sdump(u))
	}

	// 2. Cannot contain null characters.
	if strings.Contains(u.Path, "\x00") {
		return errors.Errorf("path cannot contain null characters: '%s' (%s)", u.Path, spew.Sdump(u))
	}

	// 3. Must only contain allowed characters.
	if invalidPathCharsRegex.MatchString(u.Path) {
		firstInvalidChar := invalidPathCharsRegex.FindString(u.Path)
		return errors.Errorf("path contains invalid characters (e.g., '%s'): '%s'. Allowed characters are alphanumeric, '/', '.', '_', '-'. Full object: %s", firstInvalidChar, u.Path, spew.Sdump(u))
	}

	return nil
}
