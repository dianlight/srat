package service_test

import (
	"context"
	"log"
	"os"
	"runtime/debug"
	"sync"
	"testing"

	"github.com/angusgmorrison/logfusc"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type SettingServiceSuite struct {
	suite.Suite
	settingService service.SettingServiceInterface
	//propertyRepo   repository.PropertyRepositoryInterface
	app       *fxtest.App
	testMutex sync.Mutex
}

func TestSettingServiceSuite(t *testing.T) {
	suite.Run(t, new(SettingServiceSuite))
}

func (suite *SettingServiceSuite) SetupTest() {

	defer func() {
		if r := recover(); r != nil {
			log.Fatalf("Panic in SetupTest: %#+v\n%s", r, string(debug.Stack()))
		}
	}()

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) { return context.WithCancel(context.Background()) },
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
			service.NewSettingService,
			events.NewEventBus,
			//	repository.NewPropertyRepositoryRepository,
			mock.Mock[service.TelemetryServiceInterface],
		),
		fx.Populate(&suite.settingService),
		//fx.Populate(&suite.propertyRepo),
	)
	suite.app.RequireStart()
}

func (suite *SettingServiceSuite) TearDownTest() {
	suite.testMutex.Lock()
	defer suite.testMutex.Unlock()

	// Reset command exists to default
	suite.settingService.SetCommandExists(nil)

	suite.app.RequireStop()
}

func (suite *SettingServiceSuite) TestUpdateSettings_HAUseNFS() {

	testCases := []struct {
		name            string
		settingsFactory func() dto.Settings
		verifyFunc      func(*dto.Settings, error)
	}{
		{
			name: "HAUseNFS",
			settingsFactory: func() dto.Settings {
				suite.settingService.SetCommandExists(func(cmd []string) bool {
					// Simulate that exportfs command exists
					return true
				})
				return dto.Settings{HAUseNFS: new(true)}
			},
			verifyFunc: func(loaded *dto.Settings, err error) {
				suite.Require().NoError(err)
				suite.True(*loaded.HAUseNFS)
			},
		},
		{
			name: "HAUseNFS_ExportfsNotExists",
			settingsFactory: func() dto.Settings {
				suite.settingService.SetCommandExists(func(cmd []string) bool {
					// Simulate that exportfs command does not exist
					return false
				})
				return dto.Settings{HAUseNFS: new(true)}
			},
			verifyFunc: func(loaded *dto.Settings, err error) {
				suite.Require().NoError(err)
				suite.False(*loaded.HAUseNFS)
			},
		},
		{
			name: "HAUseNFS_False",
			settingsFactory: func() dto.Settings {
				suite.settingService.SetCommandExists(func(cmd []string) bool {
					// Simulate that exportfs command exists
					return true
				})
				return dto.Settings{HAUseNFS: new(false)}
			},
			verifyFunc: func(loaded *dto.Settings, err error) {
				suite.Require().NoError(err)
				suite.False(*loaded.HAUseNFS)
			},
		},
		{
			name: "HAUseNFS_Nil",
			settingsFactory: func() dto.Settings {
				suite.settingService.SetCommandExists(func(cmd []string) bool {
					// Simulate that exportfs command exists
					return true
				})
				return dto.Settings{HAUseNFS: nil}
			},
			verifyFunc: func(loaded *dto.Settings, err error) {
				suite.Require().NoError(err)
				suite.NotNil(loaded.HAUseNFS, "HAUseNFS should not be nil but defaulted by tag")
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Reset database state for each sub-test
			suite.TearDownTest()
			suite.SetupTest()
			suite.testFieldUpdateAndLoad(tc.name, tc.settingsFactory, tc.verifyFunc)
		})
	}
}

