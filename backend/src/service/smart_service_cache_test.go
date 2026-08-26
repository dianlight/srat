package service_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/dianlight/smartmontools-sdk/bindings/go/v8"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	goerrors "gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

// SmartServiceCacheSuite verifies the GetSmartInfo result cache that prevents
// repeated SCSI probes during hardware re-enumeration (udev event storms).
type SmartServiceCacheSuite struct {
	suite.Suite
	service     service.SmartServiceInterface
	smartClient smartmontools.SmartClient
	app         *fxtest.App
}

func TestSmartServiceCacheSuite(t *testing.T) {
	suite.Run(t, new(SmartServiceCacheSuite))
}

func (suite *SmartServiceCacheSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) { return context.WithCancel(context.Background()) },
			func(ctx context.Context) events.EventBusInterface { return events.NewEventBus(ctx) },
			service.NewSmartService,
			mock.Mock[smartmontools.SmartClient],
			mock.Mock[service.BroadcasterServiceInterface],
		),
		fx.Populate(&suite.service),
		fx.Populate(&suite.smartClient),
	)
	suite.app.RequireStart()
}

func (suite *SmartServiceCacheSuite) TearDownTest() {
	suite.app.RequireStop()
}

func (suite *SmartServiceCacheSuite) TestGetSmartInfo_SecondCallServedFromCache() {
	tempFile, _ := os.CreateTemp("", "cachetest")
	defer os.Remove(tempFile.Name())

	mockSMARTInfo := &smartmontools.SMARTInfo{
		Device: smartmontools.Device{
			Name: tempFile.Name(),
			Type: "sat",
		},
		SmartSupport: &smartmontools.SmartSupport{
			Available: true,
			Enabled:   true,
		},
		Temperature: &smartmontools.Temperature{
			Current: 35,
		},
		PowerOnTime: &smartmontools.PowerOnTime{
			Hours: 1000,
		},
		PowerCycleCount: 50,
	}

	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(mockSMARTInfo, nil)

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})

	info1, err1 := suite.service.GetSmartInfo(context.Background(), tempFile.Name())
	suite.Require().NoError(err1)
	suite.Require().NotNil(info1)

	info2, err2 := suite.service.GetSmartInfo(context.Background(), tempFile.Name())
	suite.Require().NoError(err2)
	suite.Equal(info1, info2, "cached result must be identical")

	// The underlying client must have been contacted exactly once.
	mock.Verify(suite.smartClient, matchers.Times(1)).GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))
}

func (suite *SmartServiceCacheSuite) TestGetSmartInfo_ErrorIsAlsoCached() {
	// Negative caching: unsupported devices are precisely the ones being
	// re-probed by every enumeration, so errors must be cached too.
	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact("/dev/nonexistent"))).
		ThenReturn(nil, fmt.Errorf("SMART Not Supported"))

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return "/dev/nonexistent", nil
	})

	_, err1 := suite.service.GetSmartInfo(context.Background(), "nonexistent")
	suite.Require().Error(err1)
	suite.True(goerrors.Is(err1, dto.ErrorSMARTNotSupported))

	_, err2 := suite.service.GetSmartInfo(context.Background(), "nonexistent")
	suite.Require().Error(err2)
	suite.True(goerrors.Is(err2, dto.ErrorSMARTNotSupported))

	mock.Verify(suite.smartClient, matchers.Times(1)).GetSMARTInfo(mock.Any[context.Context](), mock.Exact("/dev/nonexistent"))
}

func (suite *SmartServiceCacheSuite) TestEnableSMART_InvalidatesCache() {
	tempFile, _ := os.CreateTemp("", "invalidatetest")
	defer os.Remove(tempFile.Name())

	mockSMARTInfo := &smartmontools.SMARTInfo{
		SmartSupport: &smartmontools.SmartSupport{Available: true, Enabled: false},
	}

	// Static responses: the point is that after EnableSMART drops the cache
	// entry, a subsequent GetSmartInfo must hit the client again.
	mock.When(suite.smartClient.EnableSMART(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(nil)
	mock.When(suite.smartClient.IsSMARTSupported(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(&smartmontools.SmartSupport{Available: true, Enabled: true}, nil)
	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(mockSMARTInfo, nil)

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})

	before, err := suite.service.GetSmartInfo(context.Background(), tempFile.Name())
	suite.Require().NoError(err)
	suite.NotNil(before)

	// Cached: no additional client calls yet.
	_, err = suite.service.GetSmartInfo(context.Background(), tempFile.Name())
	suite.Require().NoError(err)

	suite.Require().NoError(suite.service.EnableSMART(context.Background(), tempFile.Name()))

	// Cache was invalidated: this call must reach the client again.
	after, err := suite.service.GetSmartInfo(context.Background(), tempFile.Name())
	suite.Require().NoError(err)
	suite.Equal(before, after)

	// Three client GetSMARTInfo calls total: once before enable (cached by
	// the second pre-enable call), once from inside EnableSMART's
	// post-verification refresh, once after invalidation on our final read.
	// Fewer would mean the stale entry survived EnableSMART.
	mock.Verify(suite.smartClient, matchers.Times(3)).GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))
}
