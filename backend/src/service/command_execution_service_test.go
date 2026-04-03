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
	suite.Require().NoError(err)
	suite.Require().NotEmpty(executionID)

	snapshot := suite.waitForCompletion(executionID, 2*time.Second)
	suite.Require().False(snapshot.Running)
	suite.Require().Len(snapshot.Lines, 2)

	channels := []dto.CommandOutputChannel{snapshot.Lines[0].Channel, snapshot.Lines[1].Channel}
	suite.Require().Contains(channels, dto.CommandOutputChannelStdout)
	suite.Require().Contains(channels, dto.CommandOutputChannelStderr)

	messages := suite.broadcaster.Messages()
	suite.Require().NotEmpty(messages)

	var started bool
	var terminated bool
	var outputChannels []dto.CommandOutputChannel
	for _, message := range messages {
		switch typed := message.(type) {
		case dto.CommandStartedNotification:
			started = true
		case dto.CommandOutputNotification:
			outputChannels = append(outputChannels, typed.Channel)
		case dto.CommandTerminatedNotification:
			terminated = true
		}
	}
	suite.Require().True(started, "expected command_started notification")
	suite.Require().True(terminated, "expected command_terminated notification")
	suite.Require().Contains(outputChannels, dto.CommandOutputChannelStdout)
	suite.Require().Contains(outputChannels, dto.CommandOutputChannelStderr)
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
	suite.Require().NoError(err)

	snapshot := suite.waitForCompletion(executionID, 3*time.Second)
	suite.Require().Len(snapshot.Lines, 500)
	suite.Require().True(strings.HasPrefix(snapshot.Lines[0].Line, "line-"))
	suite.Require().Equal("line-21", snapshot.Lines[0].Line)
	suite.Require().Equal("line-520", snapshot.Lines[len(snapshot.Lines)-1].Line)
}

func (suite *CommandExecutionServiceTestSuite) TestStart_ReturnsErrorWhenCommandIsMissing() {
	executionID, err := suite.service.Start(
		context.Background(),
		"missing-command",
		"Missing",
		"definitely-missing-command-binary",
	)
	suite.Require().Error(err)
	suite.Require().Empty(executionID)

	messages := suite.broadcaster.Messages()
	suite.Require().NotEmpty(messages)

	last, ok := messages[len(messages)-1].(dto.CommandTerminatedNotification)
	suite.Require().True(ok)
	suite.Require().False(last.Success)
	suite.Require().Equal(-1, last.ExitCode)
	suite.Require().NotEmpty(last.Error)
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
	suite.Require().NoError(err)
	suite.Require().False(snapshot.Running)
	suite.Require().True(snapshot.Success)
	suite.Require().NotEmpty(snapshot.ExecutionID)
	suite.Require().NotEmpty(snapshot.Lines)
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
	suite.Require().Error(err)
	suite.Require().False(snapshot.Running)
	suite.Require().False(snapshot.Success)
	suite.Require().Equal(7, snapshot.ExitCode)
}

var _ service.BroadcasterServiceInterface = (*commandExecutionBroadcasterMock)(nil)
