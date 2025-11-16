package service_test

import (
	"context"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type SettingServiceSuite struct {
	suite.Suite
	settingService service.SettingServiceInterface
	propertyRepo   repository.PropertyRepositoryInterface
	app            *fxtest.App
}

func TestSettingServiceSuite(t *testing.T) {
	suite.Run(t, new(SettingServiceSuite))
}

func (suite *SettingServiceSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) { return context.WithCancel(context.Background()) },
			service.NewSettingService,
			events.NewEventBus,
			mock.Mock[repository.PropertyRepositoryInterface],
			mock.Mock[service.TelemetryServiceInterface],
		),
		fx.Populate(&suite.settingService),
		fx.Populate(&suite.propertyRepo),
	)
	suite.app.RequireStart()
}

func (suite *SettingServiceSuite) TearDownTest() {
	suite.app.RequireStop()
}

func (suite *SettingServiceSuite) TestHasValue_ReturnsTrueWhenPresent() {
	mock.When(suite.propertyRepo.Value(mock.Any[string](), mock.Any[bool]())).ThenReturn("some-value", nil)

	has, err := suite.settingService.HasValue("TestKey")

	suite.NoError(err)
	suite.True(has)
	mock.Verify(suite.propertyRepo, matchers.Times(1)).Value(mock.Any[string](), mock.Any[bool]())
}

func (suite *SettingServiceSuite) TestHasValue_ReturnsFalseWhenNotFound() {
	mock.When(suite.propertyRepo.Value(mock.Any[string](), mock.Any[bool]())).ThenReturn(nil, errors.WithStack(dto.ErrorNotFound))

	has, err := suite.settingService.HasValue("MissingKey")

	suite.NoError(err)
	suite.False(has)
}

func (suite *SettingServiceSuite) TestHasValue_PropagatesOtherErrors() {
	mock.When(suite.propertyRepo.Value(mock.Any[string](), mock.Any[bool]())).ThenReturn(nil, errors.New("db error"))

	has, err := suite.settingService.HasValue("AnyKey")

	suite.Error(err)
	suite.False(has)
}

func (suite *SettingServiceSuite) TestHasDefaultValue_ReturnsTrueForExistingField() {
	// Test with json tag name
	has, err := suite.settingService.HasDefaultValue("hostname")
	suite.NoError(err)
	suite.True(has)

	// Test with another field
	has, err = suite.settingService.HasDefaultValue("workgroup")
	suite.NoError(err)
	suite.True(has)

	// Test with field that has enum tag
	has, err = suite.settingService.HasDefaultValue("telemetry_mode")
	suite.NoError(err)
	suite.True(has)
}

func (suite *SettingServiceSuite) TestHasDefaultValue_ReturnsFalseForNonExistentField() {
	has, err := suite.settingService.HasDefaultValue("non_existent_field")
	suite.NoError(err)
	suite.False(has)

	// Test with completely invalid key
	has, err = suite.settingService.HasDefaultValue("invalid_key_123")
	suite.NoError(err)
	suite.False(has)
}

func (suite *SettingServiceSuite) TestHasDefaultValue_ReturnsFalseForEmptyString() {
	has, err := suite.settingService.HasDefaultValue("")
	suite.NoError(err)
	suite.False(has)
}

func (suite *SettingServiceSuite) TestHasDefaultValue_CaseInsensitiveFieldName() {
	// Test with field name (case-insensitive)
	has, err := suite.settingService.HasDefaultValue("Hostname")
	suite.NoError(err)
	suite.True(has)

	has, err = suite.settingService.HasDefaultValue("WORKGROUP")
	suite.NoError(err)
	suite.True(has)
}

func (suite *SettingServiceSuite) TestGetValue_ReturnsValueFromRepository() {
	expectedValue := "test-hostname"
	mock.When(suite.propertyRepo.Value(mock.Any[string](), mock.Any[bool]())).ThenReturn(expectedValue, nil)

	val, err := suite.settingService.GetValue("hostname")

	suite.NoError(err)
	suite.Equal(expectedValue, val)
	mock.Verify(suite.propertyRepo, matchers.Times(1)).Value(mock.Any[string](), mock.Any[bool]())
}

