package service

import (
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"go.uber.org/fx"
)

// IssueService handles business logic for issues.
type IssueService struct {
	repo      repository.IssueRepositoryInterface
	converter converter.IssueToDtoConverterImpl
}

// IssueServiceInterface defines the interface for issue services.
type IssueServiceInterface interface {
	Create(issue *dto.Issue) error
	Resolve(id uint) error
	Update(issue *dto.Issue) (*dto.Issue, error)
	FindOpen() ([]*dto.Issue, error)
}

// NewIssueService creates a new issue service.
func NewIssueService(repo repository.IssueRepositoryInterface) IssueServiceInterface {
	return &IssueService{repo: repo, converter: converter.IssueToDtoConverterImpl{}}
}

// Create creates a new issue.
func (s *IssueService) Create(issue *dto.Issue) error {
	dbom := s.converter.ToDbom(issue)
	// Search for existing issue by title
	existingIssue, err := s.repo.FindByTitle(issue.Title)
	if err != nil {
		return err
	}
	if existingIssue != nil {
		existingIssue.Repeating++
		if err := s.repo.Update(existingIssue); err != nil {
			return err
		}
		issue = s.converter.ToDto(dbom)
	} else {
		if err := s.repo.Create(dbom); err != nil {
			return err
		}
		issue = s.converter.ToDto(dbom)
	}
	return nil
}

// Resolve resolves an issue.
func (s *IssueService) Resolve(id uint) error {
	return s.repo.Delete(id)
}

// Update updates an existing issue.
func (s *IssueService) Update(issue *dto.Issue) (*dto.Issue, error) {
	dbom := s.converter.ToDbom(issue)
	if err := s.repo.Update(dbom); err != nil {
		return nil, err
	}
	return s.converter.ToDto(dbom), nil
}

// FindOpen returns all open issues.
func (s *IssueService) FindOpen() ([]*dto.Issue, error) {
	dboms, err := s.repo.FindOpen()
	if err != nil {
		return nil, err
	}
	return s.converter.ToDtoList(dboms), nil
}

// Fx provides the issue service as a dependency.
var Fx = fx.Provide(NewIssueService)
