package dbom

import (
	"github.com/glebarez/sqlite"
	"gitlab.com/tozd/go/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
		//db, err = gorm.Open(gormlite.Open(dbpath), &gorm.Config{
		TranslateError:         true,
		SkipDefaultTransaction: true,
		Logger:                 logger.Default.LogMode(logger.Silent),
	})

	if err != nil {
		panic(errors.Errorf("failed to connect database %s", dbpath))
	}
	// Migrate the schema
	db.AutoMigrate(&MountPointPath{}, &ExportedShare{}, &SambaUser{}, &Property{})
	/*
	   result, _ := db.Debug().Migrator().ColumnTypes(&Property{})

	   	for _, v := range result {
	   		fmt.Printf("%+v\n", v)
	   	}
	*/
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
