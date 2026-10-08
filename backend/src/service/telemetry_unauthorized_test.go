package service

import (
	"testing"

	sentry "github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldDropSentryEvent_Unauthorized(t *testing.T) {
	assert.False(t, shouldDropSentryEvent(nil))
	assert.True(t, shouldDropSentryEvent(&sentry.Event{Message: "Unauthorized access from"}))
	assert.True(t, shouldDropSentryEvent(&sentry.Event{Message: "§ Unauthorized access from"}))
	assert.True(t, shouldDropSentryEvent(&sentry.Event{
		Exception: []sentry.Exception{{Value: "§ Unauthorized access from"}},
	}))
	assert.False(t, shouldDropSentryEvent(&sentry.Event{Message: "real boom"}))
}

func TestSentryBeforeSend_DropsUnauthorizedKeepsRest(t *testing.T) {
	dropped := sentryBeforeSend(&sentry.Event{Message: "§ Unauthorized access from"}, nil)
	assert.Nil(t, dropped)

	kept := sentryBeforeSend(&sentry.Event{Message: "real boom", User: sentry.User{IPAddress: "1.2.3.4"}}, nil)
	require.NotNil(t, kept)
	assert.Empty(t, kept.User.IPAddress)
}

func TestShouldDropSentryEvent_ProtectedMode(t *testing.T) {
	assert.True(t, shouldDropSentryEvent(&sentry.Event{Message: "Operation not permitted in Protected mode"}))
	assert.True(t, shouldDropSentryEvent(&sentry.Event{
		Exception: []sentry.Exception{{Value: "Operation not permitted in Protected mode"}},
	}))
	assert.True(t, shouldDropSentryEvent(&sentry.Event{
		Exception: []sentry.Exception{{Value: "Failed to mount volume on event: Operation not permitted in Protected mode"}},
	}))
	assert.False(t, shouldDropSentryEvent(&sentry.Event{Message: "real boom"}))
}

// TestShouldDropSentryEvent_ProtectedModeMixedKeeps ensures a Sentry event
// carrying both the guard and an unrelated error is retained so the real
// failure still reaches Sentry.
func TestShouldDropSentryEvent_ProtectedModeMixedKeeps(t *testing.T) {
	assert.False(t, shouldDropSentryEvent(&sentry.Event{
		Message:   "Failed to mount volume on event",
		Exception: []sentry.Exception{{Value: "Operation not permitted in Protected mode"}, {Value: "disk I/O error"}},
	}))
	assert.False(t, shouldDropSentryEvent(&sentry.Event{
		Message:   "Operation not permitted in Protected mode",
		Exception: []sentry.Exception{{Value: "disk I/O error"}},
	}))
}

func TestSentryBeforeSend_DropsProtectedMode(t *testing.T) {
	dropped := sentryBeforeSend(&sentry.Event{Message: "Operation not permitted in Protected mode"}, nil)
	assert.Nil(t, dropped)
}

func TestShouldDropSentryEvent_AlreadyMounted(t *testing.T) {
	assert.True(t, shouldDropSentryEvent(&sentry.Event{Message: "Already mounted"}))
	assert.True(t, shouldDropSentryEvent(&sentry.Event{Message: "mount /mnt/x: device or resource busy"}))
	assert.True(t, shouldDropSentryEvent(&sentry.Event{
		Exception: []sentry.Exception{{Value: "Mount Fail: mount /mnt/x: device or resource busy"}},
	}))
	assert.False(t, shouldDropSentryEvent(&sentry.Event{Message: "real boom"}))
}

func TestSentryBeforeSend_DropsAlreadyMounted(t *testing.T) {
	dropped := sentryBeforeSend(&sentry.Event{Message: "Already mounted"}, nil)
	assert.Nil(t, dropped)
}