// testFieldUpdateAndLoad is a generic helper function that tests updating and loading a specific field
// It accepts a field name, a function to create settings with the field value, and a verification function
// The function includes panic recovery with detailed error reporting
func (suite *SettingServiceSuite) testFieldUpdateAndLoad(
	fieldName string,
	settingsFactory func() dto.Settings,
	verifyFunc func(*dto.Settings, error),
) {
	suite.testMutex.Lock()
	defer suite.testMutex.Unlock()
	defer func() {
		if r := recover(); r != nil {
			suite.Failf("Panic occurred during test",
				"Field: %s, Panic: %v", fieldName, r)
		}
	}()

	// Create and update settings
	testSettings := settingsFactory()
	err := suite.settingService.UpdateSettings(&testSettings)
	suite.Require().NoError(err, "UpdateSettings should not fail for field: %s", fieldName)

	// Dump table for debugging
	tableDump, dumpErr := suite.settingService.DumpTable()
	suite.Require().NoError(dumpErr, "DumpTable should not fail for field: %s", fieldName)
	suite.T().Logf("Property Table Dump after updating field %s:\n%s", fieldName, tableDump)

	// Load and verify
	loaded, loadErr := suite.settingService.Load()
	verifyFunc(loaded, loadErr)
}

// TestUpdateSettings_SaveAndLoad_AllFieldTypes tests saving, loading, and modifying all Settings field types
func (suite *SettingServiceSuite) TestUpdateSettings_SaveAndLoad_AllFieldTypes() {
	testCases := []struct {
		name            string
		settingsFactory func() dto.Settings
		verifyFunc      func(*dto.Settings, error)
	}{
		{
			name: "Workgroup_String",
			settingsFactory: func() dto.Settings {
				return dto.Settings{Workgroup: "TESTWORKGROUP"}
			},
			verifyFunc: func(loaded *dto.Settings, err error) {
				suite.Require().NoError(err)
				suite.Equal("TESTWORKGROUP", loaded.Workgroup, "Workgroup should match the set value")
			},
		},
		{
			name: "HAUseNFS_True",
			settingsFactory: func() dto.Settings {
				suite.settingService.SetCommandExists(func(cmd []string) bool {
					// Simulate that exportfs command exists
					return false
				})
				return dto.Settings{HAUseNFS: new(true)}
			},
			verifyFunc: func(loaded *dto.Settings, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(loaded.HAUseNFS)
				suite.False(*loaded.HAUseNFS, "HAUseNFS should be false when exportfs unavailable")
			},
		},
		{
			name: "HAUseNFS_False",
			settingsFactory: func() dto.Settings {
				return dto.Settings{HAUseNFS: new(false)}
			},
			verifyFunc: func(loaded *dto.Settings, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(loaded.HAUseNFS)
				suite.False(*loaded.HAUseNFS)
			},
		},
		{
			name: "Workgroup_NotEmpty",
			settingsFactory: func() dto.Settings {
				return dto.Settings{Workgroup: ""}
			},
			verifyFunc: func(loaded *dto.Settings, err error) {
				suite.Require().NoError(err)
				suite.NotEmpty(loaded.Workgroup)
			},
		},
		{
			name: "HAUseNFS_Nil",
			settingsFactory: func() dto.Settings {
				return dto.Settings{HAUseNFS: nil}
			},
			verifyFunc: func(loaded *dto.Settings, err error) {
				suite.Require().NoError(err)
				suite.NotNil(loaded.HAUseNFS)
			},
		},
		{
			name: "AllowGuest_True",
			settingsFactory: func() dto.Settings {
				return dto.Settings{AllowGuest: new(true)}
			},
			verifyFunc: func(loaded *dto.Settings, err error) {
				suite.Require().NoError(err)
				suite.NotNil(loaded.AllowGuest)
				suite.True(*loaded.AllowGuest)
			},
		},
		{
			name: "AllowGuest_False",
			settingsFactory: func() dto.Settings {
				return dto.Settings{AllowGuest: new(false)}
			},
			verifyFunc: func(loaded *dto.Settings, err error) {
				suite.Require().NoError(err)
				suite.NotNil(loaded.AllowGuest)
				suite.False(*loaded.AllowGuest)
			},
		},
		{
			name: "MultiChannel_True",
			settingsFactory: func() dto.Settings {
				return dto.Settings{MultiChannel: true}
			},
			verifyFunc: func(loaded *dto.Settings, err error) {
				suite.Require().NoError(err)
				suite.True(loaded.MultiChannel)
			},
		},
		{
			name: "Interfaces_Array",
			settingsFactory: func() dto.Settings {
				return dto.Settings{Interfaces: []string{"eth0", "eth1"}}
			},
			verifyFunc: func(loaded *dto.Settings, err error) {
				suite.Require().NoError(err)
				suite.Equal([]string{"eth0", "eth1"}, loaded.Interfaces)
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Reset database state for each sub-test
			suite.TearDownTest()
			suite.SetupTest()
			suite.testFieldUpdateAndLoad(tc.name, tc.settingsFactory, tc.verifyFunc)
		})
	}
}

