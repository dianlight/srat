package service

import (
	"context"
	"sync"

	"gitlab.com/tozd/go/errors"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/mount"
	"github.com/dianlight/srat/repository"
	"github.com/xorcare/pointer"
)

var _supervisor_api_mutex sync.Mutex

type SupervisorServiceInterface interface {
	NetworkMountShare(dbom.ExportedShare) error
	NetworkUnmountShare(dbom.ExportedShare) error
}

type SupervisorService struct {
	//dirtyservice        DirtyDataServiceInterface
	//exported_share_repo repository.ExportedShareRepositoryInterface
	prop_repo         repository.PropertyRepositoryInterface
	apiContext        context.Context
	apiContextCancel  context.CancelFunc
	mount_client      mount.ClientWithResponsesInterface
	supervisor_mounts map[string]mount.Mount // Changed to a map
	staticConfig      *dto.ContextState
}

func NewSupervisorService(
	apiContext context.Context,
	staticConfig *dto.ContextState,
	apiContextCancel context.CancelFunc,
	prop_repo repository.PropertyRepositoryInterface,
	mount_client mount.ClientWithResponsesInterface,
) SupervisorServiceInterface {
	p := &SupervisorService{}
	p.apiContext = apiContext
	p.apiContextCancel = apiContextCancel
	p.mount_client = mount_client
	p.prop_repo = prop_repo
	p.staticConfig = staticConfig
	p.supervisor_mounts = make(map[string]mount.Mount)
	//dirtyservice.AddRestartCallback(p.WriteAndRestartSambaConfig)
	return p
}

func (self *SupervisorService) refreshNetworkMountShare() error {
	_supervisor_api_mutex.Lock()
	defer _supervisor_api_mutex.Unlock()

	resp, err := self.mount_client.GetMountsWithResponse(self.apiContext)
	if err != nil {
		return errors.Errorf("Error getting mounts from ha_supervisor: %w", err)
	}
	if resp.StatusCode() != 200 {
		return errors.Errorf("Error getting mounts from ha_supervisor: %d %#v", resp.StatusCode(), resp)
	}
	self.supervisor_mounts = make(map[string]mount.Mount) // Initialize the map
	for _, mnt := range *resp.JSON200.Data.Mounts {
		self.supervisor_mounts[*mnt.Name] = mnt // Populate the map
	}

	return nil
}

func (self *SupervisorService) NetworkMountShare(share dbom.ExportedShare) error {
	self.refreshNetworkMountShare()
	conv := converter.HaSupervisorToDbomImpl{}

	rmount, ok := self.supervisor_mounts[share.Name]
	if !ok {
		// new mount
		rmount = mount.Mount{}
		conv.ExportedShareToMount(share, &rmount)
		rmount.Username = pointer.String("_ha_mount_user_")
		pwd, err := self.prop_repo.Value("_ha_mount_user_password_", true)
		if err != nil {
			return errors.Errorf("Error getting password for mount %s from ha_supervisor: %w", share.Name, err)
		}
		rmount.Password = pointer.String(pwd.(string))
		rmount.Server = &self.staticConfig.AddonIpAddress

		resp, err := self.mount_client.CreateMountWithResponse(self.apiContext, rmount)
		if err != nil {
			return errors.Errorf("Error creating mount %s from ha_supervisor: %w", share.Name, err)
		}
		if resp.StatusCode() != 200 {
			return errors.Errorf("Error updating mount %s from ha_supervisor: %d %#v", *rmount.Name, resp.StatusCode(), resp)
		}
	} else if string(share.Usage) != string(*rmount.Usage) {
		conv.ExportedShareToMount(share, &rmount)
		resp, err := self.mount_client.UpdateMountWithResponse(self.apiContext, *rmount.Name, rmount)
		if err != nil {
			return errors.Errorf("Error updating mount %s from ha_supervisor: %w", *rmount.Name, err)
		}
		if resp.StatusCode() != 200 {
			return errors.Errorf("Error updating mount %s from ha_supervisor: %d %#v", *rmount.Name, resp.StatusCode(), resp)
		}
	} else if *rmount.State != "active" {
		resp, err := self.mount_client.ReloadMountWithResponse(self.apiContext, *rmount.Name)
		if err != nil {
			return errors.Errorf("Error reloading mount %s from ha_supervisor: %w", *rmount.Name, err)
		}
		if resp.StatusCode() != 200 {
			return errors.Errorf("Error reloading mount %s from ha_supervisor: %d %#v", *rmount.Name, resp.StatusCode(), resp)
		}
	}
	return nil
}

func (self *SupervisorService) NetworkUnmountShare(dbom.ExportedShare) error {
	self.refreshNetworkMountShare()
	return nil
}
