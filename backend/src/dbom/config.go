package dbom

import (

	//"gorm.io/driver/sqlite"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

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
	adb.AutoMigrate(&MountPointData{}, &ExportedShare{}, &SambaUser{})
	db = adb
}
