package dbom

import (
	"context"
	"fmt"
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

// checkDatabaseReadonly checks if the database is readonly and logs the cause
func checkDatabaseReadonly(db *gorm.DB, dbPath string) (bool, error) {
	// Get the underlying SQL database
	sqlDB, err := db.DB()
	if err != nil {
		return false, fmt.Errorf("failed to get underlying SQL database: %w", err)
	}

	// Test write capability by trying to create a temporary table
	testTableSQL := "CREATE TEMP TABLE _readonly_test (id INTEGER)"
	_, err = sqlDB.Exec(testTableSQL)
	if err != nil {
		// Check if it's a readonly error
		if strings.Contains(strings.ToLower(err.Error()), "readonly") {
			slog.Error("Database is readonly - cannot create temporary table",
				"error", err,
				"path", dbPath)
			return true, err
		}
		// Other error, but not necessarily readonly
		slog.Warn("Failed to create test table, but might not be readonly issue",
			"error", err,
			"path", dbPath)
		return false, err
	}

	// Clean up the test table
	_, err = sqlDB.Exec("DROP TABLE _readonly_test")
	if err != nil {
		slog.Warn("Failed to clean up test table", "error", err)
	}

	return false, nil
}

// checkFileSystemPermissions checks filesystem-level issues
func checkFileSystemPermissions(dbPath string) {
	// Extract the actual file path (remove query parameters)
	filePath := strings.Split(dbPath, "?")[0]

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		slog.Info("Database file does not exist, will be created", "path", filePath)
		return
	}

	// Check file permissions
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		slog.Error("Failed to get file info", "error", err, "path", filePath)
		return
	}

	mode := fileInfo.Mode()
	slog.Debug("Database file permissions",
		"path", filePath,
		"mode", mode.String(),
		"writable", mode&0200 != 0)

	// Check if file is writable
	if mode&0200 == 0 {
		slog.Error("Database file is not writable",
			"path", filePath,
			"mode", mode.String())
	}

	// Check directory permissions
	dir := strings.TrimSuffix(filePath, "/"+fileInfo.Name())
	if dirInfo, err := os.Stat(dir); err == nil {
		dirMode := dirInfo.Mode()
		slog.Debug("Database directory permissions",
			"path", dir,
			"mode", dirMode.String(),
			"writable", dirMode&0200 != 0)

		if dirMode&0200 == 0 {
			slog.Error("Database directory is not writable",
				"path", dir,
				"mode", dirMode.String())
		}
	} else {
		slog.Error("Failed to check directory permissions", "error", err, "dir", dir)
	}
}

func NewDB(lc fx.Lifecycle, v struct {
	fx.In
	ApiCtx *dto.ContextState
}) *gorm.DB {

	// Check filesystem permissions before attempting to open database
	checkFileSystemPermissions(v.ApiCtx.DatabasePath)

	db, err := gorm.Open(sqlite.Open(v.ApiCtx.DatabasePath), &gorm.Config{
		//db, err = gorm.Open(gormlite.Open(dbpath), &gorm.Config{
		TranslateError:         true,
		SkipDefaultTransaction: true,
		Logger:                 logger.Default.LogMode(logger.Silent),
	})

	if err != nil {
		slog.Error("Failed to connect to database", "error", err, "path", v.ApiCtx.DatabasePath)

		// Check if it's a readonly issue and try to resolve
		if strings.Contains(strings.ToLower(err.Error()), "readonly") {
			slog.Error("Database connection failed due to readonly issue, attempting to create fresh database")
			checkFileSystemPermissions(v.ApiCtx.DatabasePath)

			// Remove the existing database file and try again
			filePath := strings.Split(v.ApiCtx.DatabasePath, "?")[0]
			if removeErr := os.Remove(filePath); removeErr != nil {
				slog.Error("Failed to remove readonly database file", "error", removeErr, "path", filePath)
			} else {
				slog.Info("Removed readonly database file, attempting to recreate", "path", filePath)
				return NewDB(lc, v) // Recursive call to create fresh DB
			}
		}

		panic(errors.Errorf("failed to connect database %s", v.ApiCtx.DatabasePath))
	}

	// Check if database is readonly after successful connection
	if readonly, readonlyErr := checkDatabaseReadonly(db, v.ApiCtx.DatabasePath); readonly {
		slog.Error("Database is readonly after connection", "error", readonlyErr, "path", v.ApiCtx.DatabasePath)
		slog.Warn("Closing readonly database and creating fresh instance")

		// Close the readonly database
		if sqlDB, dbErr := db.DB(); dbErr == nil {
			sqlDB.Close()
		}

		// Remove the readonly database file
		filePath := strings.Split(v.ApiCtx.DatabasePath, "?")[0]
		if removeErr := os.Remove(filePath); removeErr != nil {
			slog.Error("Failed to remove readonly database file", "error", removeErr, "path", filePath)
			panic(errors.Errorf("database is readonly and cannot be removed: %s", v.ApiCtx.DatabasePath))
		}

		slog.Info("Removed readonly database file, creating fresh database", "path", filePath)
		return NewDB(lc, v) // Recursive call to create fresh DB
	}

	// Migrate the schema
	err = db.AutoMigrate(&MountPointPath{}, &ExportedShare{}, &SambaUser{}, &Property{}, &Issue{})
	if err != nil {
		slog.Error("Failed to migrate database", "error", err, "path", v.ApiCtx.DatabasePath)

		// Check if migration failed due to readonly database
		if strings.Contains(strings.ToLower(err.Error()), "readonly") {
			slog.Error("Database migration failed due to readonly issue")
			checkFileSystemPermissions(v.ApiCtx.DatabasePath)
		}

		slog.Warn("Resetting Database to Default State")
		sqlDB, _ := db.DB()
		sqlDB.Close()

		filePath := strings.Split(v.ApiCtx.DatabasePath, "?")[0]
		if removeErr := os.Remove(filePath); removeErr != nil {
			slog.Error("Failed to remove database file during reset", "error", removeErr, "path", filePath)
		} else {
			slog.Info("Removed database file, creating fresh database", "path", filePath)
		}

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
