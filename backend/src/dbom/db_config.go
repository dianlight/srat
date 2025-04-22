package dbom

import (
	"context"

	"github.com/glebarez/sqlite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDB(lc fx.Lifecycle, v struct {
	fx.In

	DbPath string `name:"db_path"`
}) *gorm.DB {

	db, err := gorm.Open(sqlite.Open(v.DbPath), &gorm.Config{
		//db, err = gorm.Open(gormlite.Open(dbpath), &gorm.Config{
		TranslateError:         true,
		SkipDefaultTransaction: true,
		Logger:                 logger.Default.LogMode(logger.Silent),
	})

	if err != nil {
		panic(errors.Errorf("failed to connect database %s", v.DbPath))
	}
	// Migrate the schema
	db.AutoMigrate(&MountPointPath{}, &ExportedShare{}, &SambaUser{}, &Property{})

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return nil
		},
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
