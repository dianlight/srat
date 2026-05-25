package exec

import (
	"context"
	osexec "os/exec"
)

// execCommander implements Commander using os/exec.
type execCommander struct{}

func (e execCommander) Command(ctx context.Context, logger LogAdapter, name string, arg ...string) Cmd {
	logger.DebugContext(ctx, "Executing command", "name", name, "args", arg)
	cmd := osexec.CommandContext(ctx, name, arg...)
	return cmd
}
