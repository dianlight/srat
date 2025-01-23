package dbom

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestExportedSharesLoad(t *testing.T) {
	// Create test data
	share1 := ExportedShare{Name: "TestShare1", MountPointData: MountPointData{Path: "/test/path1"}}
	share2 := ExportedShare{Name: "TestShare2", MountPointData: MountPointData{Path: "/test/path2"}}
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
	require.NoError(t, err)
	assert.Len(t, shares, 2)
	assert.Equal(t, "TestShare1", shares[0].Name)
	assert.Equal(t, "TestShare2", shares[1].Name)
	assert.Len(t, shares[0].Users, 1)
	assert.Len(t, shares[1].Users, 1)
	assert.Equal(t, "TestUser1", shares[0].Users[0].Username)
	assert.Equal(t, "TestUser2", shares[1].Users[0].Username)

	// Clean up
	require.NoError(t, db.Unscoped().Delete(&share1).Error)
	require.NoError(t, db.Unscoped().Delete(&share2).Error)
	require.NoError(t, db.Unscoped().Delete(&user1).Error)
	require.NoError(t, db.Unscoped().Delete(&user2).Error)
}

/*
	func TestExportedSharesLoadPreloadsAssociations(t *testing.T) {
		// Create test data
		share := ExportedShare{Name: "TestShare"}
		user := SambaUser{Username: "TestUser"}
		roUser := SambaUser{Username: "TestROUser"}
		mountPointData := MountPointData{Path: "/test/path"}

		require.NoError(t, db.Create(&user).Error)
		require.NoError(t, db.Create(&roUser).Error)
		require.NoError(t, db.Create(&mountPointData).Error)
		require.NoError(t, db.Create(&share).Error)

		// Associate data
		db.Model(&share).Association("Users").Append(&user)
		db.Model(&share).Association("RoUsers").Append(&roUser)
		share.MountPointDataID = mountPointData.ID
		db.Save(&share)

		// Test Load function
		var shares ExportedShares
		err := shares.Load()
		require.NoError(t, err)

		// Assert
		assert.Len(t, shares, 1)
		assert.Equal(t, "TestShare", shares[0].Name)
		assert.Len(t, shares[0].Users, 1)
		assert.Equal(t, "TestUser", shares[0].Users[0].Username)
		assert.Len(t, shares[0].RoUsers, 1)
		assert.Equal(t, "TestROUser", shares[0].RoUsers[0].Username)
		assert.Equal(t, mountPointData.ID, shares[0].MountPointDataID)
		assert.Equal(t, "/test/path", shares[0].MountPointData.Path)

		// Clean up
		require.NoError(t, db.Unscoped().Delete(&shares[0]).Error)
		require.NoError(t, db.Unscoped().Delete(&user).Error)
		require.NoError(t, db.Unscoped().Delete(&roUser).Error)
		require.NoError(t, db.Unscoped().Delete(&mountPointData).Error)
	}
*/
func TestExportedSharesSave(t *testing.T) {
	// Create test data
	share1 := ExportedShare{Name: "TestShare11", MountPointData: MountPointData{Path: "/test/path12"}}
	share2 := ExportedShare{Name: "TestShare21", MountPointData: MountPointData{Path: "/test/path22"}}

	// Save the shares
	err := share1.MountPointData.Save()
	share1.MountPointDataID = share1.MountPointData.ID
	require.NoError(t, err)
	err = share2.MountPointData.Save()
	share2.MountPointDataID = share2.MountPointData.ID
	require.NoError(t, err)
	shares := ExportedShares{share1, share2}
	err = shares.Save()

	// Assert
	require.NoError(t, err)

	// Verify the shares were saved
	var loadedShares ExportedShares
	err = loadedShares.Load()
	require.NoError(t, err)
	assert.Len(t, loadedShares, 2)
	assert.Equal(t, "TestShare11", loadedShares[0].Name)
	assert.Equal(t, "TestShare21", loadedShares[1].Name)

	// Clean up
	require.NoError(t, db.Unscoped().Delete(&loadedShares[0]).Error)
	require.NoError(t, db.Unscoped().Delete(&loadedShares[1]).Error)
}

func TestExportedSharesSaveWithExistingMountpointData(t *testing.T) {
	// Create test data
	mountpoint1 := MountPointData{Path: "/test/path120"}
	share1 := ExportedShare{Name: "TestShare110", MountPointData: MountPointData{Path: "/test/path120"}}
	share2 := ExportedShare{Name: "TestShare210", MountPointData: MountPointData{Path: "/test/path220"}}

	// Save the shares
	err := mountpoint1.Save()
	require.NoError(t, err)
	err = share1.MountPointData.Save()
	share1.MountPointDataID = share1.MountPointData.ID
	require.NoError(t, err)
	err = share2.MountPointData.Save()
	share2.MountPointDataID = share2.MountPointData.ID
	require.NoError(t, err)
	shares := ExportedShares{share1, share2}
	err = shares.Save()

	// Assert
	require.NoError(t, err)

	// Verify the shares were saved
	var loadedShares ExportedShares
	err = loadedShares.Load()
	require.NoError(t, err)
	assert.Len(t, loadedShares, 2)
	assert.Equal(t, "TestShare110", loadedShares[0].Name)
	assert.Equal(t, "TestShare210", loadedShares[1].Name)

	// Clean up
	require.NoError(t, db.Unscoped().Delete(&loadedShares[0]).Error)
	require.NoError(t, db.Unscoped().Delete(&loadedShares[1]).Error)
}

func TestExportedSharesUpdateExisting(t *testing.T) {
	// Create initial test data
	share1 := ExportedShare{Name: "TestShare3", MountPointData: MountPointData{Path: "/test/path15"}}
	share2 := ExportedShare{Name: "TestShare4", MountPointData: MountPointData{Path: "/test/path25"}}
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
	initialShare := ExportedShare{Name: "TestShare", MountPointData: MountPointData{Path: "/test/path"}}
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
	initialShare := ExportedShare{Name: "TestShare", MountPointData: MountPointData{Path: "/test/path_12"}}
	err := db.Create(&initialShare).Error
	require.NoError(t, err)

	// Attempt to create another ExportedShare with the same name
	duplicateShare := ExportedShare{Name: "TestShare", MountPointData: MountPointData{Path: "/test/path2"}}
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
		MountPointData: MountPointData{Path: "/test/path_as"},
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
	testShare := ExportedShare{Name: "ConcurrentTestShare", MountPointData: MountPointData{Path: "/test/path_cc"}}
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
