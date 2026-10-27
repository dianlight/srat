package service

import (
	"testing"

	sentry "github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/assert"
)

func TestShouldDropSentryEvent_MissingBinary(t *testing.T) {
	assert.True(t, shouldDropSentryEvent(&sentry.Event{
		Message: `Error executing command`,
		Exception: []sentry.Exception{{
			Value: `exec: "xfs_admin": executable file not found in $PATH`,
		}},
	}))
	assert.True(t, shouldDropSentryEvent(&sentry.Event{
		Message: `exec: "xfs_admin": executable file not found in $PATH`,
	}))
	assert.True(t, shouldDropSentryEvent(&sentry.Event{
		Exception: []sentry.Exception{{
			Value: `failed to get partition label for device: /dev/sda2: exec: "xfs_admin": executable file not found in $PATH`,
		}},
	}))
	assert.False(t, shouldDropSentryEvent(&sentry.Event{Message: "real boom"}))
	assert.False(t, shouldDropSentryEvent(&sentry.Event{
		Exception: []sentry.Exception{{Value: "disk I/O error"}},
	}))
}

func TestSentryBeforeSend_DropsMissingBinary(t *testing.T) {
	dropped := sentryBeforeSend(&sentry.Event{
		Message: "Error executing command",
		Exception: []sentry.Exception{{
			Value: `exec: "xfs_admin": executable file not found in $PATH`,
		}},
	}, nil)
	assert.Nil(t, dropped)
}

func TestIsMissingBinaryNoise(t *testing.T) {
	assert.True(t, isMissingBinaryNoise(`exec: "xfs_admin": executable file not found in $PATH`))
	assert.True(t, isMissingBinaryNoise(`command not found`))
	assert.True(t, isMissingBinaryNoise(`EXECUTABLE FILE NOT FOUND`))
	assert.False(t, isMissingBinaryNoise("permission denied"))
	assert.False(t, isMissingBinaryNoise("no such file or directory"))
	assert.False(t, isMissingBinaryNoise("real boom"))
}
