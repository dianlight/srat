package dbom

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// RcloneLinkMigrationSuite guards against model↔migration schema drift.
//
// The production table is created by goose migration 00019, NOT by GORM
// AutoMigrate (RcloneLink is not in db_config.go's AutoMigrate list). Unit
// suites that AutoMigrate the model therefore cannot catch a column-name
// mismatch: GORM happily created "o_auth_state" for OAuthState while the
// migration shipped "oauth_state", and every INSERT failed on real
// deployments with "no such column: o_auth_state" (issue #954 follow-up).
//
// This suite applies the real embedded migrations and round-trips the model
// through them.
type RcloneLinkMigrationSuite struct {
	suite.Suite
	db *gorm.DB
}

func TestRcloneLinkMigrationSuite(t *testing.T) {
	suite.Run(t, new(RcloneLinkMigrationSuite))
}

func (suite *RcloneLinkMigrationSuite) SetupTest() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.Require().NoError(err)
	sqlDB, err := db.DB()
	suite.Require().NoError(err)

	// Replicate the production bootstrap order from db_config.go:
	// AutoMigrate of the core models FIRST, then goose migrations on top.
	// RcloneLink is deliberately absent from that list (its schema comes
	// exclusively from migration 00019).
	suite.Require().NoError(db.AutoMigrate(
		&MountPointPath{}, &ExportedShare{}, &SambaUser{}, &Property{},
		&Issue{}, &Problem{}, &HDIdleDevice{},
	))

	goose.SetBaseFS(migrations)
	suite.Require().NoError(goose.SetDialect("sqlite3"))
	suite.Require().NoError(goose.Up(sqlDB, "migrations"))
	suite.db = db
}

func (suite *RcloneLinkMigrationSuite) TestRoundTripAgainstGooseSchema() {
	link := RcloneLink{
		TargetKind:      "volume",
		TargetID:        "/mnt/Carola",
		Provider:        "dropbox",
		RemotePath:      "/srat/Carola",
		Status:          "unlinked",
		OAuthState:      "state-token",
		AutoSync:        false,
		ScheduleMinutes: 0,
	}
	suite.Require().NoError(suite.db.Save(&link).Error)

	var got RcloneLink
	err := suite.db.Where("target_kind = ? AND target_id = ?", "volume", "/mnt/Carola").First(&got).Error
	suite.Require().NoError(err)
	suite.Equal("state-token", got.OAuthState)
	suite.Equal("dropbox", got.Provider)

	// Update path (Save with existing PK) must work too.
	now := time.Now()
	got.Status = "authorized"
	got.OAuthState = ""
	got.LastSyncAt = &now
	suite.Require().NoError(suite.db.Save(&got).Error)

	var updated RcloneLink
	suite.Require().NoError(suite.db.First(&updated, "target_kind = ? AND target_id = ?", "volume", "/mnt/Carola").Error)
	suite.Equal("authorized", updated.Status)
	suite.Empty(updated.OAuthState)
	suite.Require().NotNil(updated.LastSyncAt)
}
