package dbom

import (

	//"gorm.io/driver/sqlite"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var (
	db *gorm.DB
)

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
	var err error
	db, err = gorm.Open(sqlite.Open(dbpath), &gorm.Config{
		TranslateError: true,
	})
	if err != nil {
		panic("failed to connect database")
	}
	// Migrate the schema
	db.AutoMigrate(&MountPointData{}, &ExportedShare{}, &SambaUser{}, &Property{})
}

func CloseDB() {
	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}

	// Close
	sqlDB.Close()
}

func GetDB() *gorm.DB {
	return db
}
