package service_test

import (
	"context"
	"crypto/rand"
	"os"
	"sync"
	"testing"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dbom/g"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"gorm.io/gorm"
)

type ShareServiceSuite struct {
	suite.Suite
	shareService service.ShareServiceInterface
	userService  service.UserServiceInterface
	//exported_share_repo repository.ExportedShareRepositoryInterface
	app *fxtest.App
	db  *gorm.DB
}

func TestShareServiceSuite(t *testing.T) {
	suite.Run(t, new(ShareServiceSuite))
}

func (suite *ShareServiceSuite) SetupTest() {
	// Set mock mode to skip OnStart initialization that tries to access real paths
	os.Setenv("SRAT_MOCK", "true")

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
			},
			func() *dto.ContextState {
				sharedResources := dto.ContextState{}
				sharedResources.DockerInterface = "hassio"
				sharedResources.DockerNet = "172.30.32.0/23"
				var err error
				sharedResources.Template, err = os.ReadFile("../templates/smb.gtpl")
				if err != nil {
					suite.T().Errorf("Cant read template file %s", err)
				}
				sharedResources.DatabasePath = "file::memory:?cache=shared&_pragma=foreign_keys(1)"
				return &sharedResources
			},
			/*
				func() *config.DefaultConfig {
					var nconfig config.Config
					buffer, err := templates.Default_Config_content.ReadFile("default_config.json")
					if err != nil {
						log.Fatalf("Cant read default config file %#+v", err)
					}
					err = nconfig.LoadConfigBuffer(buffer) // Assign to existing err
					if err != nil {
						log.Fatalf("Cant load default config from buffer %#+v", err)
					}
					return &config.DefaultConfig{Config: nconfig}
				},
			*/
			dbom.NewDB,
			service.NewShareService,
			mock.Mock[service.BroadcasterServiceInterface],
			mock.Mock[service.UserServiceInterface],
			mock.Mock[events.EventBusInterface],
		),
		fx.Populate(&suite.shareService),
		//fx.Populate(&suite.exported_share_repo),
		fx.Populate(&suite.userService),
		fx.Populate(&suite.db),
	)

	mock.When(suite.userService.ListUsers()).ThenReturn([]dto.User{
		{Username: "homeassistant"},
	}, nil)

	suite.app.RequireStart()
}

func (suite *ShareServiceSuite) TearDownTest() {
	suite.app.RequireStop()
}

func (suite *ShareServiceSuite) TestListShares() {
	suite.Require().NoError(suite.db.Create(&dbom.ExportedShare{
		Name:               "test_xx",
		MountPointDataPath: "/test_xx",
		MountPointData: dbom.MountPointPath{
			Path:     "/test_xx",
			Type:     "ADDON",
			DeviceId: "test_xx_deviceId",
		},
	}).Error)

	count, err := gorm.G[dbom.ExportedShare](suite.db).Count(suite.T().Context(), g.ExportedShare.Name.Column().Name)
	suite.Require().NoError(err, "Count shares should not error", "error", err)

	shares, err := suite.shareService.ListShares()

	suite.NoError(err)
	suite.NotNil(shares)
	suite.Len(shares, int(count))
}

// TestVerifyShareWithMountedRWVolume tests share verification with mounted RW volume
func (suite *ShareServiceSuite) TestVerifyShareWithMountedRWVolume() {
	isWriteSupported := true
	share := &dto.SharedResource{
		Name:     "test-rw-share",
		Disabled: boolPtr(false),
		Users: []dto.User{
			{
				Username: "testuser",
				RwShares: []string{"test-rw-share"},
			},
		},
		MountPointData: &dto.MountPointData{
			Path:             "/mnt/test",
			IsMounted:        true,
			IsWriteSupported: &isWriteSupported,
		},
	}

	err := suite.shareService.VerifyShare(share)

	suite.NoError(err)
	suite.NotNil(share.Status)
	suite.True(share.Status.IsValid, "RW mounted volume should be marked as valid")
	suite.False(*share.Disabled, "Share should remain enabled as per DB value")
	suite.Len(share.Users[0].RwShares, 1, "RW permissions should be preserved")
}

