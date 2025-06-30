package dto

import "gitlab.com/tozd/go/errors"

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
var ErrorOperationNotPermittedInProtectedMode = errors.Base("Operation not permitted in Protected mode")
