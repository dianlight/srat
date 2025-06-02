package dto

import "gitlab.com/tozd/go/errors"

var ErrorMountFail = errors.Base("Mount Fail")
var ErrorUnmountFail = errors.Base("Umount Fail")
var ErrorDeviceNotFound = errors.Base("Device not found")
var ErrorInvalidParameter = errors.Base("Invalid parameter")
var ErrorDatabaseError = errors.Base("Database error")
var ErrorShareNotFound = errors.Base("Share not found")
var ErrorShareAlreadyExists = errors.Base("Share already exists")
var ErrorAlreadyMounted = errors.Base("Already mounted")
