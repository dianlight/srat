package service

import (
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"go.uber.org/fx"
)

// IssueService handles business logic for issues.
type IssueService struct {
	repo      *repository.IssueRepository
	converter converter.IssueToDtoConverterImpl
}

// NewIssueService creates a new issue service.
func NewIssueService(repo *repository.IssueRepository) *IssueService {
	return &IssueService{repo: repo, converter: converter.IssueToDtoConverterImpl{}}
}

// Create creates a new issue.
func (s *IssueService) Create(issue *dto.Issue) (*dto.Issue, error) {
	dbom := s.converter.ToDbom(issue)
	if err := s.repo.Create(dbom); err != nil {
		return nil, err
	}
	return s.converter.ToDto(dbom), nil
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
