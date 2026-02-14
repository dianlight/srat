package smartmontools

import (
	"context"
	"os/exec"
)

// Commander interface for executing commands
type Commander interface {
	Command(ctx context.Context, logger logAdapter, name string, arg ...string) Cmd
}

// Cmd interface for command execution
type Cmd interface {
	Output() ([]byte, error)
	Run() error
	CombinedOutput() ([]byte, error)
}

// execCommander implements Commander using os/exec
type execCommander struct{}

func (e execCommander) Command(ctx context.Context, logger logAdapter, name string, arg ...string) Cmd {
	logger.DebugContext(ctx, "Executing command", "name", name, "args", arg)
	cmd := exec.CommandContext(ctx, name, arg...)
	return cmd
}
