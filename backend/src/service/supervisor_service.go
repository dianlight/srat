package service

import (
	"context"
	"encoding/json"
	"sync"

	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/mount"
	"github.com/dianlight/srat/repository"
	"github.com/xorcare/pointer"
)

var _supervisor_api_mutex sync.Mutex

type SupervisorServiceInterface interface {
	NetworkMountShare(dto.SharedResource) errors.E
	NetworkUnmountShare(shareName string) errors.E
	NetworkGetAllMounted() (mounts map[string]mount.Mount, err errors.E)
}

type SupervisorService struct {
	prop_repo        repository.PropertyRepositoryInterface
	apiContext       context.Context
	apiContextCancel context.CancelFunc
	mount_client     mount.ClientWithResponsesInterface
	//	supervisor_mounts map[string]mount.Mount // Changed to a map
	state *dto.ContextState
}

type SupervisorServiceParams struct {
	fx.In
	ApiContext       context.Context
	ApiContextCancel context.CancelFunc
	MountClient      mount.ClientWithResponsesInterface `optional:"true"`
	PropertyRepo     repository.PropertyRepositoryInterface
	State            *dto.ContextState
}

func NewSupervisorService(in SupervisorServiceParams) SupervisorServiceInterface {
	p := &SupervisorService{}
	p.apiContext = in.ApiContext
	p.apiContextCancel = in.ApiContextCancel
	p.mount_client = in.MountClient
	p.prop_repo = in.PropertyRepo
	p.state = in.State
	//	p.supervisor_mounts = make(map[string]mount.Mount)
	return p
}

func (self *SupervisorService) NetworkGetAllMounted() (mounts map[string]mount.Mount, err errors.E) {
	_supervisor_api_mutex.Lock()
	defer _supervisor_api_mutex.Unlock()
	if self.state.HACoreReady == false {
		return nil, errors.Errorf("HA Core is not ready")
	}

	if self.state.SupervisorURL != "demo" {
		resp, err := self.mount_client.GetMountsWithResponse(self.apiContext)
		if err != nil {
			return nil, errors.Errorf("Error getting mounts from ha_supervisor: %w", err)
		}
		if resp.StatusCode() != 200 {
			return nil, errors.Errorf("Error getting mounts from ha_supervisor: %d %#v", resp.StatusCode(), string(resp.Body))
		}
		mounts = make(map[string]mount.Mount) // Initialize the map
		for _, mnt := range *resp.JSON200.Data.Mounts {
			if *mnt.Server == self.state.AddonIpAddress {
				mounts[*mnt.Name] = mnt // Populate the map
			}
		}
	}
	return mounts, nil
}

func (self *SupervisorService) NetworkMountShare(share dto.SharedResource) errors.E {
	if self.state.HACoreReady == false {
		return errors.Errorf("HA Core is not ready")
	}

	mounts, err := self.NetworkGetAllMounted()
	if err != nil {
		return err
	}
	conv := converter.HaSupervisorToDtoImpl{}

	mountUsername := pointer.String("_ha_mount_user_")
	pwd, err := self.prop_repo.Value("_ha_mount_user_password_", true)
	if err != nil {
		return errors.Errorf("Error getting password for mount %s from ha_supervisor: %w", share.Name, err)
	}
	mountPassword := pointer.String(pwd.(string))

	rmount, ok := mounts[share.Name]
	if !ok {
		// new mount
		rmount = mount.Mount{}
		conv.SharedResourceToMount(share, &rmount)
		rmount.Username = mountUsername
		rmount.Password = mountPassword
		rmount.Server = &self.state.AddonIpAddress
		rmount.Type = pointer.Any(mount.MountType("cifs")).(*mount.MountType)

		resp, err := self.mount_client.CreateMountWithResponse(self.apiContext, rmount)
		if err != nil {
			return errors.Errorf("Error creating mount %s from ha_supervisor: %w", share.Name, err)
		}
		if resp.StatusCode() != 200 {
			rjson, _ := json.Marshal(rmount)
			//slog.Error("Error creating mount from ha_supervisor", "share", share, "req", string(rjson), "resp", string(resp.Body))
			return errors.Errorf("Error creating mount %s from ha_supervisor: %d \nReq:%#v\nResp:%#v", *rmount.Name, resp.StatusCode(), string(rjson), string(resp.Body))
		}
	} else if string(share.Usage) != string(*rmount.Usage) || *rmount.State != "active" {
		conv.SharedResourceToMount(share, &rmount)
		rmount.Username = mountUsername
		rmount.Password = mountPassword
		resp, err := self.mount_client.UpdateMountWithResponse(self.apiContext, *rmount.Name, rmount)
		if err != nil {
			return errors.Errorf("Error updating mount %s from ha_supervisor: %w", *rmount.Name, err)
		}
		if resp.StatusCode() != 200 {
			return errors.Errorf("Error updating mount %s from ha_supervisor: %d %#v", *rmount.Name, resp.StatusCode(), string(resp.Body))
		}
	}
	return nil
}

func (self *SupervisorService) NetworkUnmountShare(shareName string) errors.E {
	if self.state.HACoreReady == false {
		return errors.Errorf("HA Core is not ready")
	}
	resp, err := self.mount_client.RemoveMountWithResponse(self.apiContext, shareName)
	if err != nil {
		return errors.Errorf("Error unmounting share %s from ha_supervisor: %w", shareName, err)
	}
	if resp.StatusCode() != 200 {
		return errors.Errorf("Error unmounting share %s from ha_supervisor: %d %#v", shareName, resp.StatusCode(), resp)
	}
	return nil
}

/* // NetworkGetAllMounted retrieves all mounts currently known by the supervisor.
func (self *SupervisorService) NetworkGetAllMounted() ([]mount.Mount, errors.E) {
	if err := self.refreshNetworkMountShare(); err != nil {
		return nil, errors.Wrap(err, "failed to refresh supervisor mounts")
	}
	_supervisor_api_mutex.Lock()
	defer _supervisor_api_mutex.Unlock()

	allMounts := make([]mount.Mount, 0, len(self.supervisor_mounts))
	for _, mnt := range self.supervisor_mounts {
		allMounts = append(allMounts, mnt)
	}
	return allMounts, nil
} */

/* // NetworkGetMountByName retrieves a specific mount by its name from the supervisor.
func (self *SupervisorService) NetworkGetMountByName(name string) (*mount.Mount, errors.E) {
	if err := self.refreshNetworkMountShare(); err != nil {
		return nil, errors.Wrapf(err, "failed to refresh supervisor mounts before getting share '%s'", name)
	}
	_supervisor_api_mutex.Lock()
	defer _supervisor_api_mutex.Unlock()

	mnt, ok := self.supervisor_mounts[name]
	if !ok {
		return nil, nil // errors.WithDetails(dto.ErrorDeviceNotFound, "Name", name, "Message", "supervisor mount not found")
	}
	return &mnt, nil
}
*/
