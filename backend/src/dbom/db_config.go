package dbom

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/dianlight/srat/dto"
	"github.com/glebarez/sqlite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDB(lc fx.Lifecycle, v struct {
	fx.In
	ApiCtx *dto.ContextState
}) *gorm.DB {

	db, err := gorm.Open(sqlite.Open(v.ApiCtx.DatabasePath), &gorm.Config{
		//db, err = gorm.Open(gormlite.Open(dbpath), &gorm.Config{
		TranslateError:         true,
		SkipDefaultTransaction: true,
		Logger:                 logger.Default.LogMode(logger.Silent),
	})

	if err != nil {
		panic(errors.Errorf("failed to connect database %s", v.ApiCtx.DatabasePath))
	}
	// Migrate the schema
	err = db.AutoMigrate(&MountPointPath{}, &ExportedShare{}, &SambaUser{}, &Property{})
	if err != nil {
		slog.Error("failed to migrate database", "error", err, "path", v.ApiCtx.DatabasePath)
		slog.Warn("Resetting Database to Default State")
		sqlDB, _ := db.DB()
		sqlDB.Close()
		os.Remove(strings.Split(v.ApiCtx.DatabasePath, "?")[0])
		return NewDB(lc, v)
	}

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
