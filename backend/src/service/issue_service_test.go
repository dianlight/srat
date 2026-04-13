package service_test

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type IssueServiceSuite struct {
	suite.Suite
	issueService service.IssueServiceInterface
	db           *gorm.DB
	sqlDB        *sql.DB
}

func TestIssueServiceSuite(t *testing.T) {
	suite.Run(t, new(IssueServiceSuite))
}

func (suite *IssueServiceSuite) SetupTest() {
	dsn := fmt.Sprintf("file:issue-service-%d?mode=memory&cache=shared", time.Now().UnixNano())
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	suite.Require().NoError(err)
	suite.Require().NoError(gdb.AutoMigrate(&dbom.Issue{}))

	sqlDB, err := gdb.DB()
	suite.Require().NoError(err)

	suite.db = gdb
	suite.sqlDB = sqlDB
	suite.issueService = service.NewIssueService(gdb, context.Background())
}

func (suite *IssueServiceSuite) TearDownTest() {
	if suite.sqlDB != nil {
		_ = suite.sqlDB.Close()
	}
}

func (suite *IssueServiceSuite) TestCreateExistingIncrementsRepeatingSuccess() {
	existing := &dbom.Issue{
		Title:     "test",
		Repeating: 1,
	}
	suite.Require().NoError(suite.db.Create(existing).Error)

	err := suite.issueService.Create(&dto.Issue{Title: "test"})
	suite.NoError(err)

	var found dbom.Issue
	suite.Require().NoError(suite.db.Where("title = ?", "test").First(&found).Error)
	suite.Equal(uint(2), found.Repeating)
}

func (suite *IssueServiceSuite) TestCreateNewCreatesSuccess() {
	err := suite.issueService.Create(&dto.Issue{Title: "new"})
	suite.NoError(err)

	var found dbom.Issue
	suite.Require().NoError(suite.db.Where("title = ?", "new").First(&found).Error)
	suite.Equal("new", found.Title)
}

func (suite *IssueServiceSuite) TestResolveSuccess() {
	issue := &dbom.Issue{Title: "to-resolve"}
	suite.Require().NoError(suite.db.Create(issue).Error)

	err := suite.issueService.Resolve(issue.ID)
	suite.NoError(err)

	var found dbom.Issue
	err = suite.db.First(&found, issue.ID).Error
	suite.ErrorIs(err, gorm.ErrRecordNotFound)
}

func (suite *IssueServiceSuite) TestFindOpenReturnsRecentFive() {
	now := time.Now()
	for i := 1; i <= 7; i++ {
		issue := &dbom.Issue{
			Title:     "issue-" + strconv.Itoa(i),
			CreatedAt: now.Add(time.Duration(-(i - 1)) * time.Minute),
		}
		suite.Require().NoError(suite.db.Create(issue).Error)
	}

	issues, err := suite.issueService.FindOpen()
	suite.NoError(err)
	suite.Len(issues, 5)
	suite.Equal("issue-1", issues[0].Title)
	suite.Equal("issue-5", issues[4].Title)
}

func (suite *IssueServiceSuite) TestFindByTitleSuccessAndNotFound() {
	title := "lookup-title"
	suite.Require().NoError(suite.db.Create(&dbom.Issue{Title: title}).Error)

	found, err := suite.issueService.FindByTitle(title)
	suite.NoError(err)
	suite.NotNil(found)
	suite.Equal(title, found.Title)

	missing, err := suite.issueService.FindByTitle("missing-title")
	suite.NoError(err)
	suite.Nil(missing)
}

func (suite *IssueServiceSuite) TestResolveByTitle() {
	title := "to-resolve"
	suite.Require().NoError(suite.db.Create(&dbom.Issue{Title: title}).Error)

	err := suite.issueService.ResolveByTitle(title)
	suite.NoError(err)

	var found dbom.Issue
	err = suite.db.Where("title = ?", title).First(&found).Error
	suite.ErrorIs(err, gorm.ErrRecordNotFound)

	err = suite.issueService.ResolveByTitle("already-gone")
	suite.ErrorIs(err, gorm.ErrRecordNotFound)
}

func (suite *IssueServiceSuite) TestUpdateSuccess() {
	issue := &dbom.Issue{Title: "old", Description: "old"}
	suite.Require().NoError(suite.db.Create(issue).Error)

	updated, err := suite.issueService.Update(&dto.Issue{ID: issue.ID, Title: "Updated", Description: "Updated description"})
	suite.NoError(err)
	suite.NotNil(updated)

	var found dbom.Issue
	suite.Require().NoError(suite.db.First(&found, issue.ID).Error)
	suite.Equal("Updated", found.Title)
	suite.Equal("Updated description", found.Description)
}
