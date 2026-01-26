package migrations

//package main

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(Up00009, Down00009)
}

func Up00009(ctx context.Context, db *sql.DB) error {
	/*

		if os.Getenv("SRAT_MOCK") == "true" {
			return nil
		}

		setting := dto.Settings{}
		defaults.Set(&setting)

		conv := converter.DtoToDbomConverterImpl{}
		properties := dbom.Properties{}
		if err := conv.SettingsToProperties(setting, &properties); err != nil {
			slog.ErrorContext(ctx, "Error converting default settings to properties", "error", err)
			return err
		}

		queryUpdate := "INSERT OR IGNORE INTO properties (key,value) VALUES (?, ?)"

		for key, property := range properties {
			sqlvalue, err := json.Marshal(property.Value)
			if err != nil {
				slog.WarnContext(ctx, "Error marshaling default property", "key", key, "error", err)
				continue
			}
			if r, err := db.ExecContext(ctx, queryUpdate, key, sqlvalue); err != nil {
				return err
			} else {
				affected, _ := r.RowsAffected()
				if affected > 0 {
					slog.InfoContext(ctx, "Writing default property", "key", key, "value", string(sqlvalue))
				}
				//log.Printf("Inserted rows: %d", affected)
			}
		}
	*/
	return nil
}

func Down00009(ctx context.Context, db *sql.DB) error {
	return nil
}
