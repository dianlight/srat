package dto

import (
	"database/sql/driver"
	"fmt"

	"golang.org/x/sys/unix"
)

//go:generate go run github.com/dmarkham/enumer@v1.5.11 -type=MountFlag -json -sql -trimprefix=MF_
type MountFlag int
type MountFlags []MountFlag

const (
	// Old Flags 0x0000ffff
	MS_RDONLY      MountFlag = unix.MS_RDONLY      // "read_only","Mount read only",unix.MS_RDONLY
	MS_NOSUID      MountFlag = unix.MS_NOSUID      // "no_suid","Ignore setuid and setgid bits"
	MS_NODEV       MountFlag = unix.MS_NODEV       // "no_dev","Disallow access to device special files"
	MS_NOEXEC      MountFlag = unix.MS_NOEXEC      // "no_exec","Disallow execution of binaries"
	MS_SYNCHRONOUS MountFlag = unix.MS_SYNCHRONOUS // "sync","Write data synchronously (wait until data has been written)"
	MS_REMOUNT     MountFlag = unix.MS_REMOUNT     // "remount","Remount the filesystem"
	MS_MANDLOCK    MountFlag = unix.MS_MANDLOCK    // "mandatory_lock","Allow mandatory locks"
	MS_NOATIME     MountFlag = unix.MS_NOATIME     // "no_atime","Do not update access and modification times"
	MS_NODIRATIME  MountFlag = unix.MS_NODIRATIME  // "no_dir_atime","Do not update directory access and modification times"
	MS_BIND        MountFlag = unix.MS_BIND        // "bind","Bind directory at differente place"
	// New Flags 0xffff0000 + Magic number 0xc0ed0000
	MS_LAZYTIME MountFlag = unix.MS_LAZYTIME // "lazy_time","Lazily update access and modification times"
	MS_NOUSER   MountFlag = unix.MS_NOUSER   // "no_user","Do not update user and group IDs"
	MS_RELATIME MountFlag = unix.MS_RELATIME // "realtime","Update access and modification times only when necessary"
)

func (self *MountFlags) Scan(value interface{}) error {
	for _, flags := range MountFlagValues() {
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
		case string:
			if value.(string) == flags.String() {
				self.Add(flags)
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

func (self MountFlags) Value() (driver.Value, error) {
	var flags int64 = 0
	for _, flag := range self {
		flags |= int64(flag)
	}
	return flags, nil
}

func (self MountFlags) Strings() (dest []string) {
	for _, flag := range self {
		dest = append(dest, flag.String())
	}
	return
}
