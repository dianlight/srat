// Package service contains white-box tests that need access to unexported
// VolumeService internals (runProvisionalRecheck, getVolumesData).
package service

import (
	"context"
	"sync"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
)

// VolumeServiceInternalSuite exercises runProvisionalRecheck directly.
type VolumeServiceInternalSuite struct {
	suite.Suite
	ctrl          *matchers.MockController
	mockHardware  HardwareServiceInterface
	volumeService *VolumeService
	ctx           context.Context
	cancel        context.CancelFunc
}

func TestVolumeServiceInternalSuite(t *testing.T) {
	suite.Run(t, new(VolumeServiceInternalSuite))
}

func (suite *VolumeServiceInternalSuite) SetupTest() {
	suite.ctrl = mock.NewMockController(suite.T())
	suite.mockHardware = mock.Mock[HardwareServiceInterface](suite.ctrl)
	suite.ctx, suite.cancel = context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, &sync.WaitGroup{}))
	suite.volumeService = &VolumeService{
		ctx:            suite.ctx,
		hardwareClient: suite.mockHardware,
		disks:          dto.NewDiskMap(),
	}
}

func (suite *VolumeServiceInternalSuite) TearDownTest() {
	suite.cancel()
}

func (suite *VolumeServiceInternalSuite) TestRunProvisionalRecheck_CancelledContextReturnsEarly() {
	suite.cancel()
	// With a cancelled context the recheck must return without touching
	// the hardware client.
	suite.volumeService.runProvisionalRecheck()
	mock.Verify(suite.mockHardware, matchers.Times(0)).InvalidateHardwareInfo()
	_, _ = mock.Verify(suite.mockHardware, matchers.Times(0)).GetHardwareInfo()
}

func (suite *VolumeServiceInternalSuite) TestRunProvisionalRecheck_InvalidatesAndRefreshes() {
	mock.When(suite.mockHardware.GetHardwareInfo()).ThenReturn(map[string]dto.Disk{}, nil)

	suite.volumeService.runProvisionalRecheck()

	mock.Verify(suite.mockHardware, matchers.Times(1)).InvalidateHardwareInfo()
	_, _ = mock.Verify(suite.mockHardware, matchers.Times(1)).GetHardwareInfo()
}

func (suite *VolumeServiceInternalSuite) TestRunProvisionalRecheck_LogsOnGetVolumesDataError() {
	mock.When(suite.mockHardware.GetHardwareInfo()).ThenReturn(nil, errors.New("hw fetch failed"))

	suite.volumeService.runProvisionalRecheck()

	mock.Verify(suite.mockHardware, matchers.Times(1)).InvalidateHardwareInfo()
	_, _ = mock.Verify(suite.mockHardware, matchers.Times(1)).GetHardwareInfo()
}
