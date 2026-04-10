package commandexec

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
)

type commandEventCollector struct {
	mu       sync.Mutex
	messages []any
}

func (c *commandEventCollector) Handle(_ context.Context, event events.CommandExecutionEvent) errors.E {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messages = append(c.messages, event.Message)
	return nil
}

func (c *commandEventCollector) Messages() []any {
	c.mu.Lock()
	defer c.mu.Unlock()
	copied := make([]any, len(c.messages))
	copy(copied, c.messages)
	return copied
}

type CommandExecutorTestSuite struct {
	suite.Suite
	executor  Executor
	collector *commandEventCollector
}

func TestCommandExecutorTestSuite(t *testing.T) {
	suite.Run(t, new(CommandExecutorTestSuite))
}

func (suite *CommandExecutorTestSuite) SetupTest() {
	eventBus := events.NewEventBus(context.Background())
	suite.collector = &commandEventCollector{}
	unsubscribe := eventBus.OnCommandExecution(suite.collector.Handle)
	suite.T().Cleanup(unsubscribe)
	suite.executor = NewCommandExecutor(eventBus)
}

func (suite *CommandExecutorTestSuite) waitForCompletion(executionID string, timeout time.Duration) dto.CommandExecutionSnapshot {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		snapshot, ok := suite.executor.GetSnapshot(executionID)
		if ok && !snapshot.Running {
			return snapshot
		}
		time.Sleep(10 * time.Millisecond)
	}
	suite.T().Fatalf("execution %s did not complete within %s", executionID, timeout)
	return dto.CommandExecutionSnapshot{}
}

func (suite *CommandExecutorTestSuite) TestStart_StreamsStdoutAndStderrAndStoresSnapshot() {
	executionID, err := suite.executor.Start(
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

	messages := suite.collector.Messages()
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

func (suite *CommandExecutorTestSuite) TestStart_PopulatesOutputExitCodeWhenAvailable() {
	executionID, err := suite.executor.Start(
		context.Background(),
		"exit-code-output",
		"Exit Code Output",
		"sh",
		"-c",
		"i=1; while [ $i -le 2000 ]; do echo stderr-$i 1>&2; i=$((i+1)); done; exit 7",
	)
	suite.Require().NoError(err)

	snapshot := suite.waitForCompletion(executionID, 3*time.Second)
	suite.Require().False(snapshot.Running)
	suite.Require().Equal(7, snapshot.ExitCode)

	messages := suite.collector.Messages()
	suite.Require().NotEmpty(messages)

	sawOutputWithExitCode := false
	for _, message := range messages {
		typed, ok := message.(dto.CommandOutputNotification)
		if !ok || typed.ExitCode == nil {
			continue
		}
		suite.Require().Equal(7, *typed.ExitCode)
		sawOutputWithExitCode = true
		break
	}

	suite.Require().True(sawOutputWithExitCode, "expected at least one command_output notification with exit code")
}

func (suite *CommandExecutorTestSuite) TestStart_TrimsSnapshotTo500Lines() {
	executionID, err := suite.executor.Start(
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

func (suite *CommandExecutorTestSuite) TestStart_ReturnsErrorWhenCommandIsMissing() {
	executionID, err := suite.executor.Start(
		context.Background(),
		"missing-command",
		"Missing",
		"definitely-missing-command-binary",
	)
	suite.Require().Error(err)
	suite.Require().Empty(executionID)

	messages := suite.collector.Messages()
	suite.Require().NotEmpty(messages)

	last, ok := messages[len(messages)-1].(dto.CommandTerminatedNotification)
	suite.Require().True(ok)
	suite.Require().False(last.Success)
	suite.Require().Equal(-1, last.ExitCode)
	suite.Require().NotEmpty(last.Error)
}

func (suite *CommandExecutorTestSuite) TestExecute_ReturnsCompletedSnapshot() {
	snapshot, err := suite.executor.Execute(
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

func (suite *CommandExecutorTestSuite) TestExecute_ReturnsErrorOnFailure() {
	snapshot, err := suite.executor.Execute(
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

func (suite *CommandExecutorTestSuite) TestExecuteWithInput_PassesStdinToCommand() {
	snapshot, err := suite.executor.ExecuteWithInput(
		context.Background(),
		"execute-with-input",
		"Execute With Input",
		"copilot-input",
		"sh",
		"-c",
		"read value; echo stdin:$value",
	)
	suite.Require().NoError(err)
	suite.Require().False(snapshot.Running)
	suite.Require().True(snapshot.Success)
	suite.Require().NotEmpty(snapshot.Lines)
	suite.Require().Equal("stdin:copilot-input", snapshot.Lines[len(snapshot.Lines)-1].Line)
}
