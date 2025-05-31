package service

import (
	"context"
	"sync"

	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"

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
	NetworkGetAllMounted() ([]mount.Mount, error)
	NetworkGetMountByName(name string) (*mount.Mount, error)
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

type SupervisorServiceParams struct {
	fx.In
	ApiContext       context.Context
	ApiContextCancel context.CancelFunc
	MountClient      mount.ClientWithResponsesInterface `optional:"true"`
	PropertyRepo     repository.PropertyRepositoryInterface
	StaticConfig     *dto.ContextState
	//DirtyService      DirtyDataServiceInterface
}

func NewSupervisorService(in SupervisorServiceParams) SupervisorServiceInterface {
	p := &SupervisorService{}
	p.apiContext = in.ApiContext
	p.apiContextCancel = in.ApiContextCancel
	p.mount_client = in.MountClient
	p.prop_repo = in.PropertyRepo
	p.staticConfig = in.StaticConfig
	p.supervisor_mounts = make(map[string]mount.Mount)
	//dirtyservice.AddRestartCallback(p.WriteAndRestartSambaConfig)
	return p
}

func (self *SupervisorService) refreshNetworkMountShare() error {
	_supervisor_api_mutex.Lock()
	defer _supervisor_api_mutex.Unlock()

	if self.staticConfig.SupervisorURL != "demo" {
		resp, err := self.mount_client.GetMountsWithResponse(self.apiContext)
		if err != nil {
			return errors.Errorf("Error getting mounts from ha_supervisor: %w", err)
		}
		if resp.StatusCode() != 200 {
			return errors.Errorf("Error getting mounts from ha_supervisor: %d %#v", resp.StatusCode(), string(resp.Body))
		}
		self.supervisor_mounts = make(map[string]mount.Mount) // Initialize the map
		for _, mnt := range *resp.JSON200.Data.Mounts {
			self.supervisor_mounts[*mnt.Name] = mnt // Populate the map
		}
	}
	return nil
}

func (self *SupervisorService) NetworkMountShare(share dbom.ExportedShare) error {
	self.refreshNetworkMountShare()
	conv := converter.HaSupervisorToDbomImpl{}

	mountUsername := pointer.String("_ha_mount_user_")
	pwd, err := self.prop_repo.Value("_ha_mount_user_password_", true)
	if err != nil {
		return errors.Errorf("Error getting password for mount %s from ha_supervisor: %w", share.Name, err)
	}
	mountPassword := pointer.String(pwd.(string))

	rmount, ok := self.supervisor_mounts[share.Name]
	if !ok {
		// new mount
		rmount = mount.Mount{}
		conv.ExportedShareToMount(share, &rmount)
		rmount.Username = mountUsername
		rmount.Password = mountPassword
		rmount.Server = &self.staticConfig.AddonIpAddress

		resp, err := self.mount_client.CreateMountWithResponse(self.apiContext, rmount)
		if err != nil {
			return errors.Errorf("Error creating mount %s from ha_supervisor: %w", share.Name, err)
		}
		if resp.StatusCode() != 200 {
			return errors.Errorf("Error updating mount %s from ha_supervisor: %d %#v", *rmount.Name, resp.StatusCode(), resp)
		}
	} else if string(share.Usage) != string(*rmount.Usage) || *rmount.State != "active" {
		conv.ExportedShareToMount(share, &rmount)
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

func (self *SupervisorService) NetworkUnmountShare(share dbom.ExportedShare) error {
	resp, err := self.mount_client.RemoveMountWithResponse(self.apiContext, share.Name)
	if err != nil {
		return errors.Errorf("Error unmounting share %s from ha_supervisor: %w", share.Name, err)
	}
	if resp.StatusCode() != 200 {
		return errors.Errorf("Error unmounting share %s from ha_supervisor: %d %#v", share.Name, resp.StatusCode(), resp)
	}
	return nil
}

// NetworkGetAllMounted retrieves all mounts currently known by the supervisor.
func (self *SupervisorService) NetworkGetAllMounted() ([]mount.Mount, error) {
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
}

// NetworkGetMountByName retrieves a specific mount by its name from the supervisor.
func (self *SupervisorService) NetworkGetMountByName(name string) (*mount.Mount, error) {
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
