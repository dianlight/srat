package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

// LabFeatureHandlerSuite covers GET /lab_features:
//   - alpha features are omitted entirely in production builds
//   - alpha features are present and available in development/pre-release
//   - beta features track experimental_lab_mode
type LabFeatureHandlerSuite struct {
	suite.Suite
	app                *fxtest.App
	handler            *api.LabFeatureHandler
	mockSettingService service.SettingServiceInterface
	ctx                context.Context
	cancel             context.CancelFunc
}

func TestLabFeatureHandlerSuite(t *testing.T) { suite.Run(t, new(LabFeatureHandlerSuite)) }

func (suite *LabFeatureHandlerSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, &sync.WaitGroup{}))
			},
			api.NewLabFeatureHandler,
			mock.Mock[service.SettingServiceInterface],
		),
		fx.Populate(&suite.handler),
		fx.Populate(&suite.mockSettingService),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
	)
	suite.app.RequireStart()
}

func (suite *LabFeatureHandlerSuite) TearDownTest() {
	config.Version = "0.0.0-dev.0"
	if suite.cancel != nil {
		suite.cancel()
		if wg, ok := suite.ctx.Value(ctxkeys.WaitGroup).(*sync.WaitGroup); ok {
			wg.Wait()
		}
	}
	if suite.app != nil {
		suite.app.RequireStop()
	}
}

func (suite *LabFeatureHandlerSuite) labMode(on bool) {
	mock.When(suite.mockSettingService.Load()).ThenReturn(&dto.Settings{
		ExperimentalLabMode: on,
	}, nil)
}

// fetch calls GET /lab_features through huma and returns the parsed body.
func (suite *LabFeatureHandlerSuite) fetch() []dto.LabFeature {
	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterLabFeatureHandler(apiInst)

	resp := apiInst.Get("/lab_features")
	suite.Require().Equal(http.StatusOK, resp.Code, resp.Body.String())

	var out []dto.LabFeature
	suite.Require().NoError(json.Unmarshal(resp.Body.Bytes(), &out))
	return out
}

func (suite *LabFeatureHandlerSuite) byKey(features []dto.LabFeature) map[string]dto.LabFeature {
	m := make(map[string]dto.LabFeature, len(features))
	for _, f := range features {
		m[f.Key] = f
	}
	return m
}

// =============================================================================
// Environment filtered
// =============================================================================

// TestProductionOmitsAlphaEvenWithLabModeOn is the core release guarantee:
// alpha features must never be exposed to the frontend in a production build,
// no matter what experimental_lab_mode says.
func (suite *LabFeatureHandlerSuite) TestProductionOmitsAlphaEvenWithLabModeOn() {
	config.Version = "1.0.0"
	suite.labMode(true)

	features := suite.byKey(suite.fetch())
	suite.NotContains(features, "ha_custom_component")
	suite.True(features["hdidle"].Available)
	suite.True(features["smb_conf"].Available)
}

// TestProductionOmitsAlphaWithLabModeOff also verifies beta availability
// follows experimental_lab_mode in production builds.
func (suite *LabFeatureHandlerSuite) TestProductionOmitsAlphaWithLabModeOff() {
	config.Version = "1.0.0"
	suite.labMode(false)

	features := suite.byKey(suite.fetch())
	suite.NotContains(features, "ha_custom_component")
	suite.Contains(features, "hdidle")
	suite.Contains(features, "smb_conf")
	suite.False(features["hdidle"].Available)
	suite.False(features["smb_conf"].Available)
}

// TestPrereleaseIncludesAlpha projects alpha features in non-production
// builds and forces available=true (alpha availability is not
// experimental_lab_mode-gated — the build tier is the only gate).
func (suite *LabFeatureHandlerSuite) TestPrereleaseIncludesAlpha() {
	config.Version = "1.0.0-rc.1"
	suite.labMode(false)

	features := suite.byKey(suite.fetch())
	suite.Contains(features, "ha_custom_component")
	suite.True(features["ha_custom_component"].Available)
	suite.Equal("alpha", features["ha_custom_component"].Status)
	suite.Equal("beta", features["hdidle"].Status)
	suite.False(features["hdidle"].Available, "beta availability still follows lab mode")
}

// =============================================================================
// Error path
// =============================================================================

func (suite *LabFeatureHandlerSuite) TestSettingsLoadErrorReturns500() {
	config.Version = "1.0.0"
	mock.When(suite.mockSettingService.Load()).
		ThenReturn(nil, errors.New("storage unavailable"))

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterLabFeatureHandler(apiInst)

	resp := apiInst.Get("/lab_features")
	suite.Equal(http.StatusInternalServerError, resp.Code)
}
