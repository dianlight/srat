package dto

type errorCode int // ErrorMessage[string],HttpCode[int]

//go:generate go run github.com/zarldev/goenums@latest error_code.go
const (
	unknown              errorCode = iota // invalid
	generic_error                         // "An unexpected error occurred",500
	json_marshal_error                    // "Unable to marshal JSON: {{.Error}}",500
	json_unmarshal_error                  // "Unable to unmarshal JSON: {{.Error}}",500
	invalid_parameter                     // "Invalid parameter: {{.Key}}. {{.Message}}",405
	mount_fail                            // "Unable to mount {{.Device}} on {{.Path}}. {{.Message}}",406
	unmount_fail                          // "Unable to unmount {{.ID}}. {{.Message}}",406
	device_not_found                      // "Device not found {{.DeviceID}}",404
)