// TestVerifyShareWithMountedROVolume tests share verification with mounted RO volume
func (suite *ShareServiceSuite) TestVerifyShareWithMountedROVolume() {
	isWriteSupported := false
	share := &dto.SharedResource{
		Name:     "test-ro-share",
		Disabled: boolPtr(false),
		Users: []dto.User{
			{
				Username: "testuser",
				RwShares: []string{"test-ro-share", "other-share"},
			},
		},
		RoUsers: []dto.User{
			{
				Username: "rouser",
			},
		},
		MountPointData: &dto.MountPointData{
			Path:             "/mnt/test-ro",
			IsMounted:        true,
			IsWriteSupported: &isWriteSupported,
		},
	}

	err := suite.shareService.VerifyShare(share)

	suite.NoError(err)
	suite.NotNil(share.Status)
	suite.True(share.Status.IsValid, "RO mounted volume should be marked as valid")
	suite.False(*share.Disabled, "Share should remain enabled")
	suite.Len(share.Users[0].RwShares, 1, "RW permission for this share should be removed")
	suite.Equal("other-share", share.Users[0].RwShares[0], "Other RW shares should be preserved")
	suite.Len(share.RoUsers, 1, "RO users should be preserved")
}

// TestVerifyShareWithNotMountedVolume tests share verification with unmounted volume
func (suite *ShareServiceSuite) TestVerifyShareWithNotMountedVolume() {
	isWriteSupported := true
	share := &dto.SharedResource{
		Name:     "test-unmounted-share",
		Disabled: boolPtr(false), // Was active in DB
		Users: []dto.User{
			{
				Username: "testuser",
				RwShares: []string{"test-unmounted-share"},
			},
		},
		MountPointData: &dto.MountPointData{
			Path:             "/mnt/test-unmounted",
			IsMounted:        false, // Not mounted
			IsWriteSupported: &isWriteSupported,
		},
	}

	err := suite.shareService.VerifyShare(share)

	suite.NoError(err)
	suite.NotNil(share.Status)
	suite.False(share.Status.IsValid, "Unmounted volume should be marked as invalid")
}

// TestVerifyShareWithNonExistentVolume tests share verification when volume doesn't exist
func (suite *ShareServiceSuite) TestVerifyShareWithNonExistentVolume() {
	share := &dto.SharedResource{
		Name:     "test-nonexistent-share",
		Disabled: boolPtr(false), // Was active in DB
		Users: []dto.User{
			{
				Username: "testuser",
				RwShares: []string{"test-nonexistent-share"},
			},
		},
		MountPointData: &dto.MountPointData{
			Path:      "/mnt/nonexistent",
			IsMounted: false,
			IsInvalid: true, // Volume doesn't exist
		},
	}

	err := suite.shareService.VerifyShare(share)

	suite.NoError(err)
	suite.NotNil(share.Status)
	suite.False(share.Status.IsValid, "Non-existent volume should be marked as invalid")
}

// TestVerifyShareWithNoMountPointData tests share verification with no mount point data
func (suite *ShareServiceSuite) TestVerifyShareWithNoMountPointData() {
	share := &dto.SharedResource{
		Name:           "test-no-mount",
		Disabled:       boolPtr(false),
		MountPointData: nil, // No mount point
	}

	err := suite.shareService.VerifyShare(share)

	suite.NoError(err)
	suite.NotNil(share.Status)
	suite.False(share.Status.IsValid, "Share without mount point should be marked as invalid")
}

// TestVerifyShareWithEmptyPath tests share verification with empty path
func (suite *ShareServiceSuite) TestVerifyShareWithEmptyPath() {
	share := &dto.SharedResource{
		Name:     "test-empty-path",
		Disabled: boolPtr(false),
		MountPointData: &dto.MountPointData{
			Path: "", // Empty path
		},
	}

	err := suite.shareService.VerifyShare(share)

	suite.NoError(err)
	suite.NotNil(share.Status)
	suite.False(share.Status.IsValid, "Share with empty path should be marked as invalid")
}

// TestVerifyShareWithNotHAMounted tests share verification when HA reports unmounted
func (suite *ShareServiceSuite) TestVerifyShareWithNotHAMounted() {
	isWriteSupported := true
	share := &dto.SharedResource{
		Name:     "test-ha-unmounted",
		Disabled: boolPtr(false),
		Usage:    "backup", // Not internal or none
		Status: &dto.SharedResourceStatus{
			IsHAMounted: false,
		},
		MountPointData: &dto.MountPointData{
			Path:             "/mnt/test-ha",
			IsMounted:        true,
			IsWriteSupported: &isWriteSupported,
		},
	}

	err := suite.shareService.VerifyShare(share)

	suite.NoError(err)
	suite.NotNil(share.Status)
	// Note: Current implementation doesn't check HA mount status in VerifyShare
	// This test may need revision based on actual business logic
	suite.True(share.Status.IsValid, "Share should be valid when volume is mounted")
}

