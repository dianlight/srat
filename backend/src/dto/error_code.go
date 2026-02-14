package dto

import (
	"github.com/danielgtaylor/huma/v2"
	"gitlab.com/tozd/go/errors"
)

type ErrorDataModel huma.ErrorModel

var ErrorNotFound = errors.Base("Not Found")
var ErrorMountFail = errors.Base("Mount Fail")
var ErrorUnmountFail = errors.Base("Umount Fail")
var ErrorDeviceNotFound = errors.Base("Device not found")
var ErrorInvalidParameter = errors.Base("Invalid parameter")
var ErrorDatabaseError = errors.Base("Database error")
var ErrorShareNotFound = errors.Base("Share not found")
var ErrorShareAlreadyExists = errors.Base("Share already exists")
var ErrorAlreadyMounted = errors.Base("Already mounted")
var ErrorUserAlreadyExists = errors.Base("User already exists")
var ErrorUserNotFound = errors.Base("User not found")
var ErrorNoUpdateAvailable = errors.Base("No update available for the specified channel and architecture")
var ErrorSMARTNotSupported = errors.Base("SMART not supported for this device")
var ErrorHDIdleNotSupported = errors.Base("HD Idle not supported for this device")
var ErrorOperationNotPermittedInProtectedMode = errors.Base("Operation not permitted in Protected mode")
var ErrorInvalidStateForOperation = errors.Base("Invalid state for operation")
var ErrorSMARTOperationFailed = errors.Base("SMART operation failed")
var ErrorSMARTTestInProgress = errors.Base("SMART test already in progress")
var ErrorConflict = errors.Base("Operation conflict")
var ErrorUnsupportedFilesystem = errors.Base("Unsupported filesystem type")
