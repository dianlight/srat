package service

import (
	"context"
	"errors"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"gorm.io/gorm"
)

// IssueService handles business logic for issues.
type IssueService struct {
	db        *gorm.DB
	ctx       context.Context
	converter converter.IssueToDtoConverterImpl
}

// IssueServiceInterface defines the interface for issue services.
type IssueServiceInterface interface {
	Create(issue *dto.Issue) error
	Resolve(id uint) error
	ResolveByTitle(title string) error
	Update(issue *dto.Issue) (*dto.Issue, error)
	FindOpen() ([]*dto.Issue, error)
	FindByTitle(title string) (*dto.Issue, error)
}

// NewIssueService creates a new issue service.
func NewIssueService(db *gorm.DB, ctx context.Context) IssueServiceInterface {
	return &IssueService{db: db, ctx: ctx, converter: converter.IssueToDtoConverterImpl{}}
}

// Create creates a new issue.
func (s *IssueService) Create(issue *dto.Issue) error {
	dbIssue := s.converter.ToDbom(issue)
	// Search for existing issue by title
	existingIssue, err := s.FindByTitle(issue.Title)
	if err != nil {
		return err
	}
	if existingIssue != nil {
		existingDbom := s.converter.ToDbom(existingIssue)
		existingDbom.Repeating++
		if err := s.db.WithContext(s.ctx).Save(existingDbom).Error; err != nil {
			return err
		}
	} else {
		if err := s.db.WithContext(s.ctx).Create(dbIssue).Error; err != nil {
			return err
		}
	}
	return nil
}

// Resolve resolves an issue.
func (s *IssueService) Resolve(id uint) error {
	return s.db.WithContext(s.ctx).Delete(&dbom.Issue{}, id).Error
}

// ResolveByTitle resolves an issue by title if present.
func (s *IssueService) ResolveByTitle(title string) error {
	result := s.db.WithContext(s.ctx).Where("title = ?", title).Delete(&dbom.Issue{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

// Update updates an existing issue.
func (s *IssueService) Update(issue *dto.Issue) (*dto.Issue, error) {
	dbIssue := s.converter.ToDbom(issue)
	if err := s.db.WithContext(s.ctx).Save(dbIssue).Error; err != nil {
		return nil, err
	}
	return s.converter.ToDto(dbIssue), nil
}

// FindOpen returns all open issues.
func (s *IssueService) FindOpen() ([]*dto.Issue, error) {
	var dbIssues []*dbom.Issue
	err := s.db.WithContext(s.ctx).Order("created_at desc").Limit(5).Find(&dbIssues).Error
	if err != nil {
		return nil, err
	}
	return s.converter.ToDtoList(dbIssues), nil
}

// FindByTitle finds an issue by title and returns nil,nil when no issue exists.
func (s *IssueService) FindByTitle(title string) (*dto.Issue, error) {
	var dbIssue dbom.Issue
	err := s.db.WithContext(s.ctx).Where("title = ?", title).First(&dbIssue).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return s.converter.ToDto(&dbIssue), nil
}
