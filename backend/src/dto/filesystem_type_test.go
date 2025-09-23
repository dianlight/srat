package dto

import (
	"testing"

	"syscall"

	"github.com/stretchr/testify/assert"
)

func TestMountFlags_Scan_Int(t *testing.T) {
	var mf MountFlags
	value := int(syscall.MS_RDONLY | syscall.MS_NOATIME)
	err := mf.Scan(value)
	assert.NoError(t, err)
	assert.Len(t, mf, 2)
	assert.Contains(t, mf, MountFlag{Name: "ro", NeedsValue: false})
	assert.Contains(t, mf, MountFlag{Name: "noatime", NeedsValue: false})
}

func TestMountFlags_Scan_Uintptr(t *testing.T) {
	var mf MountFlags
	value := uintptr(syscall.MS_NOSUID | syscall.MS_NODEV)
	err := mf.Scan(value)
	assert.NoError(t, err)
	assert.Len(t, mf, 2)
	assert.Contains(t, mf, MountFlag{Name: "nosuid", NeedsValue: false})
	assert.Contains(t, mf, MountFlag{Name: "nodev", NeedsValue: false})
}

func TestMountFlags_Scan_Int64(t *testing.T) {
	var mf MountFlags
	value := int64(syscall.MS_NODEV | syscall.MS_BIND)
	err := mf.Scan(value)
	assert.NoError(t, err)
	assert.Len(t, mf, 2)
	assert.Contains(t, mf, MountFlag{Name: "nodev", NeedsValue: false})
	assert.Contains(t, mf, MountFlag{Name: "bind", NeedsValue: false})
}

func TestMountFlags_Scan_SliceString(t *testing.T) {
	var mf MountFlags
	value := []string{"ro", "uid=1000", "noatime", "gid=1001"}
	err := mf.Scan(value)
	assert.NoError(t, err)
	assert.Len(t, mf, 4)
	assert.Contains(t, mf, MountFlag{Name: "ro", NeedsValue: false})
	assert.Contains(t, mf, MountFlag{Name: "uid", NeedsValue: true, FlagValue: "1000"})
	assert.Contains(t, mf, MountFlag{Name: "noatime", NeedsValue: false})
	assert.Contains(t, mf, MountFlag{Name: "gid", NeedsValue: true, FlagValue: "1001"})
}

func TestMountFlags_Scan_String(t *testing.T) {
	var mf MountFlags
	value := "ro,uid=1000,noatime,gid=1001"
	err := mf.Scan(value)
	assert.NoError(t, err)
	assert.Len(t, mf, 4)
	assert.Contains(t, mf, MountFlag{Name: "ro", NeedsValue: false})
	assert.Contains(t, mf, MountFlag{Name: "uid", NeedsValue: true, FlagValue: "1000"})
	assert.Contains(t, mf, MountFlag{Name: "noatime", NeedsValue: false})
	assert.Contains(t, mf, MountFlag{Name: "gid", NeedsValue: true, FlagValue: "1001"})
}

func TestMountFlags_Scan_MapStringString(t *testing.T) {
	var mf MountFlags
	value := map[string]string{
		"ro":      "",
		"uid":     "1000",
		"noatime": "",
		"gid":     "1001",
	}
	err := mf.Scan(value)
	assert.NoError(t, err)
	assert.Len(t, mf, 4)
	assert.Contains(t, mf, MountFlag{Name: "ro", NeedsValue: false})
	assert.Contains(t, mf, MountFlag{Name: "uid", NeedsValue: true, FlagValue: "1000"})
	assert.Contains(t, mf, MountFlag{Name: "noatime", NeedsValue: false})
	assert.Contains(t, mf, MountFlag{Name: "gid", NeedsValue: true, FlagValue: "1001"})
}

func TestMountFlags_Scan_InvalidType(t *testing.T) {
	var mf MountFlags
	value := 3.14 // float64, invalid type
	err := mf.Scan(value)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid value type for MountFlags: float64")
	assert.Empty(t, mf)
}

func TestMountFlags_Scan_Int_NoMatchingFlags(t *testing.T) {
	var mf MountFlags
	value := int(0) // No flags set
	err := mf.Scan(value)
	assert.NoError(t, err)
	assert.Empty(t, mf)
}

func TestMountFlags_Scan_SliceString_UnknownFlagWithValue(t *testing.T) {
	var mf MountFlags
	value := []string{"unknown=val", "ro"}
	err := mf.Scan(value)
	assert.NoError(t, err)
	assert.Len(t, mf, 2)
	assert.Contains(t, mf, MountFlag{Name: "unknown", NeedsValue: true, FlagValue: "val"})
	assert.Contains(t, mf, MountFlag{Name: "ro", NeedsValue: false})
}