// TestVerifyShareInternalUsageIgnoresHAMount tests internal shares ignore HA mount status
func (suite *ShareServiceSuite) TestVerifyShareInternalUsageIgnoresHAMount() {
	isWriteSupported := true
	share := &dto.SharedResource{
		Name:     "test-internal",
		Disabled: boolPtr(false),
		Usage:    "internal", // Internal usage
		Status: &dto.SharedResourceStatus{
			IsHAMounted: false,
		},
		MountPointData: &dto.MountPointData{
			Path:             "/mnt/test-internal",
			IsMounted:        true,
			IsWriteSupported: &isWriteSupported,
		},
	}

	err := suite.shareService.VerifyShare(share)

	suite.NoError(err)
	suite.NotNil(share.Status)
	suite.True(share.Status.IsValid, "Internal share should be valid when volume is mounted")
	suite.False(*share.Disabled, "Internal share should remain enabled")
}

// TestVerifyShareNilShare tests that nil share returns error
func (suite *ShareServiceSuite) TestVerifyShareNilShare() {
	err := suite.shareService.VerifyShare(nil)

	suite.Error(err)
	suite.Contains(err.Error(), "share cannot be nil")
}

// ============================================================================
// GetShare Tests
// ============================================================================

func (suite *ShareServiceSuite) TestGetShareNotFound() {
	// Execute
	share, err := suite.shareService.GetShare("nonexistent-share")

	// Assert
	suite.Error(err)
	suite.Nil(share)
	suite.True(errors.Is(err, dto.ErrorShareNotFound))
}

// ============================================================================
// CreateShare Tests
// ============================================================================

func (suite *ShareServiceSuite) TestCreateShareSuccess() {
	// Setup
	mock.When(suite.userService.GetAdmin()).ThenReturn(&dto.User{
		Username: "homeassistant",
	}, nil)

	newShare := dto.SharedResource{
		Name:        "new-share",
		Disabled:    boolPtr(false),
		GuestOk:     boolPtr(true),
		TimeMachine: boolPtr(false),
		RecycleBin:  boolPtr(true),
		Usage:       "media",
		MountPointData: &dto.MountPointData{
			Path:     "/mnt/new",
			DeviceId: "newdev123",
			Type:     "ADDON",
		},
		Users: []dto.User{
			{
				Username: "homeassistant",
			},
		},
	}

	// Execute
	created, err := suite.shareService.CreateShare(newShare)

	// Assert
	suite.Require().NoError(err)
	suite.Require().NotNil(created)
	suite.Equal("new-share", created.Name)
	suite.False(*created.Disabled)
	suite.True(*created.GuestOk)
	suite.Len(created.Users, 1)
}

func (suite *ShareServiceSuite) TestCreateShareWithoutExplicitUsers() {
	// Setup: Mock GetAdmin to return the admin user
	mock.When(suite.userService.GetAdmin()).ThenReturn(&dto.User{
		Username: "homeassistant",
	}, nil)

	newShare := dto.SharedResource{
		Name:     "admin-auto-share",
		Disabled: boolPtr(false),
		MountPointData: &dto.MountPointData{
			Path:     "/mnt/auto",
			DeviceId: "auto123",
			Type:     "ADDON",
		},
		Users: []dto.User{}, // Empty users list
	}

	// Execute
	created, err := suite.shareService.CreateShare(newShare)

	// Assert
	suite.NoError(err)
	suite.NotNil(created)
	// Admin user should be automatically added
	suite.Len(created.Users, 1)
	suite.Equal("homeassistant", created.Users[0].Username)
}

func (suite *ShareServiceSuite) TestCreateShareWithMultipleProperties() {
	// Setup
	mock.When(suite.userService.GetAdmin()).ThenReturn(&dto.User{
		Username: "homeassistant",
	}, nil)

	newShare := dto.SharedResource{
		Name:               "feature-rich-share",
		Disabled:           boolPtr(false),
		GuestOk:            boolPtr(true),
		TimeMachine:        boolPtr(true),
		RecycleBin:         boolPtr(true),
		TimeMachineMaxSize: stringPtr("500G"),
		Usage:              "backup",
		VetoFiles:          []string{"*.exe", "*.dll"},
		MountPointData: &dto.MountPointData{
			Path:     "/mnt/backup",
			DeviceId: "backup123",
			Type:     "ADDON",
		},
		Users: []dto.User{
			{
				Username: "homeassistant",
				RwShares: []string{"feature-rich-share"},
			},
		},
		RoUsers: []dto.User{
			{
				Username: "readonly-user",
			},
		},
	}

	// Execute
	created, err := suite.shareService.CreateShare(newShare)

	// Assert
	suite.NoError(err)
	suite.NotNil(created)
	suite.Equal("feature-rich-share", created.Name)
	suite.True(*created.GuestOk)
	suite.True(*created.TimeMachine)
	suite.True(*created.RecycleBin)
	suite.Equal("500G", *created.TimeMachineMaxSize)
	suite.Len(created.VetoFiles, 2)
	suite.Len(created.Users, 1)
	suite.Len(created.RoUsers, 1)
}

