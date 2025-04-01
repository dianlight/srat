package dbom

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"gitlab.com/tozd/go/errors"
	"golang.org/x/sys/unix"
)

type MounDataFlag int
type MounDataFlags []MounDataFlag

const (
	// Old Flags 0x0000ffff
	MS_RDONLY      MounDataFlag = unix.MS_RDONLY      // Mount read only
	MS_NOSUID      MounDataFlag = unix.MS_NOSUID      // Ignore setuid and setgid bits
	MS_NODEV       MounDataFlag = unix.MS_NODEV       // Disallow access to device special files
	MS_NOEXEC      MounDataFlag = unix.MS_NOEXEC      // Disallow execution of binaries
	MS_SYNCHRONOUS MounDataFlag = unix.MS_SYNCHRONOUS // Write data synchronously (wait until data has been written)
	MS_REMOUNT     MounDataFlag = unix.MS_REMOUNT     // Remount the filesystem
	MS_MANDLOCK    MounDataFlag = unix.MS_MANDLOCK    // Allow mandatory locks
	MS_NOATIME     MounDataFlag = unix.MS_NOATIME     // Do not update access and modification times
	MS_NODIRATIME  MounDataFlag = unix.MS_NODIRATIME  // Do not update directory access and modification times
	MS_BIND        MounDataFlag = unix.MS_BIND        // Bind directory at differente place
	// New Flags 0xffff0000 + Magic number 0xc0ed0000
	MS_LAZYTIME MounDataFlag = unix.MS_LAZYTIME // Lazily update access and modification times
	MS_NOUSER   MounDataFlag = unix.MS_NOUSER   // Do not update user and group IDs
	MS_RELATIME MounDataFlag = unix.MS_RELATIME // Update access and modification times only when necessary
	//ReadOnlyMountPoindDataFlags MounDataFlag = unix.MS_RDONLY | unix.MS_NOATIME
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
		MS_RDONLY,      // Mount read only
		MS_NOSUID,      // Ignore setuid and setgid bits
		MS_NODEV,       // Disallow access to device special files
		MS_NOEXEC,      // Disallow execution of binaries
		MS_SYNCHRONOUS, // Write data synchronously (wait until data has been written)
		MS_REMOUNT,     // Remount the filesystem
		MS_MANDLOCK,    // Allow mandatory locks
		MS_NOATIME,     // Do not update access and modification times
		MS_NODIRATIME,  // Do not update directory access and modification times
		MS_BIND,        // Bind directory at differente place
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
		case int64:
			if value.(int64)&int64(flags) != 0 {
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
//   - driver.Value: An int64 representing the combined flags as a bitmask.
//   - error: Always nil as this operation cannot fail.
func (self MounDataFlags) Value() (driver.Value, error) {
	var flags int64 = 0
	for _, flag := range self {
		flags |= int64(flag)
	}
	return flags, nil
}

// UnmarshalJSON implements the json.Unmarshaler interface for MounDataFlags.
// It decodes a JSON-encoded byte slice into a MounDataFlags object.
//
// Parameters:
//   - b: A byte slice containing the JSON-encoded MounDataFlags data.
//
// Returns:
//   - error: An error if JSON unmarshaling fails, or nil if successful.
func (a *MounDataFlags) UnmarshalJSON(b []byte) error {
	var s []MounDataFlag
	if err := json.Unmarshal(b, &s); err != nil {
		return errors.WithStack(err)
	}
	*a = s
	return nil
}

// MarshalJSON implements the json.Marshaler interface for MounDataFlags.
// It encodes the MounDataFlags slice into a JSON-encoded byte slice.
//
// Returns:
//   - []byte: A JSON-encoded byte slice representing the MounDataFlags.
//   - error: An error if JSON marshaling fails, or nil if successful.
func (a MounDataFlags) MarshalJSON() ([]byte, error) {
	return json.Marshal([]MounDataFlag(a))
}
