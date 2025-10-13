package service_test

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type ShareServiceSuite struct {
	suite.Suite
	shareService        service.ShareServiceInterface
	exported_share_repo repository.ExportedShareRepositoryInterface
	app                 *fxtest.App
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

				return &sharedResources
			},
			service.NewShareService,
			mock.Mock[service.BroadcasterServiceInterface],
			mock.Mock[repository.ExportedShareRepositoryInterface],
			mock.Mock[repository.MountPointPathRepositoryInterface],
			mock.Mock[repository.SambaUserRepositoryInterface],
		),
		fx.Populate(&suite.shareService),
		fx.Populate(&suite.exported_share_repo),
	)
	suite.app.RequireStart()
}

func (suite *ShareServiceSuite) TearDownTest() {
	suite.app.RequireStop()
}

func (suite *ShareServiceSuite) TestAll() {
	mock.When(suite.exported_share_repo.All()).ThenReturn(&[]dbom.ExportedShare{
		{
			Name: "test",
		},
	}, nil)

	shares, err := suite.shareService.All()

	suite.NoError(err)
	suite.NotNil(shares)
	suite.Len(*shares, 1)
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
	suite.NotNil(share.Invalid)
	suite.False(*share.Invalid, "RW mounted volume should not be marked as invalid")
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
	suite.NotNil(share.Invalid)
	suite.False(*share.Invalid, "RO mounted volume should not be marked as invalid")
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
	suite.NotNil(share.Invalid)
	suite.True(*share.Invalid, "Unmounted volume should be marked as invalid/anomaly")
	suite.NotNil(share.Disabled)
	suite.True(*share.Disabled, "Share should be disabled when volume is not mounted")
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
	suite.NotNil(share.Invalid)
	suite.True(*share.Invalid, "Non-existent volume should be marked as invalid/anomaly")
	suite.NotNil(share.Disabled)
	suite.True(*share.Disabled, "Share should be disabled when volume doesn't exist")
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
	suite.NotNil(share.Invalid)
	suite.True(*share.Invalid, "Share without mount point should be marked as invalid")
	suite.NotNil(share.Disabled)
	suite.True(*share.Disabled, "Share without mount point should be disabled")
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
	suite.NotNil(share.Invalid)
	suite.True(*share.Invalid, "Share with empty path should be marked as invalid")
	suite.NotNil(share.Disabled)
	suite.True(*share.Disabled, "Share with empty path should be disabled")
}

// TestVerifyShareWithNotHAMounted tests share verification when HA reports unmounted
func (suite *ShareServiceSuite) TestVerifyShareWithNotHAMounted() {
	isWriteSupported := true
	isHAMounted := false
	haStatus := "not_mounted"
	share := &dto.SharedResource{
		Name:        "test-ha-unmounted",
		Disabled:    boolPtr(false),
		Usage:       "backup", // Not internal or none
		IsHAMounted: &isHAMounted,
		HaStatus:    &haStatus,
		MountPointData: &dto.MountPointData{
			Path:             "/mnt/test-ha",
			IsMounted:        true,
			IsWriteSupported: &isWriteSupported,
		},
	}

	err := suite.shareService.VerifyShare(share)

	suite.NoError(err)
	suite.NotNil(share.Invalid)
	suite.True(*share.Invalid, "Share not mounted in HA should be marked as invalid")
	suite.NotNil(share.Disabled)
	suite.True(*share.Disabled, "Share not mounted in HA should be disabled")
}

// TestVerifyShareInternalUsageIgnoresHAMount tests internal shares ignore HA mount status
func (suite *ShareServiceSuite) TestVerifyShareInternalUsageIgnoresHAMount() {
	isWriteSupported := true
	isHAMounted := false
	share := &dto.SharedResource{
		Name:        "test-internal",
		Disabled:    boolPtr(false),
		Usage:       "internal", // Internal usage
		IsHAMounted: &isHAMounted,
		MountPointData: &dto.MountPointData{
			Path:             "/mnt/test-internal",
			IsMounted:        true,
			IsWriteSupported: &isWriteSupported,
		},
	}

	err := suite.shareService.VerifyShare(share)

	suite.NoError(err)
	suite.NotNil(share.Invalid)
	suite.False(*share.Invalid, "Internal share should not be invalidated by HA mount status")
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
