package dto

import (
	"fmt"
	"strings"
	"syscall"
)

type FilesystemType struct {
	Name             string     `json:"name"`
	Type             string     `json:"type"`
	MountFlags       MountFlags `json:"mountFlags"`
	CustomMountFlags MountFlags `json:"customMountFlags"`
}

type FilesystemTypes []FilesystemType

// MountFlag represents a single mount option/flag.
type MountFlag struct {
	Name                 string `json:"name"`
	Description          string `json:"description,omitempty"`
	NeedsValue           bool   `json:"needsValue,omitempty"`             // Add NeedsValue field
	FlagValue            string `json:"value,omitempty"`                  // Add Value field for flags like "uid=<arg>"
	ValueDescription     string `json:"value_description,omitempty"`      // New field
	ValueValidationRegex string `json:"value_validation_regex,omitempty"` // New field
}

type MountFlags []MountFlag

func MountFlagsMap() map[string]uintptr {
	flagMap := map[string]uintptr{
		"ro":          syscall.MS_RDONLY,
		"nosuid":      syscall.MS_NOSUID,
		"nodev":       syscall.MS_NODEV,
		"noexec":      syscall.MS_NOEXEC,
		"sync":        syscall.MS_SYNCHRONOUS,
		"remount":     syscall.MS_REMOUNT,
		"mand":        syscall.MS_MANDLOCK,
		"dirsync":     syscall.MS_DIRSYNC,
		"noatime":     syscall.MS_NOATIME,
		"nodiratime":  syscall.MS_NODIRATIME,
		"bind":        syscall.MS_BIND,
		"rec":         syscall.MS_REC,
		"silent":      syscall.MS_SILENT,
		"posixacl":    syscall.MS_POSIXACL,
		"acl":         syscall.MS_POSIXACL, // Common alias
		"unbindable":  syscall.MS_UNBINDABLE,
		"private":     syscall.MS_PRIVATE,
		"slave":       syscall.MS_SLAVE,
		"shared":      syscall.MS_SHARED,
		"relatime":    syscall.MS_RELATIME,
		"strictatime": syscall.MS_STRICTATIME,
		// "lazytime":    syscall.MS_LAZYTIME, // Consistent with MountFlagsToSyscallFlagAndData
	}
	return flagMap
}

func (self *MountFlags) Scan(value interface{}) error {
	for nflag, flags := range MountFlagsMap() {
		switch value.(type) {
		case int:
			if value.(int)&int(flags) != 0 {
				self.Add(MountFlag{
					Name:       nflag,
					NeedsValue: false,
				})
			}
		case uintptr:
			if value.(uintptr)&uintptr(flags) != 0 {
				self.Add(MountFlag{
					Name:       nflag,
					NeedsValue: false,
				})
			}
		case int64:
			if value.(int64)&int64(flags) != 0 {
				self.Add(MountFlag{
					Name:       nflag,
					NeedsValue: false,
				})
			}
		case []string:
			svalue := value.([]string)
			for _, flag := range svalue {
				if flag == nflag {
					self.Add(MountFlag{
						Name:       nflag,
						NeedsValue: false,
					})
				} else if strings.HasPrefix(flag, nflag+"=") {
					// Extract the value after the '='
					parts := strings.SplitN(flag, "=", 2)
					if len(parts) == 2 {
						self.Add(MountFlag{
							Name:       nflag,
							NeedsValue: true,
							FlagValue:  parts[1],
						})
					}
				}
			}
		case string:
			svalue := value.(string)
			for _, flag := range strings.Split(svalue, ",") {
				if flag == nflag {
					self.Add(MountFlag{
						Name:       nflag,
						NeedsValue: false,
					})
				} else if strings.HasPrefix(flag, nflag+"=") {
					// Extract the value after the '='
					parts := strings.SplitN(flag, "=", 2)
					if len(parts) == 2 {
						self.Add(MountFlag{
							Name:       nflag,
							NeedsValue: true,
							FlagValue:  parts[1],
						})
					}
				}
			}
		default:
			return fmt.Errorf("invalid value type for MountFlags: %T", value)
		}
	}
	return nil
}

func (self *MountFlags) Add(value MountFlag) error {
	*self = append(*self, value)
	return nil
}

func (self MountFlags) UintPtrValue() (uintptr, error) {
	var flags uintptr = 0
	for _, flag := range self {
		ivalue, ok := MountFlagsMap()[flag.Name]
		if ok {
			flags |= ivalue
		} else {
			return 0, fmt.Errorf("unknown flag: %s", flag.Name)
		}
	}
	return flags, nil
}

func (self MountFlags) StringValue() string {
	dest := make([]string, 0, len(self))
	for _, flag := range self {
		if flag.NeedsValue {
			dest = append(dest, fmt.Sprintf("%s=%s", flag.Name, flag.FlagValue))
		} else {
			dest = append(dest, flag.Name)
		}
	}
	return strings.Join(dest, ",")
}