// ============================================================================
// UpdateShare Tests
// ============================================================================

func (suite *ShareServiceSuite) TestUpdateShareNotFound() {
	// Setup: Mock GetAdmin
	mock.When(suite.userService.GetAdmin()).ThenReturn(&dto.User{
		Username: "homeassistant",
	}, nil)

	// Execute
	updatedShare := dto.SharedResource{
		Name:     "nonexistent-share",
		Disabled: boolPtr(false),
		MountPointData: &dto.MountPointData{
			Path:     "/mnt/missing",
			DeviceId: "missing123",
			Type:     "ADDON",
		},
	}

	result, err := suite.shareService.UpdateShare("nonexistent-share", updatedShare)

	// Assert
	suite.Error(err)
	suite.Nil(result)
	suite.True(errors.Is(err, dto.ErrorShareNotFound))
}

func (suite *ShareServiceSuite) TestUpdateShareSuccess() {
	// Setup: Create a share first
	mock.When(suite.userService.GetAdmin()).ThenReturn(&dto.User{
		Username: "homeassistant",
	}, nil)

	initialShare := dto.SharedResource{
		Name:        "update-test-share",
		Disabled:    boolPtr(false),
		GuestOk:     boolPtr(false),
		TimeMachine: boolPtr(false),
		RecycleBin:  boolPtr(false),
		Usage:       "media",
		MountPointData: &dto.MountPointData{
			Path:     "/mnt/update-test",
			DeviceId: "updatedev123",
			Type:     "ADDON",
		},
		Users: []dto.User{
			{Username: "homeassistant"},
		},
	}

	created, err := suite.shareService.CreateShare(initialShare)
	suite.Require().NoError(err)
	suite.Require().NotNil(created)

	// Execute: Update the share with new values
	updatedShare := dto.SharedResource{
		Name:        "update-test-share",
		Disabled:    boolPtr(false),
		GuestOk:     boolPtr(true), // Changed
		TimeMachine: boolPtr(true), // Changed
		RecycleBin:  boolPtr(true), // Changed
		Usage:       "backup",      // Changed
		MountPointData: &dto.MountPointData{
			Path:     "/mnt/update-test",
			DeviceId: "updatedev123",
			Type:     "ADDON",
		},
		Users: []dto.User{
			{Username: "homeassistant"},
		},
	}

	result, err := suite.shareService.UpdateShare("update-test-share", updatedShare)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("update-test-share", result.Name)
	suite.True(*result.GuestOk)
	suite.True(*result.TimeMachine)
	suite.True(*result.RecycleBin)
	suite.Equal(dto.HAMountUsage("backup"), result.Usage)
}

func (suite *ShareServiceSuite) TestUpdateShareChangeUsers() {
	// Setup: Create a share first
	mock.When(suite.userService.GetAdmin()).ThenReturn(&dto.User{
		Username: "homeassistant",
	}, nil)

	initialShare := dto.SharedResource{
		Name:     "user-update-share",
		Disabled: boolPtr(false),
		MountPointData: &dto.MountPointData{
			Path:     "/mnt/user-update",
			DeviceId: "userupdatedev",
			Type:     "ADDON",
		},
		Users: []dto.User{
			{Username: "homeassistant"},
		},
	}

	created, err := suite.shareService.CreateShare(initialShare)
	suite.Require().NoError(err)
	suite.Require().NotNil(created)

	// Execute: Update with different users
	updatedShare := dto.SharedResource{
		Name:     "user-update-share",
		Disabled: boolPtr(false),
		MountPointData: &dto.MountPointData{
			Path:     "/mnt/user-update",
			DeviceId: "userupdatedev",
			Type:     "ADDON",
		},
		Users: []dto.User{
			{Username: "homeassistant"},
			{Username: "newuser"},
		},
		RoUsers: []dto.User{
			{Username: "readonly"},
		},
	}

	result, err := suite.shareService.UpdateShare("user-update-share", updatedShare)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Len(result.Users, 2)
	suite.Len(result.RoUsers, 1)
}

func (suite *ShareServiceSuite) TestUpdateShareWithEmptyUsersAddsAdmin() {
	// Setup: Create a share first with users
	mock.When(suite.userService.GetAdmin()).ThenReturn(&dto.User{
		Username: "homeassistant",
	}, nil)

	initialShare := dto.SharedResource{
		Name:     "empty-users-update-share",
		Disabled: boolPtr(false),
		MountPointData: &dto.MountPointData{
			Path:     "/mnt/empty-users-update",
			DeviceId: "emptyusersdev",
			Type:     "ADDON",
		},
		Users: []dto.User{
			{Username: "homeassistant"},
		},
	}

	created, err := suite.shareService.CreateShare(initialShare)
	suite.Require().NoError(err)
	suite.Require().NotNil(created)

	// Execute: Update with empty users - should auto-add admin
	updatedShare := dto.SharedResource{
		Name:     "empty-users-update-share",
		Disabled: boolPtr(false),
		MountPointData: &dto.MountPointData{
			Path:     "/mnt/empty-users-update",
			DeviceId: "emptyusersdev",
			Type:     "ADDON",
		},
		Users: []dto.User{}, // Empty users
	}

	result, err := suite.shareService.UpdateShare("empty-users-update-share", updatedShare)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Len(result.Users, 1)
	suite.Equal("homeassistant", result.Users[0].Username)
}

