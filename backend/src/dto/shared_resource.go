package dto

import (
	"os"
	"strings"
	"syscall"

	"github.com/ztrue/tracerr"
)

type HAMountUsage string // https://developers.home-assistant.io/docs/api/supervisor/models#mount

const (
	UsageAsNone   HAMountUsage = "none"
	UsageAsBackup HAMountUsage = "backup"
	UsageAsMedia  HAMountUsage = "media"
	UsageAsShare  HAMountUsage = "share"
)

type SharedResource struct {
	ID   *uint  `json:"id,omitempty"`
	Name string `json:"name,omitempty"  mapper:"mapkey"`
	//Path        string       `json:"path"`
	//FS          string       `json:"fs"`
	Disabled    bool         `json:"disabled,omitempty"`
	Users       []User       `json:"users"`
	RoUsers     []User       `json:"ro_users"`
	TimeMachine bool         `json:"timemachine,omitempty"`
	Usage       HAMountUsage `json:"usage,omitempty"`

	//DeviceId       *uint64        `json:"device_id,omitempty"`
	MountPointData *MountPointData `json:"mount_point_data,omitempty"`

	Invalid bool `json:"invalid,omitempty"`
}

func (s *SharedResource) CheckValidity() error {
	if s.Name == "" || s.MountPointData == nil {
		s.Invalid = true
		return tracerr.New("Name and Path must not be empty")
	} else {
		// Check if s.Path exists and is a directory
		// FIXME: I think is better on MountData side!
		sstat := syscall.Stat_t{}
		err := syscall.Stat(s.MountPointData.Path, &sstat)
		if os.IsNotExist(err) || !strings.HasPrefix(s.MountPointData.Path, "/") {
			s.Invalid = true
			return tracerr.Errorf("Path %s is not a valid mountpoint", s.MountPointData.Path)
		} else if err != nil {
			return tracerr.Wrap(err)
		} else if s.MountPointData.DeviceId == 0 || s.MountPointData.DeviceId != sstat.Dev {
			s.MountPointData.DeviceId = sstat.Dev
			s.Invalid = true
		}
	}
	return nil
}
