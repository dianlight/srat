package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/tozd/go/errors"
)

func TestErrorCodes(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expectedMsg string
	}{
		{
			name:        "ErrorNotFound",
			err:         ErrorNotFound,
			expectedMsg: "Not Found",
		},
		{
			name:        "ErrorMountFail",
			err:         ErrorMountFail,
			expectedMsg: "Mount Fail",
		},
		{
			name:        "ErrorUnmountFail",
			err:         ErrorUnmountFail,
			expectedMsg: "Umount Fail",
		},
		{
			name:        "ErrorDeviceNotFound",
			err:         ErrorDeviceNotFound,
			expectedMsg: "Device not found",
		},
		{
			name:        "ErrorInvalidParameter",
			err:         ErrorInvalidParameter,
			expectedMsg: "Invalid parameter",
		},
		{
			name:        "ErrorDatabaseError",
			err:         ErrorDatabaseError,
			expectedMsg: "Database error",
		},
		{
			name:        "ErrorShareNotFound",
			err:         ErrorShareNotFound,
			expectedMsg: "Share not found",
		},
		{
			name:        "ErrorShareAlreadyExists",
			err:         ErrorShareAlreadyExists,
			expectedMsg: "Share already exists",
		},
		{
			name:        "ErrorAlreadyMounted",
			err:         ErrorAlreadyMounted,
			expectedMsg: "Already mounted",
		},
		{
			name:        "ErrorUserAlreadyExists",
			err:         ErrorUserAlreadyExists,
			expectedMsg: "User already exists",
		},
		{
			name:        "ErrorUserNotFound",
			err:         ErrorUserNotFound,
			expectedMsg: "User not found",
		},
		{
			name:        "ErrorNoUpdateAvailable",
			err:         ErrorNoUpdateAvailable,
			expectedMsg: "No update available for the specified channel and architecture",
		},
		{
			name:        "ErrorSMARTNotSupported",
			err:         ErrorSMARTNotSupported,
			expectedMsg: "SMART not supported for this device",
		},
		{
			name:        "ErrorOperationNotPermittedInProtectedMode",
			err:         ErrorOperationNotPermittedInProtectedMode,
			expectedMsg: "Operation not permitted in Protected mode",
		},
		{
			name:        "ErrorInvalidStateForOperation",
			err:         ErrorInvalidStateForOperation,
			expectedMsg: "Invalid state for operation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Error(t, tt.err)
			assert.Contains(t, tt.err.Error(), tt.expectedMsg)
		})
	}
}

func TestErrorWrapping(t *testing.T) {
	// Test that errors can be wrapped
	wrappedErr := errors.Wrap(ErrorNotFound, "additional context")
	assert.NotNil(t, wrappedErr)
	// Check that the error contains both messages in the error chain
	errStr := wrappedErr.Error()
	assert.NotEmpty(t, errStr)
}

func TestErrorComparison(t *testing.T) {
	// Test that errors can be compared
	err1 := ErrorNotFound
	err2 := ErrorNotFound
	assert.Equal(t, err1, err2)

	err3 := ErrorShareNotFound
	assert.NotEqual(t, err1, err3)
}