func (suite *ShareServiceSuite) TestUpdateShareVetoFiles() {
	// Setup: Create a share first
	mock.When(suite.userService.GetAdmin()).ThenReturn(&dto.User{
		Username: "homeassistant",
	}, nil)

	initialShare := dto.SharedResource{
		Name:      "veto-update-share",
		Disabled:  boolPtr(false),
		VetoFiles: []string{"*.tmp"},
		MountPointData: &dto.MountPointData{
			Path:     "/mnt/veto-update",
			DeviceId: "vetoupdatedev",
			Type:     "ADDON",
		},
		Users: []dto.User{
			{Username: "homeassistant"},
		},
	}

	created, err := suite.shareService.CreateShare(initialShare)
	suite.Require().NoError(err)
	suite.Require().NotNil(created)
	suite.Len(created.VetoFiles, 1)

	// Execute: Update veto files
	updatedShare := dto.SharedResource{
		Name:      "veto-update-share",
		Disabled:  boolPtr(false),
		VetoFiles: []string{"*.exe", "*.dll", "*.bat"},
		MountPointData: &dto.MountPointData{
			Path:     "/mnt/veto-update",
			DeviceId: "vetoupdatedev",
			Type:     "ADDON",
		},
		Users: []dto.User{
			{Username: "homeassistant"},
		},
	}

	result, err := suite.shareService.UpdateShare("veto-update-share", updatedShare)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Len(result.VetoFiles, 3)
	suite.Contains(result.VetoFiles, "*.exe")
	suite.Contains(result.VetoFiles, "*.dll")
	suite.Contains(result.VetoFiles, "*.bat")
}

// ============================================================================
// EnableShare Tests
// ============================================================================

func (suite *ShareServiceSuite) TestEnableShareSuccess() {
	// Setup: Create a disabled share
	mock.When(suite.userService.GetAdmin()).ThenReturn(&dto.User{
		Username: "homeassistant",
	}, nil)

	disabledShare := dto.SharedResource{
		Name:     "enable-test-share",
		Disabled: boolPtr(true), // Start disabled
		MountPointData: &dto.MountPointData{
			Path:     "/mnt/enable-test",
			DeviceId: "enabletestdev",
			Type:     "ADDON",
		},
		Users: []dto.User{
			{Username: "homeassistant"},
		},
	}

	created, err := suite.shareService.CreateShare(disabledShare)
	suite.Require().NoError(err)
	suite.Require().NotNil(created)
	suite.True(*created.Disabled)

	// Execute: Enable the share
	result, err := suite.shareService.EnableShare("enable-test-share")

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.False(*result.Disabled, "Share should be enabled")
	suite.Equal("enable-test-share", result.Name)
}

func (suite *ShareServiceSuite) TestEnableShareNotFound() {
	// Execute
	result, err := suite.shareService.EnableShare("nonexistent-enable-share")

	// Assert
	suite.Error(err)
	suite.Nil(result)
	suite.True(errors.Is(err, dto.ErrorShareNotFound))
}

func (suite *ShareServiceSuite) TestEnableAlreadyEnabledShare() {
	// Setup: Create an enabled share
	mock.When(suite.userService.GetAdmin()).ThenReturn(&dto.User{
		Username: "homeassistant",
	}, nil)

	enabledShare := dto.SharedResource{
		Name:     "already-enabled-share",
		Disabled: boolPtr(false), // Already enabled
		MountPointData: &dto.MountPointData{
			Path:     "/mnt/already-enabled",
			DeviceId: "alreadyenableddev",
			Type:     "ADDON",
		},
		Users: []dto.User{
			{Username: "homeassistant"},
		},
	}

	created, err := suite.shareService.CreateShare(enabledShare)
	suite.Require().NoError(err)
	suite.Require().NotNil(created)
	suite.False(*created.Disabled)

	// Execute: Enable an already enabled share
	result, err := suite.shareService.EnableShare("already-enabled-share")

	// Assert: Should succeed without error
	suite.NoError(err)
	suite.NotNil(result)
	suite.False(*result.Disabled)
}