func (suite *SettingServiceSuite) TestGetValue_ReturnsDefaultWhenNotFound() {
	// Mock repository to return NotFound
	mock.When(suite.propertyRepo.Value(mock.Any[string](), mock.Any[bool]())).ThenReturn(nil, errors.WithStack(dto.ErrorNotFound))

	val, err := suite.settingService.GetValue("hostname")

	suite.NoError(err)
	// The default value should be returned (from default_config.json)
	suite.NotNil(val)
}

func (suite *SettingServiceSuite) TestGetValue_ReturnsErrorWhenNotFoundAndNoDefault() {
	// Mock repository to return NotFound
	mock.When(suite.propertyRepo.Value(mock.Any[string](), mock.Any[bool]())).ThenReturn(nil, errors.WithStack(dto.ErrorNotFound))

	val, err := suite.settingService.GetValue("non_existent_field")

	suite.Error(err)
	suite.True(errors.Is(err, dto.ErrorNotFound))
	suite.Nil(val)
}

func (suite *SettingServiceSuite) TestGetValue_PropagatesOtherErrors() {
	// Mock repository to return a database error
	mock.When(suite.propertyRepo.Value(mock.Any[string](), mock.Any[bool]())).ThenReturn(nil, errors.New("database error"))

	val, err := suite.settingService.GetValue("hostname")

	suite.Error(err)
	suite.Nil(val)
	// Should not attempt to get default value when there's a real error
}

func (suite *SettingServiceSuite) TestGetValue_ReturnsErrorForEmptyProperty() {
	val, err := suite.settingService.GetValue("")

	suite.Error(err)
	suite.Nil(val)
	suite.Contains(err.Error(), "property name cannot be empty")
}

func (suite *SettingServiceSuite) TestGetValue_WorksWithDifferentTypes() {
	// Test with boolean value
	mock.When(suite.propertyRepo.Value("compatibility_mode", true)).ThenReturn(true, nil)
	val, err := suite.settingService.GetValue("compatibility_mode")
	suite.NoError(err)
	suite.Equal(true, val)

	// Test with string slice
	expectedSlice := []string{"eth0", "eth1"}
	mock.When(suite.propertyRepo.Value("interfaces", true)).ThenReturn(expectedSlice, nil)
	val, err = suite.settingService.GetValue("interfaces")
	suite.NoError(err)
	suite.Equal(expectedSlice, val)
}

func (suite *SettingServiceSuite) TestSetValue_SucceedsWithCompatibleType() {
	// Mock existing value
	mock.When(suite.propertyRepo.Value("hostname", true)).ThenReturn("old-hostname", nil)
	mock.When(suite.propertyRepo.SetValue("hostname", "new-hostname")).ThenReturn(nil)

	err := suite.settingService.SetValue("hostname", "new-hostname")

	suite.NoError(err)
	mock.Verify(suite.propertyRepo, matchers.Times(1)).SetValue("hostname", "new-hostname")
}

func (suite *SettingServiceSuite) TestSetValue_FailsWithIncompatibleExistingType() {
	// Mock existing value as string
	mock.When(suite.propertyRepo.Value("hostname", true)).ThenReturn("old-hostname", nil)

	// Try to set as integer
	err := suite.settingService.SetValue("hostname", 123)

	suite.Error(err)
	suite.Contains(err.Error(), "type mismatch")
	suite.Contains(err.Error(), "existing type")
}

func (suite *SettingServiceSuite) TestSetValue_UsesDefaultTypeWhenNoExistingValue() {
	// Mock repository to return NotFound (no existing value)
	mock.When(suite.propertyRepo.Value("hostname", true)).ThenReturn(nil, errors.WithStack(dto.ErrorNotFound))
	mock.When(suite.propertyRepo.SetValue("hostname", "new-hostname")).ThenReturn(nil)

	// hostname has a string default, so string should be accepted
	err := suite.settingService.SetValue("hostname", "new-hostname")

	suite.NoError(err)
	mock.Verify(suite.propertyRepo, matchers.Times(1)).SetValue("hostname", "new-hostname")
}

