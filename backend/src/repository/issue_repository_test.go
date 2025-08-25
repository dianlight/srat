package repository_test

import (
	"database/sql"
	"strconv"
	"testing"
	"time"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/repository"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// go

type IssueRepositorySuite struct {
	suite.Suite
	db    *gorm.DB
	sqlDB *sql.DB
	repo  repository.IssueRepositoryInterface
}

func (s *IssueRepositorySuite) SetupTest() {
	// Open in-memory sqlite DB
	gdb, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	s.Require().NoError(err, "failed to open gorm sqlite in-memory")

	// Automigrate the Issue model
	err = gdb.AutoMigrate(&dbom.Issue{})
	s.Require().NoError(err, "auto migrate failed")

	// Keep underlying sql.DB to close later
	sqlDB, err := gdb.DB()
	s.Require().NoError(err)
	s.sqlDB = sqlDB

	s.db = gdb
	s.repo = repository.NewIssueRepository(gdb)
}

func (s *IssueRepositorySuite) TearDownTest() {
	if s.sqlDB != nil {
		_ = s.sqlDB.Close()
	}
}

func (s *IssueRepositorySuite) TestCreateIssueSuccess() {
	issue := &dbom.Issue{
		Title:       "create-test",
		Description: "create description",
	}

	err := s.repo.Create(issue)
	s.Require().NoError(err)
	s.Require().NotZero(issue.ID, "expected ID to be set after create")

	var found dbom.Issue
	err = s.db.First(&found, issue.ID).Error
	s.Require().NoError(err)
	s.Equal(issue.Title, found.Title)
	s.Equal(issue.Description, found.Description)
}

func (s *IssueRepositorySuite) TestUpdateIssueSuccess() {
	issue := &dbom.Issue{
		Title:       "update-test",
		Description: "before",
	}
	// create using repo to ensure same path
	err := s.repo.Create(issue)
	s.Require().NoError(err)

	issue.Description = "after"
	issue.Title = "updated-title"

	err = s.repo.Update(issue)
	s.Require().NoError(err)

	var found dbom.Issue
	err = s.db.First(&found, issue.ID).Error
	s.Require().NoError(err)
	s.Equal("after", found.Description)
	s.Equal("updated-title", found.Title)
}

func (s *IssueRepositorySuite) TestDeleteIssueSuccess() {
	issue := &dbom.Issue{
		Title:       "delete-test",
		Description: "to be deleted",
	}
	err := s.repo.Create(issue)
	s.Require().NoError(err)

	err = s.repo.Delete(issue.ID)
	s.Require().NoError(err)

	var found dbom.Issue
	err = s.db.First(&found, issue.ID).Error
	s.Require().Error(err)
	s.ErrorIs(err, gorm.ErrRecordNotFound, "expected record not found after delete")
}

func (s *IssueRepositorySuite) TestFindOpenReturnsRecentFive() {
	// create 7 issues with descending CreatedAt (issue-1 newest)
	now := time.Now()
	for i := 1; i <= 7; i++ {
		issue := &dbom.Issue{
			Title:       "issue-" + string(rune('0'+i)), // temporary title, will correct below
			Description: "bulk",
		}
		// set a predictable title and createdAt
		issue.Title = "issue-" + strconv.Itoa(i)
		issue.CreatedAt = now.Add(time.Duration(-(i - 1)) * time.Minute)
		// insert using GORM directly to control CreatedAt
		err := s.db.Create(issue).Error
		s.Require().NoError(err)
	}

	results, err := s.repo.FindOpen()
	s.Require().NoError(err)
	s.Require().Len(results, 5, "expected the repository to return up to 5 records")

	// results should be ordered by created_at desc so first is issue-1 (newest)
	s.Equal("issue-1", results[0].Title)
	// fifth in results should be issue-5
	s.Equal("issue-5", results[4].Title)
}

func (s *IssueRepositorySuite) TestFindByTitleSuccessAndNotFound() {
	uniqueTitle := "unique-title-123"
	issue := &dbom.Issue{
		Title:       uniqueTitle,
		Description: "unique",
	}
	err := s.repo.Create(issue)
	s.Require().NoError(err)

	found, err := s.repo.FindByTitle(uniqueTitle)
	s.Require().NoError(err)
	s.Require().NotNil(found)
	s.Equal(uniqueTitle, found.Title)

	// not found case
	_, err = s.repo.FindByTitle("no-such-title-xyz")
	s.Require().Error(err)
	s.ErrorIs(err, gorm.ErrRecordNotFound)
}

func TestIssueRepositorySuite(t *testing.T) {
	suite.Run(t, new(IssueRepositorySuite))
}