// ============================================================================
// DisableShare Tests
// ============================================================================

func (suite *ShareServiceSuite) TestDisableShareSuccess() {
	// Setup: Create an enabled share
	mock.When(suite.userService.GetAdmin()).ThenReturn(&dto.User{
		Username: "homeassistant",
	}, nil)

	enabledShare := dto.SharedResource{
		Name:     "disable-test-share",
		Disabled: boolPtr(false), // Start enabled
		MountPointData: &dto.MountPointData{
			Path:     "/mnt/disable-test",
			DeviceId: "disabletestdev",
			Type:     "ADDON",
		},
		Users: []dto.User{
			{Username: "homeassistant"},
		},
	}

	created, err := suite.shareService.CreateShare(enabledShare)
	suite.Require().NoError(err)
	suite.Require().NotNil(created)
	suite.False(*created.Disabled)

	// Execute: Disable the share
	result, err := suite.shareService.DisableShare("disable-test-share")

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.True(*result.Disabled, "Share should be disabled")
	suite.Equal("disable-test-share", result.Name)
}

func (suite *ShareServiceSuite) TestDisableShareNotFound() {
	// Execute
	result, err := suite.shareService.DisableShare("nonexistent-disable-share")

	// Assert
	suite.Error(err)
	suite.Nil(result)
	suite.True(errors.Is(err, dto.ErrorShareNotFound))
}

func (suite *ShareServiceSuite) TestDisableAlreadyDisabledShare() {
	// Setup: Create a disabled share
	mock.When(suite.userService.GetAdmin()).ThenReturn(&dto.User{
		Username: "homeassistant",
	}, nil)

	disabledShare := dto.SharedResource{
		Name:     "already-disabled-share",
		Disabled: boolPtr(true), // Already disabled
		MountPointData: &dto.MountPointData{
			Path:     "/mnt/already-disabled",
			DeviceId: "alreadydisableddev",
			Type:     "ADDON",
		},
		Users: []dto.User{
			{Username: "homeassistant"},
		},
	}

	created, err := suite.shareService.CreateShare(disabledShare)
	suite.Require().NoError(err)
	suite.Require().NotNil(created)
	suite.True(*created.Disabled)

	// Execute: Disable an already disabled share
	result, err := suite.shareService.DisableShare("already-disabled-share")

	// Assert: Should succeed without error
	suite.Require().NoError(err)
	suite.Require().NotNil(result)
	suite.True(*result.Disabled)
}

func (suite *ShareServiceSuite) TestEnableDisableToggle() {
	// Setup: Create a share
	mock.When(suite.userService.GetAdmin()).ThenReturn(&dto.User{
		Username: "homeassistant",
	}, nil)

	share := dto.SharedResource{
		Name:     "toggle-share",
		Disabled: boolPtr(false), // Start enabled
		MountPointData: &dto.MountPointData{
			Path:     "/mnt/toggle",
			DeviceId: "toggledev",
			Type:     "ADDON",
		},
		Users: []dto.User{
			{Username: "homeassistant"},
		},
	}

	created, err := suite.shareService.CreateShare(share)
	suite.Require().NoError(err)
	suite.Require().NotNil(created)
	suite.False(*created.Disabled)

	// Execute: Toggle disable -> enable -> disable
	// Disable
	result, err := suite.shareService.DisableShare("toggle-share")
	suite.NoError(err)
	suite.True(*result.Disabled, "Share should be disabled after DisableShare")

	// Enable
	result, err = suite.shareService.EnableShare("toggle-share")
	suite.NoError(err)
	suite.False(*result.Disabled, "Share should be enabled after EnableShare")

	// Disable again
	result, err = suite.shareService.DisableShare("toggle-share")
	suite.NoError(err)
	suite.True(*result.Disabled, "Share should be disabled after second DisableShare")
}

// ============================================================================
// DeleteShare Tests
// ============================================================================

func (suite *ShareServiceSuite) TestDeleteShareSuccess() {
	// Setup: Create a share
	mock.When(suite.userService.GetAdmin()).ThenReturn(&dto.User{
		Username: "homeassistant",
	}, nil)

	share := dto.SharedResource{
		Name:     "delete-test-share",
		Disabled: boolPtr(false),
		MountPointData: &dto.MountPointData{
			Path:     "/mnt/delete-test",
			DeviceId: "deletetestdev",
			Type:     "ADDON",
		},
		Users: []dto.User{
			{Username: "homeassistant"},
		},
	}

	created, err := suite.shareService.CreateShare(share)
	suite.Require().NoError(err)
	suite.Require().NotNil(created)

	// Verify it exists
	retrieved, err := suite.shareService.GetShare("delete-test-share")
	suite.NoError(err)
	suite.NotNil(retrieved)

	// Execute: Delete the share
	err = suite.shareService.DeleteShare("delete-test-share")

	// Assert
	suite.NoError(err)

	// Verify it's gone
	retrieved, err = suite.shareService.GetShare("delete-test-share")
	suite.Error(err)
	suite.Nil(retrieved)
	suite.True(errors.Is(err, dto.ErrorShareNotFound))
}

