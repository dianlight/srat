package service_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/server/ws"
	"github.com/dianlight/srat/service"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type commandExecutionBroadcasterMock struct {
	mu       sync.Mutex
	messages []any
}

func (m *commandExecutionBroadcasterMock) BroadcastMessage(msg any) any {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
	return msg
}

func (m *commandExecutionBroadcasterMock) ProcessWebSocketChannel(_ ws.Sender) {}

func (m *commandExecutionBroadcasterMock) Messages() []any {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := make([]any, len(m.messages))
	copy(copied, m.messages)
	return copied
}

type CommandExecutionServiceTestSuite struct {
	suite.Suite
	service     service.CommandExecutionServiceInterface
	broadcaster *commandExecutionBroadcasterMock
}

func TestCommandExecutionServiceTestSuite(t *testing.T) {
	suite.Run(t, new(CommandExecutionServiceTestSuite))
}

func (suite *CommandExecutionServiceTestSuite) SetupTest() {
	suite.broadcaster = &commandExecutionBroadcasterMock{}
	suite.service = service.NewCommandExecutionService(suite.broadcaster)
}

func (suite *CommandExecutionServiceTestSuite) waitForCompletion(executionID string, timeout time.Duration) dto.CommandExecutionSnapshot {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		snapshot, ok := suite.service.GetSnapshot(executionID)
		if ok && !snapshot.Running {
			return snapshot
		}
		time.Sleep(10 * time.Millisecond)
	}
	suite.T().Fatalf("execution %s did not complete within %s", executionID, timeout)
	return dto.CommandExecutionSnapshot{}
}

func (suite *CommandExecutionServiceTestSuite) TestStart_StreamsStdoutAndStderrAndStoresSnapshot() {
	executionID, err := suite.service.Start(
		context.Background(),
		"test-command",
		"Test Command",
		"sh",
		"-c",
		"echo stdout-line; echo stderr-line 1>&2",
	)
	require.NoError(suite.T(), err)
	require.NotEmpty(suite.T(), executionID)

	snapshot := suite.waitForCompletion(executionID, 2*time.Second)
	require.False(suite.T(), snapshot.Running)
	require.Len(suite.T(), snapshot.Lines, 2)

	channels := []dto.CommandOutputChannel{snapshot.Lines[0].Channel, snapshot.Lines[1].Channel}
	require.Contains(suite.T(), channels, dto.CommandOutputChannelStdout)
	require.Contains(suite.T(), channels, dto.CommandOutputChannelStderr)

	messages := suite.broadcaster.Messages()
	require.NotEmpty(suite.T(), messages)

	var started bool
	var terminated bool
	for _, message := range messages {
		switch message.(type) {
		case dto.CommandStartedNotification:
			started = true
		case dto.CommandTerminatedNotification:
			terminated = true
		}
	}
	require.True(suite.T(), started, "expected command_started notification")
	require.True(suite.T(), terminated, "expected command_terminated notification")
}

func (suite *CommandExecutionServiceTestSuite) TestStart_TrimsSnapshotTo500Lines() {
	executionID, err := suite.service.Start(
		context.Background(),
		"trim-buffer",
		"Trim Buffer",
		"sh",
		"-c",
		"i=1; while [ $i -le 520 ]; do echo line-$i; i=$((i+1)); done",
	)
	require.NoError(suite.T(), err)

	snapshot := suite.waitForCompletion(executionID, 3*time.Second)
	require.Len(suite.T(), snapshot.Lines, 500)
	require.True(suite.T(), strings.HasPrefix(snapshot.Lines[0].Line, "line-"))
	require.Equal(suite.T(), "line-21", snapshot.Lines[0].Line)
	require.Equal(suite.T(), "line-520", snapshot.Lines[len(snapshot.Lines)-1].Line)
}

func (suite *CommandExecutionServiceTestSuite) TestStart_ReturnsErrorWhenCommandIsMissing() {
	executionID, err := suite.service.Start(
		context.Background(),
		"missing-command",
		"Missing",
		"definitely-missing-command-binary",
	)
	require.Error(suite.T(), err)
	require.Empty(suite.T(), executionID)

	messages := suite.broadcaster.Messages()
	require.NotEmpty(suite.T(), messages)

	last, ok := messages[len(messages)-1].(dto.CommandTerminatedNotification)
	require.True(suite.T(), ok)
	require.False(suite.T(), last.Success)
	require.Equal(suite.T(), -1, last.ExitCode)
	require.NotEmpty(suite.T(), last.Error)
}

func (suite *CommandExecutionServiceTestSuite) TestExecute_ReturnsCompletedSnapshot() {
	snapshot, err := suite.service.Execute(
		context.Background(),
		"execute-sync",
		"Execute Sync",
		"sh",
		"-c",
		"echo sync-line",
	)
	require.NoError(suite.T(), err)
	require.False(suite.T(), snapshot.Running)
	require.True(suite.T(), snapshot.Success)
	require.NotEmpty(suite.T(), snapshot.ExecutionID)
	require.NotEmpty(suite.T(), snapshot.Lines)
}

func (suite *CommandExecutionServiceTestSuite) TestExecute_ReturnsErrorOnFailure() {
	snapshot, err := suite.service.Execute(
		context.Background(),
		"execute-fail",
		"Execute Fail",
		"sh",
		"-c",
		"echo fail-line; exit 7",
	)
	require.Error(suite.T(), err)
	require.False(suite.T(), snapshot.Running)
	require.False(suite.T(), snapshot.Success)
	require.Equal(suite.T(), 7, snapshot.ExitCode)
}

var _ service.BroadcasterServiceInterface = (*commandExecutionBroadcasterMock)(nil)
