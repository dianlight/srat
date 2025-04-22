package dbom

import (
	"context"

	"github.com/glebarez/sqlite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDB(v struct {
	fx.In
	lc     fx.Lifecycle
	dbPath string `name:"db_path"`
}) *gorm.DB {

	db, err := gorm.Open(sqlite.Open(v.dbPath), &gorm.Config{
		//db, err = gorm.Open(gormlite.Open(dbpath), &gorm.Config{
		TranslateError:         true,
		SkipDefaultTransaction: true,
		Logger:                 logger.Default.LogMode(logger.Silent),
	})

	if err != nil {
		panic(errors.Errorf("failed to connect database %s", v.dbPath))
	}
	// Migrate the schema
	db.AutoMigrate(&MountPointPath{}, &ExportedShare{}, &SambaUser{}, &Property{})

	v.lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			sqlDB, err := db.DB()
			if err != nil {
				panic(err)
			}
			// Close
			sqlDB.Close()
			return nil
		},
	})
	return db
}
