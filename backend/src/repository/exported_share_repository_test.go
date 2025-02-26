package repository_test

import (
	"errors"
	"testing"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/repository"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type ExportedSharesRepositorySuite struct {
	suite.Suite
	export_share_repo repository.ExportedShareRepositoryInterface
}

func TestExportedSharesSuite(t *testing.T) {
	csuite := new(ExportedSharesRepositorySuite)
	csuite.export_share_repo = repository.NewExportedShareRepository(dbom.GetDB())
	suite.Run(t, csuite)
}

func (suite *ExportedSharesRepositorySuite) TestExportedShareRepository_Save() {
	// Arrange
	share := &dbom.ExportedShare{
		Name: "test_share",
		MountPointData: dbom.MountPointPath{
			Path:   "/mnt/test_share",
			Source: "test_source",
			FSType: "ext4",
		},
	}

	// Act
	err := suite.export_share_repo.Save(share)

	// Assert
	suite.Require().NoError(err)

	// Cleanup
	dbom.GetDB().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.ExportedShare{})
	dbom.GetDB().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.MountPointPath{})
}

func (suite *ExportedSharesRepositorySuite) TestExportedShareRepository_SaveAll() {
	// Arrange
	shares := &[]dbom.ExportedShare{
		{
			Name: "test_share1",
			MountPointData: dbom.MountPointPath{
				Path:   "/mnt/test_share1",
				Source: "test_source1",
				FSType: "ext4",
			},
		},
		{
			Name: "test_share2",
			MountPointData: dbom.MountPointPath{
				Path:   "/mnt/test_share2",
				Source: "test_source2",
				FSType: "ntfs",
			},
		},
	}

	// Act
	err := suite.export_share_repo.SaveAll(shares)

	// Assert
	suite.Require().NoError(err)
	// Cleanup
	dbom.GetDB().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.ExportedShare{})
	dbom.GetDB().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.MountPointPath{})
}

func (suite *ExportedSharesRepositorySuite) TestExportedShareRepository_FindByName() {
	// Arrange
	share := &dbom.ExportedShare{
		Name: "find_me",
		MountPointData: dbom.MountPointPath{
			Path:   "/mnt/find_me",
			Source: "find_source",
			FSType: "ext4",
		},
		Users: []dbom.SambaUser{
			{Username: "user1a", Password: "pass1"},
			{Username: "user2a", Password: "pass2"},
		},
		RoUsers: []dbom.SambaUser{
			{Username: "rouser1a", Password: "ropass1"},
			{Username: "rouser2a", Password: "ropass2"},
		},
	}
	err := suite.export_share_repo.Save(share)
	suite.Require().NoError(err)

	// Act
	foundShare, err := suite.export_share_repo.FindByName("find_me")

	// Assert
	suite.Require().NoError(err)
	suite.Require().NotNil(foundShare)
	suite.Equal("find_me", foundShare.Name)
	suite.Len(foundShare.Users, 2)
	suite.Len(foundShare.RoUsers, 2)

	// Cleanup
	dbom.GetDB().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.ExportedShare{})
	dbom.GetDB().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.MountPointPath{})
	dbom.GetDB().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.SambaUser{})
}

func (suite *ExportedSharesRepositorySuite) TestExportedShareRepository_FindByName_NotFound() {
	// Act
	foundShare, err := suite.export_share_repo.FindByName("not_found")

	// Assert
	suite.Require().Error(err)
	suite.True(errors.Is(err, gorm.ErrRecordNotFound))
	suite.Nil(foundShare)
}

func (suite *ExportedSharesRepositorySuite) TestExportedShareRepository_All() {
	// Arrange
	shares := []dbom.ExportedShare{
		{
			Name: "all_share1",
			MountPointData: dbom.MountPointPath{
				Path:   "/mnt/all_share1",
				Source: "all_source1",
				FSType: "ext4",
			},
			Users:   []dbom.SambaUser{},
			RoUsers: []dbom.SambaUser{},
		},
		{
			Name: "all_share2",
			MountPointData: dbom.MountPointPath{
				Path:   "/mnt/all_share2",
				Source: "all_source2",
				FSType: "ntfs",
			},
			Users:   []dbom.SambaUser{},
			RoUsers: []dbom.SambaUser{},
		},
	}
	for _, share := range shares {
		err := suite.export_share_repo.Save(&share)
		suite.Require().NoError(err)
	}

	// Act
	allShares := []dbom.ExportedShare{}
	err := suite.export_share_repo.All(&allShares)

	// Assert
	suite.Require().NoError(err)
	suite.Require().Len(allShares, 2)
	if !cmp.Equal(shares, allShares, cmpopts.IgnoreFields(dbom.ExportedShare{}, "CreatedAt", "UpdatedAt", "DeletedAt", "MountPointDataID", "MountPointData")) {
		suite.Equal(shares, allShares)
	}

	// Cleanup
	dbom.GetDB().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.ExportedShare{})
	dbom.GetDB().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.MountPointPath{})
}
func (suite *ExportedSharesRepositorySuite) TestExportedShareRepository_Delete() {
	// Arrange
	share := &dbom.ExportedShare{
		Name: "delete_me",
		MountPointData: dbom.MountPointPath{
			Path:   "/mnt/delete_me",
			Source: "delete_source",
			FSType: "ext4",
		},
	}
	err := suite.export_share_repo.Save(share)
	suite.Require().NoError(err)

	// Act
	err = suite.export_share_repo.Delete("delete_me")

	// Assert
	suite.Require().NoError(err)

	// Check that it is really deleted
	_, err = suite.export_share_repo.FindByName("delete_me")
	suite.Require().Error(err)
	suite.True(errors.Is(err, gorm.ErrRecordNotFound))

	// Cleanup
	dbom.GetDB().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.MountPointPath{})
}

func (suite *ExportedSharesRepositorySuite) TestExportedShareRepository_UpdateName() {
	// Arrange
	share := &dbom.ExportedShare{
		Name: "old_name",
		MountPointData: dbom.MountPointPath{
			Path:   "/mnt/old_name",
			Source: "old_source",
			FSType: "ext4",
		},
	}
	err := suite.export_share_repo.Save(share)
	suite.Require().NoError(err)

	// Act
	err = suite.export_share_repo.UpdateName("old_name", "new_name")

	// Assert
	suite.Require().NoError(err)

	// Check that it is really updated
	_, err = suite.export_share_repo.FindByName("old_name")
	suite.Require().Error(err)
	suite.True(errors.Is(err, gorm.ErrRecordNotFound))

	newShare, err := suite.export_share_repo.FindByName("new_name")
	suite.Require().NoError(err)
	suite.Require().NotNil(newShare)
	suite.Equal("new_name", newShare.Name)

	// Cleanup
	dbom.GetDB().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.ExportedShare{})
	dbom.GetDB().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.MountPointPath{})
}