// TestUpdateSettings_ModifyMultipleFields tests modifying multiple fields in sequence
func (suite *SettingServiceSuite) TestUpdateSettings_ModifyMultipleFields() {
	// Set initial values with multiple fields
	/*
		initialSettings := dto.Settings{
			Workgroup: "INITIAL",
			HAUseNFS:  new(true),
		}
		err := suite.settingService.UpdateSettings(&initialSettings)
		suite.Require().NoError(err)

		loaded, err := suite.settingService.Load()
		suite.Require().NoError(err)
		suite.Equal("INITIAL", loaded.Workgroup)
	*/

	// Modify one field - test that only specified field changes
	testCases := []struct {
		name            string
		settingsFactory func() dto.Settings
		verifyFunc      func(*dto.Settings, error)
	}{
		{
			name: "Modified_Workgroup",
			settingsFactory: func() dto.Settings {
				return dto.Settings{Workgroup: "MODIFIED"}
			},
			verifyFunc: func(loaded *dto.Settings, err error) {
				suite.Require().NoError(err)
				suite.Equal("MODIFIED", loaded.Workgroup)
			},
		},
		{
			name: "Multiple_Fields_Update",
			settingsFactory: func() dto.Settings {
				return dto.Settings{
					Workgroup:    "FINAL",
					MultiChannel: true,
				}
			},
			verifyFunc: func(loaded *dto.Settings, err error) {
				suite.Require().NoError(err)
				suite.Equal("FINAL", loaded.Workgroup)
				suite.True(loaded.MultiChannel)
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Reset database state for each sub-test
			suite.TearDownTest()
			suite.SetupTest()
			suite.testFieldUpdateAndLoad(tc.name, tc.settingsFactory, tc.verifyFunc)
		})
	}
}

func (suite *SettingServiceSuite) TestUpdateSettings_PreservesHASmbPasswordWhenEmpty() {
	suite.testMutex.Lock()
	defer suite.testMutex.Unlock()

	initial := dto.Settings{
		Workgroup:     "INITIAL",
		HASmbPassword: logfusc.NewSecret("super-secret"),
	}
	err := suite.settingService.UpdateSettings(&initial)
	suite.Require().NoError(err)

	update := dto.Settings{
		Workgroup: "UPDATED",
	}
	err = suite.settingService.UpdateSettings(&update)
	suite.Require().NoError(err)

	loaded, loadErr := suite.settingService.Load()
	suite.Require().NoError(loadErr)
	suite.Equal("UPDATED", loaded.Workgroup)
	suite.Equal("super-secret", loaded.HASmbPassword.Expose())
}

// TestUpdateSettings_NilPointerFields tests handling of nil pointer fields
func (suite *SettingServiceSuite) TestUpdateSettings_NilPointerFields() {
	testCases := []struct {
		name            string
		settingsFactory func() dto.Settings
		verifyFunc      func(*dto.Settings, error)
	}{
		{
			name: "HAUseNFS_Nil",
			settingsFactory: func() dto.Settings {
				return dto.Settings{HAUseNFS: nil}
			},
			verifyFunc: func(loaded *dto.Settings, err error) {
				suite.Require().NoError(err)
				suite.NotNil(loaded.HAUseNFS)
			},
		},
		{
			name: "AllowGuest_Nil",
			settingsFactory: func() dto.Settings {
				return dto.Settings{AllowGuest: nil}
			},
			verifyFunc: func(loaded *dto.Settings, err error) {
				suite.Require().NoError(err)
				suite.NotNil(loaded.AllowGuest)
			},
		},
		{
			name: "LocalMaster_Nil",
			settingsFactory: func() dto.Settings {
				return dto.Settings{LocalMaster: nil}
			},
			verifyFunc: func(loaded *dto.Settings, err error) {
				suite.Require().NoError(err)
				suite.NotNil(loaded.LocalMaster)
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Reset database state for each sub-test
			suite.TearDownTest()
			suite.SetupTest()
			suite.testFieldUpdateAndLoad(tc.name, tc.settingsFactory, tc.verifyFunc)
		})
	}
}
