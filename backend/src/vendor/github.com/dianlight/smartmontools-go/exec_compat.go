package smartmontools

import (
	"log/slog"

	smexec "github.com/dianlight/smartmontools-go/backends/exec"
	"github.com/dianlight/tlog"
)

// ExecBackend is the default backend that shells out to the smartctl binary.
type ExecBackend = smexec.ExecBackend

// ExecBackendOption configures an ExecBackend.
type ExecBackendOption = smexec.Option

// NewExecBackend creates a new exec backend.
func NewExecBackend(opts ...ExecBackendOption) (*ExecBackend, error) {
	return smexec.New(opts...)
}

// WithExecSmartctlPath sets a custom path to the smartctl binary for ExecBackend.
func WithExecSmartctlPath(path string) ExecBackendOption {
	return smexec.WithSmartctlPath(path)
}

// WithExecCommander sets a custom commander for ExecBackend.
func WithExecCommander(commander Commander) ExecBackendOption {
	return smexec.WithCommander(commander)
}

// WithExecLogHandler sets a custom logger adapter for ExecBackend.
func WithExecLogHandler(logger LogAdapter) ExecBackendOption {
	return smexec.WithLogHandler(logger)
}

// WithExecSlogHandler sets a slog.Logger for ExecBackend.
func WithExecSlogHandler(logger *slog.Logger) ExecBackendOption {
	return smexec.WithSlogHandler(logger)
}

// WithExecTLogHandler sets a tlog.Logger for ExecBackend.
func WithExecTLogHandler(logger *tlog.Logger) ExecBackendOption {
	return smexec.WithTLogHandler(logger)
}

// DrivedbUpstreamCommit is the upstream smartmontools commit SHA from which
// the embedded drivedb.h was taken. It is re-exported from the exec backend.
const DrivedbUpstreamCommit = smexec.DrivedbUpstreamCommit

// DrivedbUpstreamDate is the commit date of DrivedbUpstreamCommit in RFC 3339 format.
// It is re-exported from the exec backend.
const DrivedbUpstreamDate = smexec.DrivedbUpstreamDate
