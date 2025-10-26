package dbom

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"strings"

	_ "github.com/dianlight/srat/dbom/migrations"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/tlog"
	"github.com/glebarez/sqlite"
	"github.com/pressly/goose/v3"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

//go:embed migrations/*.sql
var migrations embed.FS

// checkFileSystemPermissions checks filesystem-level issues
func checkFileSystemPermissions(dbPath string) errors.E {
	// Extract the actual file path (remove query parameters)
	filePath := strings.Split(dbPath, "?")[0]

	if strings.Contains(dbPath, ":memory:") {
		// In-memory database, no file to check
		return nil
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		slog.Info("Database file does not exist, will be created", "path", filePath)
		// check if path is writable
		baseDir := filePath
		if !strings.HasSuffix(baseDir, "/") {
			baseDir = baseDir[:strings.LastIndex(baseDir, "/")]
		}
		if baseDir == "" {
			baseDir = "."
		}
		testFile := baseDir + "/.db_write_test"
		f, err := os.Create(testFile)
		if err != nil {
			return errors.Errorf("Database directory %s is not writable %w",
				baseDir, err)
		}
		f.Close()
		os.Remove(testFile)
		return nil
	}

	// Check file permissions
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		//		slog.Error("Failed to get file info", "error", err, "path", filePath)
		return errors.WithStack(err)
	}

	mode := fileInfo.Mode()
	slog.Debug("Database file permissions",
		"path", filePath,
		"mode", mode.String(),
		"writable", mode&0200 != 0)

	// Check if file is writable
	if mode&0200 == 0 {
		return errors.Errorf("Database file %s is not writable mode: %s",
			filePath,
			mode.String())
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
			return errors.Errorf("Database directory %s is not writable mode:%s",
				dir,
				dirMode.String())
		}
	} else {
		return errors.WithStack(err)
	}
	return nil
}

func NewDB(lc fx.Lifecycle, v struct {
	fx.In
	ApiCtx *dto.ContextState
}) *gorm.DB {

	// Check filesystem permissions before attempting to open database
	errE := checkFileSystemPermissions(v.ApiCtx.DatabasePath)
	if errE != nil {
		tlog.Fatal("Filesystem permissions check failed", "error", errE, "path", v.ApiCtx.DatabasePath)
		return nil
	}

	// Ensure a robust SQLite DSN with sane defaults for concurrency
	// - cache=shared to allow sharing the cache between connections
	// - _pragma=foreign_keys(1) to enforce FKs
	// - _pragma=journal_mode(WAL) to allow readers during writes
	// - _pragma=busy_timeout(5000) to wait for up to 5s instead of returning SQLITE_BUSY
	// - _pragma=synchronous(NORMAL) a common balance when using WAL
	dsn := v.ApiCtx.DatabasePath
	if !strings.Contains(dsn, "?") {
		dsn += "?cache=shared"
	} else if !strings.Contains(dsn, "cache=shared") {
		dsn += "&cache=shared"
	}
	// helper to append pragma only if missing
	addPragma := func(s, pragma string) string {
		if strings.Contains(strings.ToLower(s), strings.ToLower(pragma)) {
			return s
		}
		// use & separator since we ensured there is already a ?
		return s + "&_pragma=" + pragma
	}
	dsn = addPragma(dsn, "foreign_keys(1)")
	dsn = addPragma(dsn, "journal_mode(WAL)")
	dsn = addPragma(dsn, "busy_timeout(5000)")
	dsn = addPragma(dsn, "synchronous(NORMAL)")
	dsn = addPragma(dsn, "cell_size_check(1)")
	dsn = addPragma(dsn, "optimize")

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		TranslateError:         true,
		SkipDefaultTransaction: true,
		Logger:                 logger.Default.LogMode(logger.Silent),
	})

	if errE = errors.WithStack(err); errE != nil {
		slog.Error("Failed to connect to database", "error", errE, "path", v.ApiCtx.DatabasePath)

		// Check if it's a readonly issue and try to resolve
		if strings.Contains(strings.ToLower(err.Error()), "readonly") {
			slog.Error("Database connection failed due to readonly issue, attempting to create fresh database")
			// Remove the existing database file and try again
			return replaceDatabase(lc, v)
		}
		panic(errors.Errorf("failed to connect database %s", v.ApiCtx.DatabasePath))
	}

	errE = checkDBIntegrity(db)
	if errE != nil {
		slog.Error("Failed to check database integrity", "error", errE, "path", v.ApiCtx.DatabasePath)
		return replaceDatabase(lc, v)
	}

	// Apply conservative connection pool settings for SQLite
	sqlDB, dbErr := db.DB()
	if dbErr == nil {
		// Single connection avoids many SQLITE_BUSY scenarios in embedded SQLite
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
	} else {
		slog.Warn("Failed to get SQL DB for pool tuning", "error", dbErr)
	}

	// Migrate the schema
	err = db.AutoMigrate(&MountPointPath{}, &ExportedShare{}, &SambaUser{}, &Property{}, &Issue{}, &HDIdleDevice{})
	if errE = errors.WithStack(err); errE != nil {
		slog.Error("Failed to migrate database", "error", errE, "path", v.ApiCtx.DatabasePath)
		return replaceDatabase(lc, v)
	}

	// GooseDBMigration
	goose.SetBaseFS(migrations)
	goose.WithSlog(slog.Default())

	if err := goose.SetDialect("sqlite3"); err != nil {
		panic(err)
	}

	if err := goose.Up(sqlDB, "migrations"); err != nil {
		slog.Error("Failed to apply migrations", "error", err, "path", v.ApiCtx.DatabasePath)
		panic(err)
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return nil
		},
		OnStop: func(ctx context.Context) error {
			sqlDB, err := db.DB()
			if errE = errors.WithStack(err); errE != nil {
				slog.Error("Failed to get SQL DB on shutdown", "error", errE, "path", v.ApiCtx.DatabasePath)
			} else {
				sqlDB.Close()
			}
			return nil
		},
	})

	return db
}

