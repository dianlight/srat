package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/require"
)

func newFormatTestDevice(t *testing.T, content []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "device.img")
	require.NoError(t, os.WriteFile(path, content, 0o600))
	return path
}

func newFormatTestRunner(snapshot dto.CommandExecutionSnapshot) *fakeCommandRunner {
	return &fakeCommandRunner{
		lookPath: func(command string) (string, error) {
			return command, nil
		},
		start: func(context.Context, string, string, string, ...string) (string, error) {
			return "format-execution", nil
		},
		getSnapshot: func(string) (dto.CommandExecutionSnapshot, bool) {
			return snapshot, true
		},
	}
}

func collectFormatStatuses() (*[]string, dto.ProgressCallback) {
	statuses := make([]string, 0, 4)
	return &statuses, func(status string, _ int, _ []string) {
		statuses = append(statuses, status)
	}
}

func TestXfsFormat_NonZeroExitReturnsError(t *testing.T) {
	adapter := NewXfsAdapter().(*XfsAdapter)
	adapter.SetCommandRunner(newFormatTestRunner(dto.CommandExecutionSnapshot{
		Running:  false,
		Success:  false,
		ExitCode: 1,
		Error:    "exit status 1",
		Lines: []dto.CommandOutputLineSnapshot{{
			Channel: dto.CommandOutputChannelStderr,
			Line:    "Usage: mkfs.xfs ...",
		}},
	}))

	statuses, progress := collectFormatStatuses()
	err := adapter.Format(
		context.Background(),
		newFormatTestDevice(t, make([]byte, 4096)),
		dto.FormatOptions{Force: true},
		progress,
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "Usage")
	require.Contains(t, *statuses, "failure")
	require.NotContains(t, *statuses, "success")
}

func TestXfsFormat_SuccessWithoutXfsSignatureReturnsError(t *testing.T) {
	adapter := NewXfsAdapter().(*XfsAdapter)
	adapter.SetCommandRunner(newFormatTestRunner(dto.CommandExecutionSnapshot{
		Running:  false,
		Success:  true,
		ExitCode: 0,
	}))

	// mkfs.xfs reported success but left a blank (non-XFS) device behind.
	statuses, progress := collectFormatStatuses()
	err := adapter.Format(
		context.Background(),
		newFormatTestDevice(t, make([]byte, 4096)),
		dto.FormatOptions{Force: true},
		progress,
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "does not contain an XFS filesystem")
	require.Contains(t, *statuses, "failure")
	require.NotContains(t, *statuses, "success")
}

func TestXfsFormat_SuccessWithUnreadableDeviceReturnsError(t *testing.T) {
	adapter := NewXfsAdapter().(*XfsAdapter)
	adapter.SetCommandRunner(newFormatTestRunner(dto.CommandExecutionSnapshot{
		Running:  false,
		Success:  true,
		ExitCode: 0,
	}))

	statuses, progress := collectFormatStatuses()
	err := adapter.Format(
		context.Background(),
		filepath.Join(t.TempDir(), "missing-device.img"),
		dto.FormatOptions{Force: true},
		progress,
	)

	require.Error(t, err)
	require.Contains(t, *statuses, "failure")
	require.NotContains(t, *statuses, "success")
}

func TestXfsFormat_SuccessWithXfsSignatureSucceeds(t *testing.T) {
	adapter := NewXfsAdapter().(*XfsAdapter)
	adapter.SetCommandRunner(newFormatTestRunner(dto.CommandExecutionSnapshot{
		Running:  false,
		Success:  true,
		ExitCode: 0,
	}))

	device := make([]byte, 4096)
	copy(device, []byte{'X', 'F', 'S', 'B'})

	statuses, progress := collectFormatStatuses()
	err := adapter.Format(
		context.Background(),
		newFormatTestDevice(t, device),
		dto.FormatOptions{Force: true},
		progress,
	)

	require.NoError(t, err)
	require.Contains(t, *statuses, "success")
	require.NotContains(t, *statuses, "failure")
}
