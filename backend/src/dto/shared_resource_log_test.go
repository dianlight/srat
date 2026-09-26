package dto

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSharedResourceSlogDoesNotLeakPassword ensures slog JSON output for a
// SharedResource containing a password never contains password bytes
// (issue #1129). Secret redacts via String/GoString and null MarshalJSON.
func TestSharedResourceSlogDoesNotLeakPassword(t *testing.T) {
	password := "s3cr3t-pw-1129-xyz"
	share := SharedResource{
		Name:  "log-test-share",
		Usage: "media",
		Users: []User{
			{Username: "loguser", Password: new(NewSecret(password))},
		},
		Status: &SharedResourceStatus{IsValid: true},
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	logger.Error("Mounting error", "share", share)

	out := buf.String()
	require.NotEmpty(t, out)
	assert.NotContains(t, out, password, "slog output must not contain password bytes")
}

// TestShareNameLogPattern documents the intended fix: log only share.Name
// so error logs carry the share name without user topology.
func TestShareNameLogPattern(t *testing.T) {
	password := "s3cr3t-pw-1129-xyz"
	share := SharedResource{
		Name:  "log-test-share",
		Usage: "media",
		Users: []User{
			{Username: "leakcheckuser", Password: new(NewSecret(password))},
		},
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	logger.Error("Mounting error", "share_name", share.Name)

	out := buf.String()
	require.NotEmpty(t, out)
	assert.Contains(t, out, "log-test-share")
	assert.NotContains(t, out, password)
	assert.NotContains(t, out, "leakcheckuser")
}
