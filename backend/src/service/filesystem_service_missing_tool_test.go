package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/srat/service/filesystem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type missingToolExecutor struct {
	err error
}

func newMissingToolExecutor() *missingToolExecutor {
	return &missingToolExecutor{err: errors.New(`exec: "tune2fs": executable file not found in $PATH`)}
}

func (f *missingToolExecutor) LookPath(command string) (string, error) {
	return "", f.err
}

func (f *missingToolExecutor) Start(_ context.Context, _, _, _ string, _ ...string) (string, error) {
	return "", f.err
}

func (f *missingToolExecutor) StartQuiet(_ context.Context, _, _, _ string, _ ...string) (string, error) {
	return "", f.err
}

func (f *missingToolExecutor) StartWithInput(_ context.Context, _, _, _, _ string, _ ...string) (string, error) {
	return "", f.err
}

func (f *missingToolExecutor) StartWithInputQuiet(_ context.Context, _, _, _, _ string, _ ...string) (string, error) {
	return "", f.err
}

func (f *missingToolExecutor) Execute(_ context.Context, _, _, _ string, _ ...string) (dto.CommandExecutionSnapshot, error) {
	return dto.CommandExecutionSnapshot{ExitCode: -1}, f.err
}

func (f *missingToolExecutor) ExecuteQuiet(_ context.Context, _, _, _ string, _ ...string) (dto.CommandExecutionSnapshot, error) {
	return dto.CommandExecutionSnapshot{ExitCode: -1}, f.err
}

func (f *missingToolExecutor) ExecuteWithInput(_ context.Context, _, _, _, _ string, _ ...string) (dto.CommandExecutionSnapshot, error) {
	return dto.CommandExecutionSnapshot{ExitCode: -1}, f.err
}

func (f *missingToolExecutor) ExecuteWithInputQuiet(_ context.Context, _, _, _, _ string, _ ...string) (dto.CommandExecutionSnapshot, error) {
	return dto.CommandExecutionSnapshot{ExitCode: -1}, f.err
}

func (f *missingToolExecutor) GetSnapshot(_ string) (dto.CommandExecutionSnapshot, bool) {
	return dto.CommandExecutionSnapshot{}, false
}

func newMissingToolFilesystemService(ctx context.Context, cancel context.CancelFunc) service.FilesystemServiceInterface {
	bus := events.NewEventBus(ctx)
	return service.NewFilesystemService(ctx, cancel, bus)
}

func TestGetPartitionLabel_MissingToolReturnsUnknown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fsService := newMissingToolFilesystemService(ctx, cancel)

	reset := filesystem.SetDefaultCommandRunner(newMissingToolExecutor())
	defer reset()

	label, err := fsService.GetPartitionLabel(ctx, "/dev/missing-tool-1353-label", "ext4")
	require.NoError(t, err)
	assert.Empty(t, label)
}

func TestGetPartitionState_MissingToolReturnsUnknown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fsService := newMissingToolFilesystemService(ctx, cancel)

	reset := filesystem.SetDefaultCommandRunner(newMissingToolExecutor())
	defer reset()

	state, err := fsService.GetPartitionState(ctx, "/dev/missing-tool-1353-state", "ext4")
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Equal(t, "Unknown", state.StateDescription)
}
