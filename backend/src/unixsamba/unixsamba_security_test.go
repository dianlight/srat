package unixsamba

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateUnixSambaCommand_AllowsKnownCommands(t *testing.T) {
	err := validateUnixSambaCommand("pdbedit", "-L", "-v", "-u", "testuser")
	require.NoError(t, err)
}

func TestValidateUnixSambaCommand_RejectsUnknownCommand(t *testing.T) {
	err := validateUnixSambaCommand("sh", "-c", "echo unsafe")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not allowed")
}

func TestValidateUnixSambaCommand_RejectsNewlineInArguments(t *testing.T) {
	err := validateUnixSambaCommand("useradd", "valid", "arg-with\nnewline")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid newline")
}
