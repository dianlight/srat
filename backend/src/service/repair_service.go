package service

import (
	"context"
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/tlog"
	"github.com/google/uuid"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

type RepairServiceInterface interface {
	Create(command dto.RepairCommandMessage) (*dto.ManagedRepair, error)
	Update(command dto.RepairCommandMessage) (*dto.ManagedRepair, error)
	Delete(repairID string) error
	Get(repairID string) (*dto.ManagedRepair, bool)
	List() []dto.ManagedRepair
	ApplyLifecycle(event dto.RepairLifecycleMessage) (*dto.ManagedRepair, error)
	EnqueueCommand(command dto.RepairCommandMessage) error
	FlushQueuedCommands() []dto.RepairCommandMessage
	QueueSize() int
}

type RepairService struct {
	ctx          context.Context
	stateContext *dto.ContextState
	broadcaster  BroadcasterServiceInterface
	problemSvc   ProblemServiceInterface
	mutex        sync.RWMutex
	state        map[string]dto.ManagedRepair
	queue        []dto.RepairCommandMessage
	queuedIDs    map[string]struct{}
}

type RepairServiceParams struct {
	fx.In
	Ctx         context.Context
	State       *dto.ContextState
	Broadcaster BroadcasterServiceInterface `optional:"true"`
	Problem     ProblemServiceInterface     `optional:"true"`
}

func NewRepairService(params RepairServiceParams) RepairServiceInterface {
	return &RepairService{
		ctx:          params.Ctx,
		stateContext: params.State,
		broadcaster:  params.Broadcaster,
		problemSvc:   params.Problem,
		state:        make(map[string]dto.ManagedRepair),
		queue:        make([]dto.RepairCommandMessage, 0),
		queuedIDs:    make(map[string]struct{}),
	}
}

func (s *RepairService) Create(command dto.RepairCommandMessage) (*dto.ManagedRepair, error) {
	if err := command.Validate(); err != nil {
		return nil, err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, ok := s.state[command.RepairID]; ok {
		return nil, fmt.Errorf("repair %q already exists", command.RepairID)
	}

	record := dto.ManagedRepair{
		RepairID:      command.RepairID,
		LastCommandID: command.CommandID,
		LastAction:    command.Action,
		Status:        dto.RepairLifecycleStatusUpdated,
		UpdatedAt:     time.Now(),
		Command:       command,
	}

	if command.Action == dto.RepairCommandActionUpsert {
		record.Status = dto.RepairLifecycleStatusCreated
	}

	s.state[command.RepairID] = record
	s.syncProblemFromCommand(command, dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSCREATED)
	if s.broadcaster != nil {
		s.broadcaster.BroadcastMessage(command)
	} else {
		tlog.WarnContext(s.ctx, "No broadcaster available to broadcast repair command", "repair_id", command.RepairID)
	}
	if s.stateContext == nil || s.stateContext.HAWsComponent == nil {
		s.enqueueLocked(command)
	}

	copyRecord := record
	return &copyRecord, nil
}

func (s *RepairService) Update(command dto.RepairCommandMessage) (*dto.ManagedRepair, error) {
	if err := command.Validate(); err != nil {
		return nil, err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	record, ok := s.state[command.RepairID]
	if !ok {
		return nil, fmt.Errorf("repair %q not found", command.RepairID)
	}

	record.LastCommandID = command.CommandID
	record.LastAction = command.Action
	record.Command = command
	record.Status = dto.RepairLifecycleStatusUpdated
	record.LastError = nil
	record.UpdatedAt = time.Now()
	s.state[command.RepairID] = record
	s.syncProblemFromCommand(command, dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSUPDATED)
	if s.broadcaster != nil {
		s.broadcaster.BroadcastMessage(command)
	}
	if s.stateContext == nil || s.stateContext.HAWsComponent == nil {
		s.enqueueLocked(command)
	}

	copyRecord := record
	return &copyRecord, nil
}

func (s *RepairService) Delete(repairID string) error {
	if repairID == "" {
		return errors.BaseWrapf(dto.ErrorInvalidParameter, "repair_id is required")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, ok := s.state[repairID]; !ok {
		return errors.BaseWrapf(dto.ErrorNotFound, "repair %q not found", repairID)
	}

	delete(s.state, repairID)
	s.dismissProblem(repairID)
	if s.broadcaster != nil {
		s.broadcaster.BroadcastMessage(dto.RepairCommandMessage{
			CommandID: uuid.NewString(),
			RepairID:  repairID,
			Action:    dto.RepairCommandActionDelete,
		})
	}
	return nil
}

func (s *RepairService) Get(repairID string) (*dto.ManagedRepair, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	record, ok := s.state[repairID]
	if !ok {
		return nil, false
	}

	copyRecord := record
	return &copyRecord, true
}

func (s *RepairService) List() []dto.ManagedRepair {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	ret := make([]dto.ManagedRepair, 0, len(s.state))
	for value := range maps.Values(s.state) {
		ret = append(ret, value)
	}

	return ret
}

func (s *RepairService) ApplyLifecycle(event dto.RepairLifecycleMessage) (*dto.ManagedRepair, error) {
	if err := event.Validate(); err != nil {
		return nil, err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	record, ok := s.state[event.RepairID]
	if !ok {
		return nil, errors.BaseWrapf(dto.ErrorNotFound, "repair %q not found", event.RepairID)
	}

	record.Status = event.Status
	record.LastCommandID = event.CommandID
	record.LastError = event.Error
	record.UpdatedAt = time.Now()
	copyEvent := event
	record.Lifecycle = &copyEvent
	s.state[event.RepairID] = record
	s.applyProblemLifecycle(event)

	copyRecord := record
	return &copyRecord, nil
}

func (s *RepairService) syncProblemFromCommand(command dto.RepairCommandMessage, status dto.ProblemLifecycleStatus) {
	if s.problemSvc == nil {
		return
	}

	title := command.TranslationKey
	if title == "" {
		title = command.RepairID
	}

	_, err := s.problemSvc.Upsert(&dto.Problem{
		ProblemKey:              command.RepairID,
		Title:                   title,
		Description:             title,
		Severity:                mapRepairSeverity(command.Severity),
		Status:                  status,
		TranslationKey:          command.TranslationKey,
		TranslationPlaceholders: command.TranslationPlaceholders,
		Data:                    command.Data,
		LearnMoreURL:            command.LearnMoreURL,
		IsFixable:               command.IsFixable,
		IsPersistent:            command.IsPersistent,
	})
	if err != nil {
		tlog.WarnContext(s.ctx, "Failed to sync repair command to problem", "repair_id", command.RepairID, "error", err)
	}
}

func (s *RepairService) dismissProblem(repairID string) {
	if s.problemSvc == nil {
		return
	}

	if err := s.problemSvc.Dismiss(repairID); err != nil {
		tlog.WarnContext(s.ctx, "Failed to dismiss mirrored problem", "problem_key", repairID, "error", err)
	}
}

func (s *RepairService) applyProblemLifecycle(event dto.RepairLifecycleMessage) {
	if s.problemSvc == nil {
		return
	}

	_, err := s.problemSvc.ApplyLifecycle(event.RepairID, mapRepairLifecycleStatus(event.Status), event.Error)
	if err != nil {
		tlog.WarnContext(s.ctx, "Failed to sync repair lifecycle to problem", "repair_id", event.RepairID, "status", event.Status, "error", err)
	}
}

func mapRepairSeverity(severity dto.RepairIssueSeverity) dto.ProblemSeverity {
	switch severity {
	case dto.RepairIssueSeverities.REPAIRISSUESEVERITYCRITICAL:
		return dto.ProblemSeverities.PROBLEMSEVERITYCRITICAL
	case dto.RepairIssueSeverities.REPAIRISSUESEVERITYERROR:
		return dto.ProblemSeverities.PROBLEMSEVERITYERROR
	default:
		return dto.ProblemSeverities.PROBLEMSEVERITYWARNING
	}
}

func mapRepairLifecycleStatus(status dto.RepairLifecycleStatus) dto.ProblemLifecycleStatus {
	switch status {
	case dto.RepairLifecycleStatuses.REPAIRLIFECYCLESTATUSCREATED:
		return dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSCREATED
	case dto.RepairLifecycleStatuses.REPAIRLIFECYCLESTATUSIGNORED:
		return dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSIGNORED
	case dto.RepairLifecycleStatuses.REPAIRLIFECYCLESTATUSFIXED:
		return dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSFIXED
	case dto.RepairLifecycleStatuses.REPAIRLIFECYCLESTATUSDISMISSED:
		return dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSDISMISSED
	case dto.RepairLifecycleStatuses.REPAIRLIFECYCLESTATUSDELETED:
		return dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSDELETED
	case dto.RepairLifecycleStatuses.REPAIRLIFECYCLESTATUSERROR:
		return dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSERROR
	default:
		return dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSUPDATED
	}
}

func (s *RepairService) EnqueueCommand(command dto.RepairCommandMessage) error {
	if err := command.Validate(); err != nil {
		return err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.enqueueLocked(command)
	return nil
}

func (s *RepairService) enqueueLocked(command dto.RepairCommandMessage) {
	if command.CommandID != "" {
		if _, exists := s.queuedIDs[command.CommandID]; exists {
			return
		}
		s.queuedIDs[command.CommandID] = struct{}{}
	}
	s.queue = append(s.queue, command)
}

func (s *RepairService) FlushQueuedCommands() []dto.RepairCommandMessage {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(s.queue) == 0 {
		return nil
	}

	queued := make([]dto.RepairCommandMessage, len(s.queue))
	copy(queued, s.queue)
	clear(s.queuedIDs)
	s.queue = s.queue[:0]
	return queued
}

func (s *RepairService) QueueSize() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.queue)
}
