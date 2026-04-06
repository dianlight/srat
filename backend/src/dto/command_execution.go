package dto

// CommandOutputChannel identifies the source stream for a command output line.
type CommandOutputChannel string

const (
	CommandOutputChannelStdout CommandOutputChannel = "stdout"
	CommandOutputChannelStderr CommandOutputChannel = "stderr"
)

// CommandExecutionNotification is the typed payload contract for command
// lifecycle notifications emitted by the shared executor.
type CommandExecutionNotification interface {
	isCommandExecutionNotification()
}

// CommandStartedNotification announces the start of a command execution.
type CommandStartedNotification struct {
	ExecutionID string   `json:"execution_id"`
	CommandID   string   `json:"command_id"`
	Label       string   `json:"label,omitempty"`
	Command     string   `json:"command"`
	Args        []string `json:"args,omitempty"`
	StartedAt   int64    `json:"started_at"`
}

func (CommandStartedNotification) isCommandExecutionNotification() {}

// CommandOutputNotification carries one output line from an execution stream.
// ExitCode is nil while the command is still running and becomes available once
// the process has already finished.
type CommandOutputNotification struct {
	ExecutionID string               `json:"execution_id"`
	CommandID   string               `json:"command_id"`
	Channel     CommandOutputChannel `json:"channel"`
	Line        string               `json:"line"`
	Timestamp   int64                `json:"timestamp"`
	ExitCode    *int                 `json:"exit_code,omitempty"`
}

func (CommandOutputNotification) isCommandExecutionNotification() {}

// CommandTerminatedNotification announces command completion with final status.
type CommandTerminatedNotification struct {
	ExecutionID string `json:"execution_id"`
	CommandID   string `json:"command_id"`
	ExitCode    int    `json:"exit_code"`
	Success     bool   `json:"success"`
	FinishedAt  int64  `json:"finished_at"`
	Error       string `json:"error,omitempty"`
}

func (CommandTerminatedNotification) isCommandExecutionNotification() {}

// CommandOutputLineSnapshot stores one buffered output line for an execution.
type CommandOutputLineSnapshot struct {
	Channel   CommandOutputChannel `json:"channel"`
	Line      string               `json:"line"`
	Timestamp int64                `json:"timestamp"`
}

// CommandExecutionSnapshot stores the latest buffered state of an execution.
type CommandExecutionSnapshot struct {
	ExecutionID string                      `json:"execution_id"`
	CommandID   string                      `json:"command_id"`
	Label       string                      `json:"label,omitempty"`
	Command     string                      `json:"command"`
	Args        []string                    `json:"args,omitempty"`
	StartedAt   int64                       `json:"started_at"`
	FinishedAt  int64                       `json:"finished_at,omitempty"`
	Running     bool                        `json:"running"`
	ExitCode    int                         `json:"exit_code,omitempty"`
	Success     bool                        `json:"success"`
	Error       string                      `json:"error,omitempty"`
	Lines       []CommandOutputLineSnapshot `json:"lines"`
}
