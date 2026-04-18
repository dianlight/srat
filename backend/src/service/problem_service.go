package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type ProblemServiceInterface interface {
	Upsert(problem *dto.Problem) (*dto.Problem, error)
	Dismiss(problemKey string) error
	Get(problemKey string) (*dto.Problem, error)
	List() ([]*dto.Problem, error)
	ApplyLifecycle(problemKey string, status dto.ProblemLifecycleStatus, lastError *string) (*dto.Problem, error)
}

type ProblemServiceParams struct {
	fx.In
	Ctx      context.Context
	DB       *gorm.DB
	EventBus events.EventBusInterface
}

type ProblemService struct {
	ctx      context.Context
	db       *gorm.DB
	eventBus events.EventBusInterface

	mu    sync.RWMutex
	cache map[string]dto.Problem
}

var problemConv = converter.ProblemToDtoConverterImpl{}

func NewProblemService(params ProblemServiceParams) ProblemServiceInterface {
	svc := &ProblemService{
		ctx:      params.Ctx,
		db:       params.DB,
		eventBus: params.EventBus,
		cache:    make(map[string]dto.Problem),
	}
	svc.loadCache()
	return svc
}

func (s *ProblemService) loadCache() {
	var rows []dbom.Problem
	if err := s.db.WithContext(s.ctx).Find(&rows).Error; err != nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, row := range rows {
		item := problemConv.ToDto(&row)
		s.cache[s.cacheKey(item.ProblemKey, item.Title)] = *item
	}
}

