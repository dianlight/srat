package service

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/google/uuid"
)

const commandExecutionBufferSize = 500

type CommandExecutionServiceInterface interface {
	Start(ctx context.Context, commandID, label, command string, args ...string) (string, error)
	Execute(ctx context.Context, commandID, label, command string, args ...string) (dto.CommandExecutionSnapshot, error)
	GetSnapshot(executionID string) (dto.CommandExecutionSnapshot, bool)
}

type CommandExecutionService struct {
	broadcaster BroadcasterServiceInterface

	mu        sync.RWMutex
	snapshots map[string]dto.CommandExecutionSnapshot
}

func NewCommandExecutionService(broadcaster BroadcasterServiceInterface) CommandExecutionServiceInterface {
	return &CommandExecutionService{
		broadcaster: broadcaster,
		snapshots:   make(map[string]dto.CommandExecutionSnapshot),
	}
}

func (s *CommandExecutionService) Start(ctx context.Context, commandID, label, command string, args ...string) (string, error) {
	executionID := uuid.NewString()
	startedAt := time.Now().UnixMilli()

	snapshot := dto.CommandExecutionSnapshot{
		ExecutionID: executionID,
		CommandID:   commandID,
		Label:       label,
		Command:     command,
		Args:        append([]string(nil), args...),
		StartedAt:   startedAt,
		Running:     true,
		Lines:       make([]dto.CommandOutputLineSnapshot, 0, commandExecutionBufferSize),
	}

	s.mu.Lock()
	s.snapshots[executionID] = snapshot
	s.mu.Unlock()

	started := dto.CommandStartedNotification{
		ExecutionID: executionID,
		CommandID:   commandID,
		Label:       label,
		Command:     command,
		Args:        append([]string(nil), args...),
		StartedAt:   startedAt,
	}
	s.broadcaster.BroadcastMessage(started)

	cmd := exec.CommandContext(ctx, command, args...)
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		s.terminateWithError(executionID, commandID, -1, err.Error())
		return "", err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		s.terminateWithError(executionID, commandID, -1, err.Error())
		return "", err
	}
	if err := cmd.Start(); err != nil {
		s.terminateWithError(executionID, commandID, -1, err.Error())
		return "", err
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		s.scanPipe(ctx, executionID, commandID, dto.CommandOutputChannelStdout, stdoutPipe)
	})
	wg.Go(func() {
		s.scanPipe(ctx, executionID, commandID, dto.CommandOutputChannelStderr, stderrPipe)
	})

	go func() {
		wg.Wait()
		exitCode := 0
		success := true
		errMsg := ""

		if waitErr := cmd.Wait(); waitErr != nil {
			success = false
			errMsg = waitErr.Error()
			exitCode = -1
			if exitErr, ok := waitErr.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}

		s.mu.Lock()
		snapshot, ok := s.snapshots[executionID]
		if ok {
			snapshot.Running = false
			snapshot.FinishedAt = time.Now().UnixMilli()
			snapshot.ExitCode = exitCode
			snapshot.Success = success
			snapshot.Error = errMsg
			s.snapshots[executionID] = snapshot
		}
		s.mu.Unlock()

		s.broadcaster.BroadcastMessage(dto.CommandTerminatedNotification{
			ExecutionID: executionID,
			CommandID:   commandID,
			ExitCode:    exitCode,
			Success:     success,
			FinishedAt:  time.Now().UnixMilli(),
			Error:       errMsg,
		})
	}()

	return executionID, nil
}

func (s *CommandExecutionService) Execute(ctx context.Context, commandID, label, command string, args ...string) (dto.CommandExecutionSnapshot, error) {
	executionID, err := s.Start(ctx, commandID, label, command, args...)
	if err != nil {
		return dto.CommandExecutionSnapshot{}, err
	}

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return dto.CommandExecutionSnapshot{}, ctx.Err()
		case <-ticker.C:
			snapshot, ok := s.GetSnapshot(executionID)
			if !ok {
				continue
			}
			if !snapshot.Running {
				if !snapshot.Success {
					if snapshot.Error != "" {
						return snapshot, errors.New(snapshot.Error)
					}
					return snapshot, errors.New("command execution failed")
				}
				return snapshot, nil
			}
		}
	}
}

func (s *CommandExecutionService) GetSnapshot(executionID string) (dto.CommandExecutionSnapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot, ok := s.snapshots[executionID]
	if !ok {
		return dto.CommandExecutionSnapshot{}, false
	}
	copySnapshot := snapshot
	copySnapshot.Args = append([]string(nil), snapshot.Args...)
	copySnapshot.Lines = append([]dto.CommandOutputLineSnapshot(nil), snapshot.Lines...)
	return copySnapshot, true
}

func (s *CommandExecutionService) scanPipe(
	ctx context.Context,
	executionID, commandID string,
	channel dto.CommandOutputChannel,
	pipe io.Reader,
) {
	scanner := bufio.NewScanner(pipe)
	buffer := make([]byte, 0, 1024)
	scanner.Buffer(buffer, 1024*1024)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}
		line := scanner.Text()
		ts := time.Now().UnixMilli()

		s.mu.Lock()
		snapshot, ok := s.snapshots[executionID]
		if ok {
			snapshot.Lines = append(snapshot.Lines, dto.CommandOutputLineSnapshot{
				Channel:   channel,
				Line:      line,
				Timestamp: ts,
			})
			if len(snapshot.Lines) > commandExecutionBufferSize {
				snapshot.Lines = append([]dto.CommandOutputLineSnapshot(nil), snapshot.Lines[len(snapshot.Lines)-commandExecutionBufferSize:]...)
			}
			s.snapshots[executionID] = snapshot
		}
		s.mu.Unlock()

		s.broadcaster.BroadcastMessage(dto.CommandOutputNotification{
			ExecutionID: executionID,
			CommandID:   commandID,
			Channel:     channel,
			Line:        line,
			Timestamp:   ts,
		})
	}
}

func (s *CommandExecutionService) terminateWithError(executionID, commandID string, exitCode int, errMsg string) {
	s.mu.Lock()
	snapshot, ok := s.snapshots[executionID]
	if ok {
		snapshot.Running = false
		snapshot.Success = false
		snapshot.ExitCode = exitCode
		snapshot.Error = errMsg
		snapshot.FinishedAt = time.Now().UnixMilli()
		s.snapshots[executionID] = snapshot
	}
	s.mu.Unlock()

	s.broadcaster.BroadcastMessage(dto.CommandTerminatedNotification{
		ExecutionID: executionID,
		CommandID:   commandID,
		ExitCode:    exitCode,
		Success:     false,
		FinishedAt:  time.Now().UnixMilli(),
		Error:       errMsg,
	})
}
