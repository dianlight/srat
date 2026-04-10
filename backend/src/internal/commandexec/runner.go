package commandexec

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/google/uuid"
)

const bufferSize = 500

// Executor defines the shared backend contract for starting commands, collecting
// snapshots, and emitting lifecycle notifications through the shared event bus.
type Executor interface {
	LookPath(command string) (string, error)
	Start(ctx context.Context, commandID, label, command string, args ...string) (string, error)
	StartQuiet(ctx context.Context, commandID, label, command string, args ...string) (string, error)
	StartWithInput(ctx context.Context, commandID, label, stdinContent, command string, args ...string) (string, error)
	StartWithInputQuiet(ctx context.Context, commandID, label, stdinContent, command string, args ...string) (string, error)
	Execute(ctx context.Context, commandID, label, command string, args ...string) (dto.CommandExecutionSnapshot, error)
	ExecuteQuiet(ctx context.Context, commandID, label, command string, args ...string) (dto.CommandExecutionSnapshot, error)
	ExecuteWithInput(ctx context.Context, commandID, label, stdinContent, command string, args ...string) (dto.CommandExecutionSnapshot, error)
	ExecuteWithInputQuiet(ctx context.Context, commandID, label, stdinContent, command string, args ...string) (dto.CommandExecutionSnapshot, error)
	GetSnapshot(executionID string) (dto.CommandExecutionSnapshot, bool)
}

type executionState struct {
	snapshot dto.CommandExecutionSnapshot
	quiet    bool
}

// Service is the default in-memory implementation of `Executor`.
type Service struct {
	eventBus events.EventBusInterface

	mu        sync.RWMutex
	snapshots map[string]executionState
}

// NewCommandExecutor is the FX-friendly constructor for the shared command executor.
func NewCommandExecutor(eventBus events.EventBusInterface) Executor {
	return &Service{
		eventBus:  eventBus,
		snapshots: make(map[string]executionState),
	}
}

func (s *Service) LookPath(command string) (string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "", errors.New("command is empty")
	}
	return exec.LookPath(command)
}

func (s *Service) Start(ctx context.Context, commandID, label, command string, args ...string) (string, error) {
	return s.startWithInput(ctx, false, commandID, label, "", command, args...)
}

func (s *Service) StartQuiet(ctx context.Context, commandID, label, command string, args ...string) (string, error) {
	return s.startWithInput(ctx, true, commandID, label, "", command, args...)
}

func (s *Service) StartWithInput(ctx context.Context, commandID, label, stdinContent, command string, args ...string) (string, error) {
	return s.startWithInput(ctx, false, commandID, label, stdinContent, command, args...)
}

func (s *Service) StartWithInputQuiet(ctx context.Context, commandID, label, stdinContent, command string, args ...string) (string, error) {
	return s.startWithInput(ctx, true, commandID, label, stdinContent, command, args...)
}

func (s *Service) startWithInput(ctx context.Context, quiet bool, commandID, label, stdinContent, command string, args ...string) (string, error) {
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
		Lines:       make([]dto.CommandOutputLineSnapshot, 0, bufferSize),
	}

	s.mu.Lock()
	s.snapshots[executionID] = executionState{snapshot: snapshot, quiet: quiet}
	s.mu.Unlock()

	if !quiet {
		s.emitCommandEvent(events.EventTypes.START, dto.CommandStartedNotification{
			ExecutionID: executionID,
			CommandID:   commandID,
			Label:       label,
			Command:     command,
			Args:        append([]string(nil), args...),
			StartedAt:   startedAt,
		})
	}

	cmd := exec.CommandContext(ctx, command, args...)
	if stdinContent != "" {
		cmd.Stdin = strings.NewReader(stdinContent)
	}
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
		finishedAt := time.Now().UnixMilli()
		var lastOutputLine *dto.CommandOutputLineSnapshot
		shouldEmit := true

		s.mu.Lock()
		state, ok := s.snapshots[executionID]
		if ok {
			snapshot := state.snapshot
			shouldEmit = !state.quiet
			snapshot.Running = false
			snapshot.FinishedAt = finishedAt
			snapshot.ExitCode = exitCode
			snapshot.Success = success
			snapshot.Error = errMsg
			for i := len(snapshot.Lines) - 1; i >= 0; i-- {
				lineCopy := snapshot.Lines[i]
				if lastOutputLine == nil {
					lastOutputLine = &lineCopy
				}
				if snapshot.Lines[i].Channel == dto.CommandOutputChannelStderr {
					lastOutputLine = &lineCopy
					break
				}
			}
			state.snapshot = snapshot
			s.snapshots[executionID] = state
		}
		s.mu.Unlock()

		if shouldEmit && lastOutputLine != nil {
			s.emitCommandEvent(events.EventTypes.UPDATE, dto.CommandOutputNotification{
				ExecutionID: executionID,
				CommandID:   commandID,
				Channel:     lastOutputLine.Channel,
				Line:        lastOutputLine.Line,
				Timestamp:   lastOutputLine.Timestamp,
				ExitCode:    new(exitCode),
			})
		}

		if shouldEmit {
			eventType := events.EventTypes.STOP
			if !success {
				eventType = events.EventTypes.ERROR
			}
			s.emitCommandEvent(eventType, dto.CommandTerminatedNotification{
				ExecutionID: executionID,
				CommandID:   commandID,
				ExitCode:    exitCode,
				Success:     success,
				FinishedAt:  finishedAt,
				Error:       errMsg,
			})
		}
	}()

	return executionID, nil
}

