package config

import (
	"time"

	//"gorm.io/driver/sqlite"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

type MountPointData struct {
	CreatedAt time.Time      `json:"-"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Path      string         `json:"path"`
	Label     string         `json:"label"`
	Name      string         `json:"name" gorm:"primarykey"`
	FSType    string         `json:"fstype"`
	Flags     MounDataFlags  `json:"flags" gorm:"type:mount_data_flags"`
	Data      string         `json:"data,omitempty"`
}

// initDB initializes the database connection and performs schema migration.
//
// It opens a connection to the SQLite database specified by dbpath using GORM,
// and automatically migrates the MountPointData schema.
//
// Parameters:
//   - dbpath: A string representing the path to the SQLite database file.
//
// The function panics if it fails to connect to the database.
func InitDB(dbpath string) {
	adb, err := gorm.Open(sqlite.Open(dbpath), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	adb.AutoMigrate(&MountPointData{})
	db = adb
}

// ListMountPointData retrieves the list of volumes from the database.
func ListMountPointData() ([]MountPointData, error) {
	var mountPoints []MountPointData
	err := db.Find(&mountPoints).Error
	return mountPoints, err
}

// SaveMountPointData saves a new mount point data entry to the database.
//
// Parameters:
//   - mp: A MountPointData struct containing the mount point information to be saved.
//
// Returns:
//   - error: An error if the save operation fails, or nil if successful.
func SaveMountPointData(mp MountPointData) error {
	return db.Create(&mp).Error
}
