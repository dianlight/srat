package repository

import (
	"github.com/dianlight/srat/dbom"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// IssueRepository handles database operations for issues.
type IssueRepository struct {
	db *gorm.DB
}

// NewIssueRepository creates a new issue repository.
func NewIssueRepository(db *gorm.DB) *IssueRepository {
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
	err := r.db.Find(&issues).Error
	return issues, err
}

// Fx provides the issue repository as a dependency.
var Fx = fx.Provide(NewIssueRepository)
