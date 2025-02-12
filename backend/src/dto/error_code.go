package dto

type errorCode int // ErrorMessage[string],Recoverable[bool]

//go:generate go run github.com/zarldev/goenums@latest error_code.go
const (
	unknown              errorCode = iota // invalid
	generic_error                         // "An unexpected error occurred",false
	json_marshal_error                    // "Unable to marshal JSON: {{.Error}}",false
	json_unmarshal_error                  // "Unable to unmarshal JSON: {{.Error}}",false
	invalid_parameter                     // "Invalid parameter: {{.Key}}. {{.Message}}",false
	mount_fail                            // "Unable to mount {{.Device}} on {{.Path}}. {{.Message}}",false
	unmount_fail                          // "Unable to unmount {{.ID}}. {{.Message}}",false
	device_not_found                      // "Device not found {{.DeviceID}}",false
)

var ()
