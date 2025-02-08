package dbom

import (
	"fmt"
	"sync"
	"testing"

	//"github.com/dianlight/srat/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type ExportedSharesSuite struct {
	suite.Suite
	//mount_repo repository.MountPointPathRepositoryInterface
	//mockBoradcaster *MockBroadcasterServiceInterface
	// VariableThatShouldStartAtFive int
}

func TestExportedSharesSuite(t *testing.T) {
	csuite := new(ExportedSharesSuite)
	//csuite.mount_repo = repository.NewMountPointPathRepository(GetDB())
	/*
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		csuite.mockBoradcaster = NewMockBroadcasterServiceInterface(ctrl)
		csuite.mockBoradcaster.EXPECT().AddOpenConnectionListener(gomock.Any()).AnyTimes()
		csuite.mockBoradcaster.EXPECT().BroadcastMessage(gomock.Any()).AnyTimes()
	*/
	suite.Run(t, csuite)
}

func (suite *ExportedSharesSuite) TestExportedSharesLoad() {
	// Create test data
	share1 := ExportedShare{Name: "TestShare1", MountPointData: MountPointPath{Path: "/test/path1"}}
	share2 := ExportedShare{Name: "TestShare2", MountPointData: MountPointPath{Path: "/test/path2"}}
	db.Create(&share1)
	db.Create(&share2)

	// Create test users
	user1 := SambaUser{Username: "TestUser1"}
	user2 := SambaUser{Username: "TestUser2"}
	db.Create(&user1)
	db.Create(&user2)

	// Associate users with shares
	db.Model(&share1).Association("Users").Append(&user1)
	db.Model(&share2).Association("Users").Append(&user2)

	// Test Load function
	var shares ExportedShares
	err := shares.Load()

	// Assert
	suite.Require().NoError(err)
	suite.Len(shares, 2)
	suite.Equal("TestShare1", shares[0].Name)
	suite.Equal("TestShare2", shares[1].Name)
	suite.Len(shares[0].Users, 1)
	suite.Len(shares[1].Users, 1)
	suite.Equal("TestUser1", shares[0].Users[0].Username)
	suite.Equal("TestUser2", shares[1].Users[0].Username)

	// Clean up
	suite.NoError(db.Unscoped().Delete(&share1).Error)
	suite.NoError(db.Unscoped().Delete(&share2).Error)
	suite.NoError(db.Unscoped().Delete(&user1).Error)
	suite.NoError(db.Unscoped().Delete(&user2).Error)
}

func (suite *ExportedSharesSuite) TestExportedSharesSave() {
	// Create test data
	share1 := ExportedShare{Name: "TestShare11", MountPointData: MountPointPath{Path: "/test/path12"}}
	share2 := ExportedShare{Name: "TestShare21", MountPointData: MountPointPath{Path: "/test/path22"}}

	// Save the shares
	//err := share1.MountPointData.Save()
	//share1.MountPointDataID = share1.MountPointData.ID
	//require.NoError(t, err)
	//err = share2.MountPointData.Save()
	//share2.MountPointDataID = share2.MountPointData.ID
	//require.NoError(t, err)
	shares := ExportedShares{share1, share2}
	err := shares.Save()

	// Assert
	suite.Require().NoError(err)

	// Verify the shares were saved
	var loadedShares ExportedShares
	err = loadedShares.Load()
	suite.Require().NoError(err)
	suite.Len(loadedShares, 2)
	suite.Equal("TestShare11", loadedShares[0].Name)
	suite.Equal("TestShare21", loadedShares[1].Name)

	// Clean up
	suite.NoError(db.Unscoped().Delete(&loadedShares[0]).Error)
	suite.NoError(db.Unscoped().Delete(&loadedShares[1]).Error)
}

