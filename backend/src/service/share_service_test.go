package service_test

import (
	"context"
	"log"
	"os"
	"sync"
	"testing"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dbom/g"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/srat/templates"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
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
			dbom.NewDB,
			service.NewShareService,
			mock.Mock[service.BroadcasterServiceInterface],
			mock.Mock[service.UserServiceInterface],
			//mock.Mock[repository.ExportedShareRepositoryInterface],
			//	mock.Mock[repository.MountPointPathRepositoryInterface],
			mock.Mock[repository.SambaUserRepositoryInterface],
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

// Helper function
func boolPtr(b bool) *bool {
	return &b
}
