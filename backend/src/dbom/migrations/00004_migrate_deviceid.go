package migrations

//package main

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"

	"github.com/pressly/goose/v3"
)

var (
	readDirFunc      = os.ReadDir
	evalSymlinksFunc = filepath.EvalSymlinks
)

func init() {
	goose.AddMigrationNoTxContext(Up00004, Down00004)
}

func Up00004(ctx context.Context, db *sql.DB) error {
	paths, devices, err := getMountPointPatha(db)
	if err != nil {
		return err
	}
	for i, path := range paths {
		//fmt.Printf("Processing path: %s\n", path)
		device := devices[i]
		if device == "" {
			continue
		}
		// Find in /dev/disk/by-id/ the device that links to device
		// If not found update deleted_at to now
		// If found, use the UUID (the last part of the link) prefixed with by-id-
		// Example: /dev/disk/by-id/1234-5678 -> ../../sda1 -> /dev/sda1
		// We want to store by-id-1234-5678
		// Update mount_point_paths set device_id = 'by-id-1234-5678' where path = path

		deviceID := ""
		entries, err := readDirFunc("/dev/disk/by-id/")
		if err == nil {
			for _, entry := range entries {
				if entry.Type()&os.ModeSymlink != 0 {
					linkPath := filepath.Join("/dev/disk/by-id/", entry.Name())
					resolved, err := evalSymlinksFunc(linkPath)
					if err != nil {
						continue
					}
					//fmt.Printf("Resolved link: %s -> %s\n", linkPath, resolved)
					if resolved == "/dev/"+device {
						deviceID = "by-id-" + entry.Name()
						break
					}
				}
			}
		}
		//fmt.Printf("Device: %s, DeviceID: %s\n", device, deviceID)
		if deviceID != "" {
			query := "UPDATE mount_point_paths SET device_id = $1 WHERE path = $2"
			if _, err := db.ExecContext(ctx, query, deviceID, path); err != nil {
				return err
			}
		} else {
			query := "UPDATE mount_point_paths SET deleted_at = CURRENT_TIMESTAMP WHERE path = $1"
			if _, err := db.ExecContext(ctx, query, path); err != nil {
				return err
			}
		}
	}
	return nil
}

func getMountPointPatha(db *sql.DB) ([]string, []string, error) {
	var paths, devices []string
	rows, err := db.Query("SELECT path, device FROM mount_point_paths")
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, nil
		} else if strings.Contains(err.Error(), "no such column: device") {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var path, device string
		if err := rows.Scan(&path, &device); err != nil {
			return nil, nil, err
		}
		paths = append(paths, path)
		devices = append(devices, device)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return paths, devices, nil
}

func Down00004(ctx context.Context, db *sql.DB) error {
	return nil
}