/*
func (suite *ExportedSharesSuite) TestExportedSharesSaveWithExistingMountpointData() {
	// Create test data
	//mountpoint1 := MountPointPath{Path: "/test/path120"}
	//err := suite.mount_repo.Save(&mountpoint1)
	share1 := ExportedShare{Name: "TestShare110", MountPointData: MountPointPath{Path: "/mnt/LIBRARY"}}
	share2 := ExportedShare{Name: "TestShare210", MountPointData: MountPointPath{Path: "/mnt/LIBRARY"}}

	// Save the shares
	/*
		err := mountpoint1.Save()
		require.NoError(t, err)
		err = share1.MountPointData.Save()
		share1.MountPointDataID = share1.MountPointData.ID
		require.NoError(t, err)
		err = share2.MountPointData.Save()
		share2.MountPointDataID = share2.MountPointData.ID
		require.NoError(t, err)
	* /
	shares := ExportedShares{share1, share2}
	err := shares.Save()

	// Assert
	suite.NoError(err)

	// Verify the shares were saved
	var loadedShares ExportedShares
	err = loadedShares.Load()
	suite.NoError(err)
	suite.Len(loadedShares, 2)
	suite.Equal("TestShare110", loadedShares[0].Name)
	suite.Equal("TestShare210", loadedShares[1].Name)

	// Clean up
	suite.NoError(db.Unscoped().Delete(&loadedShares[0]).Error)
	suite.NoError(db.Unscoped().Delete(&loadedShares[1]).Error)
}
*/

