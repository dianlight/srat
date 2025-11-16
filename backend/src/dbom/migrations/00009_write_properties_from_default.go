package migrations

//package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"reflect"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/templates"
	"github.com/pressly/goose/v3"
	"github.com/thoas/go-funk"
)

func init() {
	goose.AddMigrationNoTxContext(Up00009, Down00009)
}

func Up00009(ctx context.Context, db *sql.DB) error {

	if os.Getenv("SRAT_MOCK") == "true" {
		return nil
	}

	buffer, err := templates.Default_Config_content.ReadFile("default_config.json")
	if err != nil {
		log.Fatalf("Cant read default config file %#+v", err)
	}
	var config config.Config
	err = config.LoadConfigBuffer(buffer) // Assign to existing err
	if err != nil {
		log.Fatalf("Cant load default config from buffer %#+v", err)
	}

	queryUpdate := "INSERT OR IGNORE INTO properties (key,value,internal) VALUES (?, ?, ?)"

	vsource := reflect.Indirect(reflect.ValueOf(config))
	for i := 0; i < vsource.NumField(); i++ {
		key := vsource.Type().Field(i).Name
		if funk.Contains([]string{"Shares", "OtherUsers", "ACL", "Medialibrary"}, key) {
			continue
		}
		newvalue := reflect.ValueOf(config).FieldByName(key)
		if newvalue.IsZero() {
			continue
		}
		// use json to serialize value
		sqlvalue, err := json.Marshal(newvalue.Interface())
		if err != nil {
			slog.Warn("Error marshaling default property", "key", key, "error", err)
			continue
		}

		if r, err := db.ExecContext(ctx, queryUpdate, key, sqlvalue, 0); err != nil {
			return err
		} else {
			affected, _ := r.RowsAffected()
			if affected > 0 {
				slog.Info("Writing default property", "key", key, "value", string(sqlvalue))
			}
			//log.Printf("Inserted rows: %d", affected)
		}
	}

	return nil
}

func Down00009(ctx context.Context, db *sql.DB) error {
	return nil
}
