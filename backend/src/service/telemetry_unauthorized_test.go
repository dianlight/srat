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

func TestSentryBeforeSend_DropsProtectedMode(t *testing.T) {
	dropped := sentryBeforeSend(&sentry.Event{Message: "Operation not permitted in Protected mode"}, nil)
	assert.Nil(t, dropped)
}
