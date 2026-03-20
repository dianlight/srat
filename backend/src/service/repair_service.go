package service

import (
	"context"
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/dianlight/srat/dto"
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
	mutex        sync.RWMutex
	state        map[string]dto.ManagedRepair
	queue        []dto.RepairCommandMessage
	queuedIDs    map[string]struct{}
}

func NewRepairService(ctx context.Context, state *dto.ContextState) RepairServiceInterface {
	return &RepairService{
		ctx:          ctx,
		stateContext: state,
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
	if s.stateContext == nil || s.stateContext.HAWsComponent == nil {
		s.enqueueLocked(command)
	}

	copyRecord := record
	return &copyRecord, nil
}

func (s *RepairService) Delete(repairID string) error {
	if repairID == "" {
		return fmt.Errorf("repair_id is required")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, ok := s.state[repairID]; !ok {
		return fmt.Errorf("repair %q not found", repairID)
	}

	delete(s.state, repairID)
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
		return nil, fmt.Errorf("repair %q not found", event.RepairID)
	}

	record.Status = event.Status
	record.LastCommandID = event.CommandID
	record.LastError = event.Error
	record.UpdatedAt = time.Now()
	copyEvent := event
	record.Lifecycle = &copyEvent
	s.state[event.RepairID] = record

	copyRecord := record
	return &copyRecord, nil
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