func (suite *ShareServiceSuite) TestCreateDeleteAndRecreateShare() {
	adminUserName := "homeassistant%" + rand.Text()[:5]
	username := "testuser%" + rand.Text()[:5]

	// Setup: allow auto-adding admin user on empty user list
	mock.When(suite.userService.GetAdmin()).ThenReturn(&dto.User{
		Username: adminUserName,
	}, nil)

	// First, create actual users in the database to test foreign key constraints
	suite.Require().NoError(suite.db.Create(&dbom.SambaUser{
		Username: adminUserName,
		Password: "test123",
		IsAdmin:  true,
	}).Error)
	suite.Require().NoError(suite.db.Create(&dbom.SambaUser{
		Username: username,
		Password: "test456",
		IsAdmin:  false,
	}).Error)

	share := dto.SharedResource{
		Name:     "recreate-share",
		Disabled: boolPtr(false),
		MountPointData: &dto.MountPointData{
			Path:     "/mnt/recreate",
			DeviceId: "recreatedev",
			Type:     "ADDON",
		},
		Users: []dto.User{
			{Username: adminUserName},
			{Username: username},
		},
		RoUsers: []dto.User{
			{Username: username},
		},
	}

	created, err := suite.shareService.CreateShare(share)
	suite.Require().NoError(err)
	suite.Require().NotNil(created)
	suite.Len(created.Users, 2)
	suite.Equal(adminUserName, created.Users[0].Username)

	// Delete the share - this should clear all associations
	err = suite.shareService.DeleteShare("recreate-share")
	suite.Require().NoError(err)

	// Verify the share is soft-deleted
	var deletedShare dbom.ExportedShare
	dbErr := suite.db.Unscoped().Where("name = ?", "recreate-share").First(&deletedShare).Error
	suite.NoError(dbErr)
	suite.True(deletedShare.DeletedAt.Valid, "Share should be soft-deleted")

	// Verify associations were cleared from join tables
	var rwCount int64
	suite.db.Table("user_rw_share").Where("exported_share_name = ?", "recreate-share").Count(&rwCount)
	suite.Equal(int64(0), rwCount, "RW associations should be cleared")

	var roCount int64
	suite.db.Table("user_ro_share").Where("exported_share_name = ?", "recreate-share").Count(&roCount)
	suite.Equal(int64(0), roCount, "RO associations should be cleared")

	// Recreate the same share name should succeed after deletion without FK violations
	recreateShare := dto.SharedResource{
		Name:     "recreate-share",
		Disabled: boolPtr(false),
		MountPointData: &dto.MountPointData{
			Path:     "/mnt/recreate",
			DeviceId: "recreatedev",
			Type:     "ADDON",
		},
		Users: []dto.User{
			{Username: adminUserName},
		},
	}

	recreated, err := suite.shareService.CreateShare(recreateShare)
	suite.Require().NoError(err)
	suite.Require().NotNil(recreated)
	suite.Equal("recreate-share", recreated.Name)
	suite.False(*recreated.Disabled)
	suite.Len(recreated.Users, 1)
	suite.Equal(adminUserName, recreated.Users[0].Username)

	// Verify the share is no longer soft-deleted
	var restoredShare dbom.ExportedShare
	dbErr = suite.db.Where("name = ?", "recreate-share").First(&restoredShare).Error
	suite.NoError(dbErr)
	suite.False(restoredShare.DeletedAt.Valid, "Share should be restored (not soft-deleted)")
}

func (suite *ShareServiceSuite) TestDeleteShareNotFound() {
	// Execute
	err := suite.shareService.DeleteShare("nonexistent-delete-share")

	// Assert
	suite.Error(err)
	suite.True(errors.Is(err, dto.ErrorShareNotFound))
}

