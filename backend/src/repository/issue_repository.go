package repository

import (
	"github.com/dianlight/srat/dbom"
	"gorm.io/gorm"
)

// IssueRepository handles database operations for issues.
type IssueRepository struct {
	db *gorm.DB
}

// IssueRepositoryInterface defines the methods for the issue repository.
type IssueRepositoryInterface interface {
	Create(issue *dbom.Issue) error
	Update(issue *dbom.Issue) error
	Delete(id uint) error
	FindOpen() ([]*dbom.Issue, error)
	FindByTitle(title string) (*dbom.Issue, error)
}

// NewIssueRepository creates a new issue repository.
func NewIssueRepository(db *gorm.DB) IssueRepositoryInterface {
	return &IssueRepository{db: db}
}

// Create creates a new issue.
func (r *IssueRepository) Create(issue *dbom.Issue) error {
	return r.db.Create(issue).Error
}

// Update updates an existing issue.
func (r *IssueRepository) Update(issue *dbom.Issue) error {
	return r.db.Save(issue).Error
}

// Delete deletes an issue.
func (r *IssueRepository) Delete(id uint) error {
	return r.db.Delete(&dbom.Issue{}, id).Error
}

// FindOpen returns all open issues.
func (r *IssueRepository) FindOpen() ([]*dbom.Issue, error) {
	var issues []*dbom.Issue
	err := r.db.Order("created_at desc").Limit(5).Find(&issues).Error
	return issues, err
}

// FindByTitle finds an issue by title.
func (r *IssueRepository) FindByTitle(title string) (*dbom.Issue, error) {
	var issue dbom.Issue
	err := r.db.Where("title = ?", title).First(&issue).Error
	if err != nil {
		return nil, err
	}
	return &issue, nil
}
