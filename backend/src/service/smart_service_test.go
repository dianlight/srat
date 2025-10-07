package service

import (
	"os"
	"testing"

	"github.com/dianlight/srat/dto"
	gocache "github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
)

type SmartServiceSuite struct {
	suite.Suite
	service SmartServiceInterface
}

func (suite *SmartServiceSuite) SetupTest() {
	suite.service = NewSmartService()
}

func (suite *SmartServiceSuite) TearDownTest() {
}

func (suite *SmartServiceSuite) TestGetSmartInfoCacheHit() {
	// Setup: Manually set cache
	expectedInfo := &dto.SmartInfo{DiskType: "SATA"}
	cacheKey := smartCacheKeyPrefix + "/dev/sda"
	suite.service.(*smartService).cache.Set(cacheKey, expectedInfo, gocache.DefaultExpiration)

	// Execute
	info, err := suite.service.GetSmartInfo("/dev/sda")

	// Assert
	suite.NoError(err)
	suite.Equal(expectedInfo, info)
}

func (suite *SmartServiceSuite) TestGetSmartInfoDeviceNotExist() {
	// Execute with invalid path
	info, err := suite.service.GetSmartInfo("/dev/nonexistent")

	// Assert
	suite.Error(err)
	suite.Nil(info)
	suite.True(errors.Is(err, dto.ErrorSMARTNotSupported))
	// Verify details
	details := errors.Details(err)
	suite.Equal("/dev/nonexistent", details["device"])
	suite.Equal("does not exist", details["reason"])
}

func (suite *SmartServiceSuite) TestGetSmartInfoDeviceNotReadable() {
	// Create a temp file and remove read permission
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())
	os.Chmod(tempFile.Name(), 0000)
	defer os.Chmod(tempFile.Name(), 0644) // Restore for cleanup

	// Execute
	info, err := suite.service.GetSmartInfo(tempFile.Name())

	// Assert
	suite.Error(err)
	suite.Nil(info)
	suite.True(errors.Is(err, dto.ErrorSMARTNotSupported))
	details := errors.Details(err)
	suite.Equal("not readable", details["reason"])
}

func TestSmartServiceSuite(t *testing.T) {
	suite.Run(t, new(SmartServiceSuite))
}
