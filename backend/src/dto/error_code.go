package dto

import "gitlab.com/tozd/go/errors"

//type errorCode int // ErrorMessage[string],HttpCode[int]

//go_:generate go run github.com/zarldev/goenums@v0.3.8 error_code.go
/*
const (
	unknown              errorCode = iota // invalid
	generic_error                         // "An unexpected error occurred",500
	json_marshal_error                    // "Unable to marshal JSON: {{.Error}}",500
	json_unmarshal_error                  // "Unable to unmarshal JSON: {{.Error}}",500
	invalid_parameter                     // "Invalid parameter: {{.Key}}. {{.Message}}",405
	mount_fail                            // "Unable to mount {{.Device}} on {{.Path}}. {{.Message}}",406
	unmount_fail                          // "Unable to unmount {{.ID}}. {{.Message}}",406
	device_not_found                      // "Device not found {{.DeviceID}}",404
	network_timeout                       // "Network operation timed out",408
	permission_denied                     // "Permission denied for {{.Action}}",403
)
*/

var ErrorMountFail = errors.Base("Mount Fail")
var ErrorUnmountFail = errors.Base("Umount Fail")
var ErrorDeviceNotFound = errors.Base("Device not found")
var ErrorInvalidParameter = errors.Base("Invalid parameter")
var ErrorDatabaseError = errors.Base("Database error")
