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

// ValidateMountPointPath checks a mount point path against the same rules
// enforced on save, so callers can reject invalid paths before any OS work.
func ValidateMountPointPath(path string) error {
	if path == "" {
		return errors.Errorf("path cannot be empty")
	}

	// Validate path format
	// 1. Must start with a forward slash.
	if !strings.HasPrefix(path, "/") {
		return errors.Errorf("path must start with '/': got '%s'", path)
	}

	// 2. Cannot contain null characters.
	if strings.Contains(path, "\x00") {
		return errors.Errorf("path cannot contain null characters: '%s'", path)
	}

	// 3. Must only contain allowed characters.
	if invalidPathCharsRegex.MatchString(path) {
		firstInvalidChar := invalidPathCharsRegex.FindString(path)
		return errors.Errorf("path contains invalid characters (e.g., '%s'): '%s'. Allowed characters are alphanumeric, '/', '.', '_', '-'.", firstInvalidChar, path)
	}

	return nil
}

// SuggestSanitizedMountPointPath returns a best-effort valid path by
// replacing every disallowed character with an underscore.
func SuggestSanitizedMountPointPath(path string) string {
	return invalidPathCharsRegex.ReplaceAllString(path, "_")
}

func (u *MountPointPath) BeforeSave(tx *gorm.DB) (err error) {
	if err := ValidateMountPointPath(u.Path); err != nil {
		return errors.Errorf("%s (%s)", err.Error(), spew.Sdump(u))
	}

	return nil
}