func (s *Service) Execute(ctx context.Context, commandID, label, command string, args ...string) (dto.CommandExecutionSnapshot, error) {
	return s.executeWithInput(ctx, false, commandID, label, "", command, args...)
}

func (s *Service) ExecuteQuiet(ctx context.Context, commandID, label, command string, args ...string) (dto.CommandExecutionSnapshot, error) {
	return s.executeWithInput(ctx, true, commandID, label, "", command, args...)
}

func (s *Service) ExecuteWithInput(ctx context.Context, commandID, label, stdinContent, command string, args ...string) (dto.CommandExecutionSnapshot, error) {
	return s.executeWithInput(ctx, false, commandID, label, stdinContent, command, args...)
}

func (s *Service) ExecuteWithInputQuiet(ctx context.Context, commandID, label, stdinContent, command string, args ...string) (dto.CommandExecutionSnapshot, error) {
	return s.executeWithInput(ctx, true, commandID, label, stdinContent, command, args...)
}

func (s *Service) executeWithInput(ctx context.Context, quiet bool, commandID, label, stdinContent, command string, args ...string) (dto.CommandExecutionSnapshot, error) {
	executionID, err := s.startWithInput(ctx, quiet, commandID, label, stdinContent, command, args...)
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

func (s *Service) GetSnapshot(executionID string) (dto.CommandExecutionSnapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, ok := s.snapshots[executionID]
	if !ok {
		return dto.CommandExecutionSnapshot{}, false
	}
	snapshot := state.snapshot
	copySnapshot := snapshot
	copySnapshot.Args = append([]string(nil), snapshot.Args...)
	copySnapshot.Lines = append([]dto.CommandOutputLineSnapshot(nil), snapshot.Lines...)
	return copySnapshot, true
}

func (s *Service) scanPipe(
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

		shouldEmit := true
		s.mu.Lock()
		state, ok := s.snapshots[executionID]
		if ok {
			snapshot := state.snapshot
			shouldEmit = !state.quiet
			snapshot.Lines = append(snapshot.Lines, dto.CommandOutputLineSnapshot{
				Channel:   channel,
				Line:      line,
				Timestamp: ts,
			})
			if len(snapshot.Lines) > bufferSize {
				snapshot.Lines = append([]dto.CommandOutputLineSnapshot(nil), snapshot.Lines[len(snapshot.Lines)-bufferSize:]...)
			}
			state.snapshot = snapshot
			s.snapshots[executionID] = state
		}
		s.mu.Unlock()

		if shouldEmit {
			s.emitCommandEvent(events.EventTypes.UPDATE, dto.CommandOutputNotification{
				ExecutionID: executionID,
				CommandID:   commandID,
				Channel:     channel,
				Line:        line,
				Timestamp:   ts,
			})
		}
	}
}

func (s *Service) terminateWithError(executionID, commandID string, exitCode int, errMsg string) {
	finishedAt := time.Now().UnixMilli()
	shouldEmit := true

	s.mu.Lock()
	state, ok := s.snapshots[executionID]
	if ok {
		snapshot := state.snapshot
		shouldEmit = !state.quiet
		snapshot.Running = false
		snapshot.Success = false
		snapshot.ExitCode = exitCode
		snapshot.Error = errMsg
		snapshot.FinishedAt = finishedAt
		state.snapshot = snapshot
		s.snapshots[executionID] = state
	}
	s.mu.Unlock()

	if shouldEmit {
		s.emitCommandEvent(events.EventTypes.ERROR, dto.CommandTerminatedNotification{
			ExecutionID: executionID,
			CommandID:   commandID,
			ExitCode:    exitCode,
			Success:     false,
			FinishedAt:  finishedAt,
			Error:       errMsg,
		})
	}
}

// JoinChannelOutput joins buffered command output lines, optionally filtering by channel.
func JoinChannelOutput(lines []dto.CommandOutputLineSnapshot, channels ...dto.CommandOutputChannel) string {
	if len(lines) == 0 {
		return ""
	}

	allowed := make(map[dto.CommandOutputChannel]struct{}, len(channels))
	for _, channel := range channels {
		allowed[channel] = struct{}{}
	}

	var builder strings.Builder
	for _, line := range lines {
		if len(allowed) > 0 {
			if _, ok := allowed[line.Channel]; !ok {
				continue
			}
		}
		if builder.Len() > 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString(line.Line)
	}

	return strings.TrimSpace(builder.String())
}

func (s *Service) emitCommandEvent(eventType events.EventType, message dto.CommandExecutionNotification) {
	if s == nil || s.eventBus == nil {
		return
	}
	s.eventBus.EmitCommandExecution(events.CommandExecutionEvent{
		Event:   events.Event{Type: eventType},
		Message: message,
	})
}

var _ Executor = (*Service)(nil)