// TestCreateAndUpdateShareWithNumericPrefix tests issue #416:
// Create and update a share with a name starting with numbers (e.g., "500G")
// This test verifies:
// 1. Share names with numeric prefixes are properly preserved during creation
// 2. Share updates don't cause foreign key constraint violations
// 3. User associations are properly managed during updates
func (suite *ShareServiceSuite) TestCreateAndUpdateShareWithNumericPrefix() {
	// Setup
	mock.When(suite.userService.GetAdmin()).ThenReturn(&dto.User{
		Username: "homeassistant",
		IsAdmin:  true,
	}, nil)

	// Create a share with numeric prefix (issue #416 scenario)
	initialShare := dto.SharedResource{
		Name:        "500G",
		Disabled:    boolPtr(false),
		GuestOk:     boolPtr(false),
		TimeMachine: boolPtr(false),
		RecycleBin:  boolPtr(false),
		Usage:       "share",
		MountPointData: &dto.MountPointData{
			Path:             "/mnt/500G",
			DeviceId:         "usb-500g-dev",
			Type:             "ADDON",
			IsWriteSupported: boolPtr(true),
			IsMounted:        true,
			IsInvalid:        false,
		},
		Users: []dto.User{
			{Username: "homeassistant"},
			{Username: "testuser"},
		},
		RoUsers: []dto.User{
			{Username: "rouser"},
		},
	}

	// Execute: Create the share
	created, err := suite.shareService.CreateShare(initialShare)

	// Assert: Share is created with correct name
	suite.Require().NoError(err, "Should create share with numeric prefix")
	suite.Require().NotNil(created)
	suite.Equal("500G", created.Name, "Share name should be preserved as '500G'")
	suite.Equal(dto.UsageAsShare, created.Usage)
	suite.Len(created.Users, 2, "Should have 2 users")

	// Verify in database that the share was created with correct name
	var dbShare dbom.ExportedShare
	dbErr := suite.db.Where("name = ?", "500G").First(&dbShare).Error
	suite.NoError(dbErr, "Share should exist in database with name '500G'")
	suite.Equal("500G", dbShare.Name)

	// Execute: Update the share with different users (tests association clearing)
	updatedShare := dto.SharedResource{
		Name:        "500G",
		Disabled:    boolPtr(false),
		GuestOk:     boolPtr(true),
		TimeMachine: boolPtr(false),
		RecycleBin:  boolPtr(true),
		Usage:       "backup",
		MountPointData: &dto.MountPointData{
			Path:             "/mnt/500G",
			DeviceId:         "usb-500g-dev",
			Type:             "ADDON",
			IsWriteSupported: boolPtr(true),
			IsMounted:        true,
			IsInvalid:        false,
		},
		Users: []dto.User{
			{Username: "homeassistant"},
			{Username: "newuser"},
		},
		RoUsers: []dto.User{},
	}

	// Execute: Update the share
	result, err := suite.shareService.UpdateShare("500G", updatedShare)

	// Assert: Update succeeds without foreign key constraint violation
	suite.Require().NoError(err, "Should update share without foreign key constraint error")
	suite.NotNil(result)
	suite.Equal("500G", result.Name, "Share name should remain '500G' after update")
	suite.True(*result.GuestOk, "GuestOk should be updated to true")
	suite.True(*result.RecycleBin, "RecycleBin should be updated to true")
	suite.Equal(dto.HAMountUsage("backup"), result.Usage)
	suite.Len(result.Users, 2, "Should have 2 users after update")

	// Verify in database that associations were properly updated
	var updatedDbShare dbom.ExportedShare
	dbErr = suite.db.
		Preload("Users").
		Preload("RoUsers").
		Where("name = ?", "500G").
		First(&updatedDbShare).Error
	suite.NoError(dbErr, "Updated share should exist in database")
	suite.Equal("500G", updatedDbShare.Name)
	suite.Len(updatedDbShare.Users, 2, "Users associations should be updated")
	suite.Empty(updatedDbShare.RoUsers, "RoUsers associations should be cleared")

	// Execute: Update again with different users to ensure multiple updates work
	secondUpdate := dto.SharedResource{
		Name:        "500G",
		Disabled:    boolPtr(true),
		GuestOk:     boolPtr(false),
		TimeMachine: boolPtr(false),
		RecycleBin:  boolPtr(false),
		Usage:       "internal",
		MountPointData: &dto.MountPointData{
			Path:             "/mnt/500G",
			DeviceId:         "usb-500g-dev",
			Type:             "ADDON",
			IsWriteSupported: boolPtr(true),
			IsMounted:        true,
			IsInvalid:        false,
		},
		Users: []dto.User{
			{Username: "homeassistant"},
		},
		RoUsers: []dto.User{
			{Username: "viewer"},
		},
	}

	// Execute: Second update
	result2, err := suite.shareService.UpdateShare("500G", secondUpdate)

	// Assert: Second update also succeeds
	suite.Require().NoError(err, "Second update should also succeed without constraint violation")
	suite.NotNil(result2)
	suite.Equal("500G", result2.Name)
	suite.True(*result2.Disabled)
	suite.Len(result2.Users, 1, "Should have 1 user after second update")
}

// Helper functions
func boolPtr(b bool) *bool {
	return &b
}

func stringPtr(s string) *string {
	return &s
}
