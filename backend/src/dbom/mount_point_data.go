package dbom

import (
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/ztrue/tracerr"
	"gorm.io/gorm"
)

type MountPointData struct {
	BlockDeviceId uint64 `gorm:"primaryKey;autoIncrement:false"`
	Name          string `gorm:"index,unique"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
	Path          string
	DefaultPath   string
	FSType        string
	Flags         dto.MounDataFlags `gorm:"type:mount_data_flags"`
	Data          string
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
func (u *MountPointData) BeforeSave(tx *gorm.DB) (err error) {
	if u.DefaultPath == "" {
		u.DefaultPath = u.Path
	}
	return
}

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

// FromName retrieves a MountPointData entry from the database by its name.
// It populates the receiver MountPointData struct with the data from the database.
//
// Parameters:
//   - name: A string representing the name of the mount point to retrieve.
//
// Returns:
//   - error: An error if the retrieval operation fails, or nil if successful.
//     Possible errors include database connection issues or if no record is found.
func (mp *MountPointData) FromName(name string) error {
	if name == "" {
		return tracerr.Errorf("name cannot be empty")
	}
	//log.Printf("FromName \n%s \n%v \n%v", name, db, &mp)
	return db.Limit(1).Find(&mp, "name = ?", name).Error
}