func replaceDatabase(lc fx.Lifecycle, v struct {
	fx.In
	ApiCtx *dto.ContextState
}) *gorm.DB {
	filePath := strings.Split(v.ApiCtx.DatabasePath, "?")[0]
	if removeErr := os.Remove(filePath); removeErr != nil {
		slog.Error("Failed to remove readonly database file", "error", removeErr, "path", filePath)
	} else {
		slog.Info("Removed readonly database file, attempting to recreate", "path", filePath)
		return NewDB(lc, v)
	}
	return nil
}

func checkDBIntegrity(db *gorm.DB) errors.E {
	sqlDB, dbErr := db.DB()
	if errE := errors.WithStack(dbErr); errE != nil {
		return errors.Errorf("failed to get SQL DB for PRAGMA checks: %w", errE)
	}
	// Run integrity_check
	rows, icErr := sqlDB.Query("PRAGMA integrity_check;")
	if errE := errors.WithStack(icErr); errE != nil {
		slog.Warn("Failed to run integrity_check pragma", "error", errE)
	} else {
		defer rows.Close()
		index := 0
		problems := make([]string, 0, 10)
		for rows.Next() {
			index++
			var result string
			if scanErr := rows.Scan(&result); scanErr == nil {
				if index == 1 && result == "ok" {
					break
				}
				slog.Info("PRAGMA integrity_check result", "result", result)
				problems = append(problems, result)
			}
		}
		if len(problems) != 0 {
			return errors.Errorf("database integrity check failed: %v", problems)
		}
	}
	// Run foreign_key_check
	rows, fkErr := sqlDB.Query("PRAGMA foreign_key_check;")
	if errE := errors.WithStack(fkErr); errE != nil {
		slog.Warn("Failed to run foreign_key_check pragma", "error", errE)
	} else {
		defer rows.Close()
		index := 0
		problems := make([]string, 0, 20)
		for rows.Next() {
			index++
			var table string
			var rowid, parent, fkid interface{}
			if scanErr := rows.Scan(&table, &rowid, &parent, &fkid); scanErr == nil {
				slog.Info("PRAGMA foreign_key_check result", "table", table, "rowid", rowid, "parent", parent, "fkid", fkid)
				problems = append(problems, fmt.Sprintf("Table: %s, RowID: %v, Parent: %v, FkID: %v", table, rowid, parent, fkid))
			}
		}
		if len(problems) > 0 {
			return errors.Errorf("database foreign key check failed: %v", problems)
		}
	}
	return nil
}