func (s *ProblemService) Upsert(problem *dto.Problem) (*dto.Problem, error) {
	if problem == nil {
		return nil, fmt.Errorf("problem is required")
	}
	if problem.ProblemKey == "" && strings.TrimSpace(problem.Title) == "" {
		return nil, fmt.Errorf("problem_key or title is required")
	}

	key := strings.TrimSpace(problem.ProblemKey)
	title := strings.TrimSpace(problem.Title)
	var existing dbom.Problem

	query := s.db.WithContext(s.ctx)
	if key != "" {
		query = query.Where("problem_key = ?", key)
	} else {
		query = query.Where("title = ?", title)
	}

	err := query.First(&existing).Error
	if err == nil {
		existing.Title = problem.Title
		existing.Description = problem.Description
		existing.Severity = problem.Severity
		existing.Status = dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSUPDATED
		existing.Ignored = problem.Ignored
		existing.Actions = problem.Actions
		existing.TranslationKey = problem.TranslationKey
		existing.TranslationPlaceholders = problem.TranslationPlaceholders
		existing.Data = problem.Data
		existing.DetailLink = problem.DetailLink
		existing.ResolutionLink = problem.ResolutionLink
		existing.IsFixable = problem.IsFixable
		existing.IsPersistent = problem.IsPersistent
		existing.Repeating++
		if problem.LearnMoreURL != nil {
			existing.LearnMoreURL = *problem.LearnMoreURL
		}
		if problem.LastError != nil {
			existing.LastError = *problem.LastError
		} else {
			existing.LastError = ""
		}

		if saveErr := s.db.WithContext(s.ctx).Save(&existing).Error; saveErr != nil {
			return nil, saveErr
		}

		item := problemConv.ToDto(&existing)
		s.mu.Lock()
		s.cache[s.cacheKey(item.ProblemKey, item.Title)] = *item
		s.mu.Unlock()
		s.eventBus.EmitProblem(events.ProblemEvent{Event: events.Event{Type: events.EventTypes.UPDATE}, Problem: item})
		return item, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	row := problemConv.ToDbom(problem)
	if row.Repeating == 0 {
		row.Repeating = 1
	}

	if createErr := s.db.WithContext(s.ctx).Create(row).Error; createErr != nil {
		return nil, createErr
	}

	item := problemConv.ToDto(row)
	s.mu.Lock()
	s.cache[s.cacheKey(item.ProblemKey, item.Title)] = *item
	s.mu.Unlock()
	s.eventBus.EmitProblem(events.ProblemEvent{Event: events.Event{Type: events.EventTypes.ADD}, Problem: item})
	return item, nil
}

func (s *ProblemService) Dismiss(problemKey string) error {
	if strings.TrimSpace(problemKey) == "" {
		return fmt.Errorf("problem_key is required")
	}

	var row dbom.Problem
	if err := s.db.WithContext(s.ctx).Where("problem_key = ?", problemKey).First(&row).Error; err != nil {
		return err
	}

	if err := s.db.WithContext(s.ctx).Unscoped().Delete(&dbom.Problem{}, row.ID).Error; err != nil {
		return err
	}

	item := problemConv.ToDto(&row)
	s.mu.Lock()
	delete(s.cache, s.cacheKey(item.ProblemKey, item.Title))
	s.mu.Unlock()
	s.eventBus.EmitProblem(events.ProblemEvent{Event: events.Event{Type: events.EventTypes.REMOVE}, Problem: item})
	return nil
}

func (s *ProblemService) Get(problemKey string) (*dto.Problem, error) {
	if strings.TrimSpace(problemKey) == "" {
		return nil, fmt.Errorf("problem_key is required")
	}

	s.mu.RLock()
	if value, ok := s.cache[s.cacheKey(problemKey, "")]; ok {
		copyValue := value
		s.mu.RUnlock()
		return &copyValue, nil
	}
	s.mu.RUnlock()

	var row dbom.Problem
	if err := s.db.WithContext(s.ctx).Where("problem_key = ?", problemKey).First(&row).Error; err != nil {
		return nil, err
	}

	item := problemConv.ToDto(&row)
	s.mu.Lock()
	s.cache[s.cacheKey(item.ProblemKey, item.Title)] = *item
	s.mu.Unlock()
	return item, nil
}

func (s *ProblemService) List() ([]*dto.Problem, error) {
	s.mu.RLock()
	if len(s.cache) > 0 {
		ret := make([]*dto.Problem, 0, len(s.cache))
		for _, item := range s.cache {
			copyItem := item
			ret = append(ret, &copyItem)
		}
		s.mu.RUnlock()
		sort.SliceStable(ret, func(i, j int) bool {
			return ret[i].UpdatedAt.After(ret[j].UpdatedAt)
		})
		return ret, nil
	}
	s.mu.RUnlock()

	var rows []dbom.Problem
	if err := s.db.WithContext(s.ctx).Order("updated_at desc").Find(&rows).Error; err != nil {
		return nil, err
	}

	ret := make([]*dto.Problem, 0, len(rows))
	for _, row := range rows {
		ret = append(ret, problemConv.ToDto(&row))
	}
	return ret, nil
}

func (s *ProblemService) ApplyLifecycle(problemKey string, status dto.ProblemLifecycleStatus, lastError *string) (*dto.Problem, error) {
	if strings.TrimSpace(problemKey) == "" {
		return nil, fmt.Errorf("problem_key is required")
	}

	var row dbom.Problem
	if err := s.db.WithContext(s.ctx).Where("problem_key = ?", problemKey).First(&row).Error; err != nil {
		return nil, err
	}

	row.Status = status
	if lastError != nil {
		row.LastError = *lastError
	} else {
		row.LastError = ""
	}

	if err := s.db.WithContext(s.ctx).Save(&row).Error; err != nil {
		return nil, err
	}

	item := problemConv.ToDto(&row)
	s.mu.Lock()
	s.cache[s.cacheKey(item.ProblemKey, item.Title)] = *item
	s.mu.Unlock()
	s.eventBus.EmitProblem(events.ProblemEvent{Event: events.Event{Type: events.EventTypes.UPDATE}, Problem: item})
	return item, nil
}

func (s *ProblemService) cacheKey(problemKey, title string) string {
	if strings.TrimSpace(problemKey) != "" {
		return "key:" + strings.TrimSpace(problemKey)
	}
	return "title:" + strings.TrimSpace(title)
}
