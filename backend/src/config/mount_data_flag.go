package config

import (
	"database/sql/driver"
	"fmt"

	"golang.org/x/sys/unix"
)

type MounDataFlag int
type MounDataFlags []MounDataFlag

const (
	MS_RDONLY                   MounDataFlag = unix.MS_RDONLY
	MS_BIND                     MounDataFlag = unix.MS_BIND
	MS_LAZYTIME                 MounDataFlag = unix.MS_LAZYTIME
	MS_NOEXEC                   MounDataFlag = unix.MS_NOEXEC
	MS_NOSUID                   MounDataFlag = unix.MS_NOSUID
	MS_NOUSER                   MounDataFlag = unix.MS_NOUSER
	MS_RELATIME                 MounDataFlag = unix.MS_RELATIME
	MS_SYNC                     MounDataFlag = unix.MS_SYNC
	MS_NOATIME                  MounDataFlag = unix.MS_NOATIME
	ReadOnlyMountPoindDataFlags MounDataFlag = unix.MS_RDONLY | unix.MS_NOATIME
)

func (self *MounDataFlag) Scan(value interface{}) error {
	*self = MounDataFlag(value.(int))
	return nil
}

func (self MounDataFlag) Value() (driver.Value, error) {
	return int(self), nil
}

// All returns a slice containing all defined MounDataFlag constants.
//
// This function does not take any parameters.
//
// Returns:
//   - []MounDataFlag: A slice containing all the MounDataFlag constants defined in this package.
func (self MounDataFlags) EnumValues() []MounDataFlag {
	return []MounDataFlag{
		MS_RDONLY,
		MS_BIND,
		MS_LAZYTIME,
		MS_NOEXEC,
		MS_NOSUID,
		MS_NOUSER,
		MS_RELATIME,
		MS_SYNC,
		MS_NOATIME,
	}
}

// Add appends a new MounDataFlag to the MounDataFlags slice.
//
// Parameters:
//   - value: The MounDataFlag to be added to the slice.
//
// Returns:
//   - error: Always returns nil as this operation cannot fail.
func (self *MounDataFlags) Add(value MounDataFlag) error {
	*self = append(*self, value)
	return nil
}

// Scan implements the sql.Scanner interface for MounDataFlags.
// It converts a database value to a MounDataFlags type.
//
// Parameters:
//   - value: An interface{} that should contain the database representation of MounDataFlags.
//
// Returns:
//   - error: An error if the scan operation fails, or nil if successful.
func (self *MounDataFlags) Scan(value interface{}) error {
	for _, flags := range self.EnumValues() {
		switch value.(type) {
		case int:
			if value.(int)&int(flags) != 0 {
				self.Add(flags)
			}
		case uintptr:
			if value.(uintptr)&uintptr(flags) != 0 {
				self.Add(flags)
			}
		default:
			return fmt.Errorf("invalid value type for MounDataFlags: %T", value)
		}
	}
	return nil
}

// Value implements the driver.Valuer interface for MounDataFlags.
// It converts MounDataFlags to a value that can be stored in the database.
//
// Returns:
//   - driver.Value: An int representing the combined flags as a bitmask.
//   - error: Always nil as this operation cannot fail.
func (self MounDataFlags) Value() (driver.Value, error) {
	var flags = 0
	for _, flag := range self {
		flags |= int(flag)
	}
	return flags, nil
}