func (suite *SettingServiceSuite) TestSetValue_FailsWithIncompatibleDefaultType() {
	// Mock repository to return NotFound (no existing value)
	mock.When(suite.propertyRepo.Value("hostname", true)).ThenReturn(nil, errors.WithStack(dto.ErrorNotFound))

	// hostname has a string default, try to set as boolean
	err := suite.settingService.SetValue("hostname", true)

	suite.Error(err)
	suite.Contains(err.Error(), "type mismatch")
	suite.Contains(err.Error(), "default type")
}

func (suite *SettingServiceSuite) TestSetValue_AllowsAnyTypeForNewPropertyWithoutDefault() {
	// Mock repository to return NotFound (no existing value)
	mock.When(suite.propertyRepo.Value("custom_property", true)).ThenReturn(nil, errors.WithStack(dto.ErrorNotFound))
	mock.When(suite.propertyRepo.SetValue("custom_property", "any-value")).ThenReturn(nil)

	// No default value exists for custom_property, any type should be allowed
	err := suite.settingService.SetValue("custom_property", "any-value")

	suite.NoError(err)
	mock.Verify(suite.propertyRepo, matchers.Times(1)).SetValue("custom_property", "any-value")
}

func (suite *SettingServiceSuite) TestSetValue_RejectsNilValue() {
	err := suite.settingService.SetValue("hostname", nil)

	suite.Error(err)
	suite.Contains(err.Error(), "value cannot be nil")
}

func (suite *SettingServiceSuite) TestSetValue_RejectsEmptyProperty() {
	err := suite.settingService.SetValue("", "some-value")

	suite.Error(err)
	suite.Contains(err.Error(), "property name cannot be empty")
}

func (suite *SettingServiceSuite) TestSetValue_PropagatesRepositoryErrors() {
	// Mock repository to return a database error
	mock.When(suite.propertyRepo.Value("hostname", true)).ThenReturn(nil, errors.New("database error"))

	err := suite.settingService.SetValue("hostname", "new-hostname")

	suite.Error(err)
	suite.Contains(err.Error(), "database error")
}

func (suite *SettingServiceSuite) TestSetValue_WorksWithBooleanTypes() {
	// Mock existing boolean value
	mock.When(suite.propertyRepo.Value("compatibility_mode", true)).ThenReturn(true, nil)
	mock.When(suite.propertyRepo.SetValue("compatibility_mode", false)).ThenReturn(nil)

	err := suite.settingService.SetValue("compatibility_mode", false)

	suite.NoError(err)
	mock.Verify(suite.propertyRepo, matchers.Times(1)).SetValue("compatibility_mode", false)
}

func (suite *SettingServiceSuite) TestSetValue_WorksWithSliceTypes() {
	// Mock existing slice value
	oldSlice := []string{"eth0"}
	newSlice := []string{"eth0", "eth1"}
	mock.When(suite.propertyRepo.Value("interfaces", true)).ThenReturn(oldSlice, nil)
	mock.When(suite.propertyRepo.SetValue("interfaces", newSlice)).ThenReturn(nil)

	err := suite.settingService.SetValue("interfaces", newSlice)

	suite.NoError(err)
	mock.Verify(suite.propertyRepo, matchers.Times(1)).SetValue("interfaces", newSlice)
}

func (suite *SettingServiceSuite) TestSetValue_RejectsIncompatibleSliceElementTypes() {
	// Mock existing string slice
	mock.When(suite.propertyRepo.Value("interfaces", true)).ThenReturn([]string{"eth0"}, nil)

	// Try to set as int slice
	err := suite.settingService.SetValue("interfaces", []int{1, 2, 3})

	suite.Error(err)
	suite.Contains(err.Error(), "type mismatch")
}

func (suite *SettingServiceSuite) TestSetValue_WorksWithPointerTypes() {
	// Mock existing pointer value
	trueVal := true
	falseVal := false
	mock.When(suite.propertyRepo.Value("local_master", true)).ThenReturn(&trueVal, nil)
	mock.When(suite.propertyRepo.SetValue("local_master", &falseVal)).ThenReturn(nil)

	err := suite.settingService.SetValue("local_master", &falseVal)

	suite.NoError(err)
	mock.Verify(suite.propertyRepo, matchers.Times(1)).SetValue("local_master", &falseVal)
}