func TestExportedSharesUpdateExisting(t *testing.T) {
	// Create initial test data
	share1 := ExportedShare{Name: "TestShare3", MountPointData: MountPointPath{Path: "/test/path15"}}
	share2 := ExportedShare{Name: "TestShare4", MountPointData: MountPointPath{Path: "/test/path25"}}
	initialShares := ExportedShares{share1, share2}
	err := initialShares.Save()
	require.NoError(t, err)

	// Modify the shares
	initialShares[0].Name = "UpdatedShare1"
	initialShares[1].TimeMachine = true
	updatedShares := ExportedShares{
		initialShares[0],
		initialShares[1],
	}

	// Save the updated shares
	err = updatedShares.Save()
	require.NoError(t, err)

	// Verify the shares were updated
	var loadedShares ExportedShares
	err = loadedShares.Load()
	require.NoError(t, err)
	assert.Len(t, loadedShares, 2)
	assert.Equal(t, "UpdatedShare1", loadedShares[0].Name)
	assert.True(t, loadedShares[1].TimeMachine)

	// Clean up
	db.Unscoped().Delete(&initialShares)
}
func TestExportedSharesSaveWithInvalidData(t *testing.T) {
	// Create an ExportedShare with an invalid Name (empty string)
	invalidShare := ExportedShare{Name: ""}
	shares := ExportedShares{invalidShare}

	// Attempt to save the shares
	err := shares.Save()

	// Assert that an error was returned
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid name")

	// Verify that no shares were saved
	var loadedShares ExportedShares
	loadErr := loadedShares.Load()
	require.NoError(t, loadErr)
	assert.Empty(t, loadedShares)
}
func TestExportedShareSaveUpdatesExisting(t *testing.T) {
	// Create an initial ExportedShare
	initialShare := ExportedShare{Name: "TestShare", MountPointData: MountPointPath{Path: "/test/path"}}
	err := db.Create(&initialShare).Error
	require.NoError(t, err)

	// Modify the share
	initialShare.Name = "UpdatedTestShare"

	// Save the modified share
	err = initialShare.Save()
	require.NoError(t, err)

	// Retrieve the share from the database
	var retrievedShare ExportedShare
	err = db.First(&retrievedShare, initialShare.ID).Error
	require.NoError(t, err)

	// Assert that the share was updated, not created anew
	assert.Equal(t, initialShare.ID, retrievedShare.ID)
	assert.Equal(t, "UpdatedTestShare", retrievedShare.Name)

	// Verify that no new record was created
	var count int64
	db.Model(&ExportedShare{}).Count(&count)
	assert.Equal(t, int64(1), count)

	// Clean up
	db.Unscoped().Delete(&initialShare)
}
func TestExportedShareSaveWithDuplicateName(t *testing.T) {
	// Create an initial ExportedShare
	initialShare := ExportedShare{Name: "TestShare", MountPointData: MountPointPath{Path: "/test/path_12"}}
	err := db.Create(&initialShare).Error
	require.NoError(t, err)

	// Attempt to create another ExportedShare with the same name
	duplicateShare := ExportedShare{Name: "TestShare", MountPointData: MountPointPath{Path: "/test/path2"}}
	err = duplicateShare.Save()

	// Assert that an error was returned
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicated key")

	// Verify that only one share exists in the database
	var count int64
	db.Model(&ExportedShare{}).Count(&count)
	assert.Equal(t, int64(1), count)

	// Clean up
	db.Unscoped().Delete(&initialShare)
}
func TestExportedShareSaveWithAssociations(t *testing.T) {
	// Create test users
	user1 := SambaUser{Username: "TestUser1"}
	user2 := SambaUser{Username: "TestUser2"}
	db.Create(&user1)
	db.Create(&user2)

	// Create an ExportedShare with associated users
	share := ExportedShare{
		Name:           "TestShareWithUsers",
		MountPointData: MountPointPath{Path: "/test/path_as"},
		Users:          []SambaUser{user1, user2},
	}

	// Save the share
	err := share.Save()
	require.NoError(t, err)

	// Retrieve the share from the database
	var retrievedShare ExportedShare
	err = db.Preload("Users").First(&retrievedShare, share.ID).Error
	require.NoError(t, err)

	// Assert that the share was saved correctly
	assert.Equal(t, "TestShareWithUsers", retrievedShare.Name)
	assert.Len(t, retrievedShare.Users, 2)
	assert.Equal(t, "TestUser1", retrievedShare.Users[0].Username)
	assert.Equal(t, "TestUser2", retrievedShare.Users[1].Username)

	// Clean up
	db.Unscoped().Delete(&share)
	db.Unscoped().Delete(&user1)
	db.Unscoped().Delete(&user2)
}
func TestExportedShareGetNonExistent(t *testing.T) {
	// Create a new ExportedShare with a non-existent ID
	nonExistentShare := ExportedShare{ID: 9999}

	// Attempt to get the non-existent share
	err := nonExistentShare.Get()

	// Assert that an error was returned
	require.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)

	// Assert that the share's fields are still empty
	assert.Equal(t, uint(9999), nonExistentShare.ID)
	assert.Empty(t, nonExistentShare.Name)
	assert.Empty(t, nonExistentShare.Users)
	assert.Empty(t, nonExistentShare.RoUsers)
}

func TestExportedShareGetConcurrent(t *testing.T) {
	// Create a test share
	testShare := ExportedShare{Name: "ConcurrentTestShare", MountPointData: MountPointPath{Path: "/test/path_cc"}}
	err := db.Create(&testShare).Error
	require.NoError(t, err)

	// Number of concurrent operations
	concurrentOps := 10

	// Create a wait group to synchronize goroutines
	var wg sync.WaitGroup
	wg.Add(concurrentOps)

	// Create a channel to collect errors
	errChan := make(chan error, concurrentOps)

	// Perform concurrent Get operations
	for i := 0; i < concurrentOps; i++ {
		go func() {
			defer wg.Done()
			share := ExportedShare{ID: testShare.ID}
			err := share.Get()
			if err != nil {
				errChan <- err
			} else if share.Name != "ConcurrentTestShare" {
				errChan <- fmt.Errorf("unexpected share name: %s", share.Name)
			}
		}()
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(errChan)

	// Check for any errors
	for err := range errChan {
		t.Errorf("concurrent Get operation failed: %v", err)
	}

	// Clean up
	db.Unscoped().Delete(&testShare)
}
