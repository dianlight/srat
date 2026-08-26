package dbom

import (
	"time"

	"gorm.io/gorm"
)

// RcloneLink represents a cloud-sync link between a local SRAT target
// (a mounted volume or the hassos-data partition) and an rclone remote
// (issue #954, lab feature). One row per target: a target can be linked to
// at most one remote.
type RcloneLink struct {
	// TargetKind discriminates the local side ("volume" or "hassos_data").
	TargetKind string `gorm:"primaryKey;type:text"`
	// TargetID is the volume path/root composite id, or "hassos-data".
	TargetID string `gorm:"primaryKey;type:text"`
	// Provider is the registered rclone driver name (e.g. "dropbox").
	Provider string `gorm:"type:text"`
	// RemotePath is the provider-specific path inside the remote,
	// e.g. "backups/srat".
	RemotePath string `gorm:"type:text"`
	// Status is one of: "unlinked", "authorized", "error".
	Status string `gorm:"type:text;default:'unlinked'"`
	// OAuthState holds the in-flight OAuth anti-CSRF token while Status is
	// "unlinked"; cleared once authorization completes.
	//
	// The column name is pinned: GORM's default naming would derive
	// "o_auth_state" from the acronym, while goose migration 00019 created
	// "oauth_state". Production schema comes from the migration (RcloneLink
	// is deliberately not in the AutoMigrate list), so the mismatch surfaced
	// only as "no such column: o_auth_state" on real deployments.
	OAuthState      string `gorm:"column:oauth_state;type:text"`
	LastSyncAt      *time.Time
	LastSyncResult  string `gorm:"type:text"` // "success" | "failure" | ""
	LastSyncMessage string `gorm:"type:text"`
	AutoSync        bool   `gorm:"default:false"`
	ScheduleMinutes int    `gorm:"default:0"` // 0 = manual only
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

// TableName pins the table name for goose migrations and GORM.
func (RcloneLink) TableName() string { return "rclone_links" }
