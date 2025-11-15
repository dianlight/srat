package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/homeassistant/mount"
	"github.com/dianlight/srat/repository"
	"github.com/xorcare/pointer"
)

var _supervisor_api_mutex sync.Mutex

type SupervisorServiceInterface interface {
	NetworkMountShare(dto.SharedResource) errors.E
	NetworkUnmountShare(shareName string) errors.E
	NetworkGetAllMounted() (mounts map[string]mount.Mount, err errors.E)
	NetworkUnmountAllShares() errors.E
}

type SupervisorService struct {
	prop_repo        repository.PropertyRepositoryInterface
	apiContext       context.Context
	apiContextCancel context.CancelFunc
	mount_client     mount.ClientWithResponsesInterface
	state            *dto.ContextState
	share_service    ShareServiceInterface
	eventBus         events.EventBusInterface
}

type SupervisorServiceParams struct {
	fx.In
	ApiContext       context.Context
	ApiContextCancel context.CancelFunc
	MountClient      mount.ClientWithResponsesInterface `optional:"true"`
	PropertyRepo     repository.PropertyRepositoryInterface
	State            *dto.ContextState
	ShareService     ShareServiceInterface
	EventBus         events.EventBusInterface
}

func NewSupervisorService(lc fx.Lifecycle, in SupervisorServiceParams) SupervisorServiceInterface {
	p := &SupervisorService{}
	p.apiContext = in.ApiContext
	p.apiContextCancel = in.ApiContextCancel
	p.mount_client = in.MountClient
	p.prop_repo = in.PropertyRepo
	p.state = in.State
	p.share_service = in.ShareService
	p.eventBus = in.EventBus
	unsubscribe := make([]func(), 2)
	unsubscribe[0] = p.eventBus.OnDirtyData(func(event events.DirtyDataEvent) {
		slog.Debug("DirtyDataService received DirtyData event", "tracker", event.DataDirtyTracker)
		if event.Type == events.EventTypes.CLEAN {
			p.mountHaStorage()
		}
	})
	unsubscribe[1] = p.eventBus.OnShare(func(event events.ShareEvent) {
		if event.Type == events.EventTypes.REMOVE {
			err := p.NetworkUnmountShare(event.Share.Name)
			if err != nil {
				slog.Error("Error unmounting share from ha_supervisor", "share", event.Share.Name, "err", err)
			}
		} else if event.Type == events.EventTypes.UPDATE &&
			(event.Share.Disabled != nil && *event.Share.Disabled == true) {
			err := p.NetworkUnmountShare(event.Share.Name)
			if err != nil {
				slog.Error("Error unmounting share from ha_supervisor", "share", event.Share.Name, "err", err)
			}
		} else if event.Type == events.EventTypes.UPDATE &&
			(event.Share.Usage == dto.UsageAsInternal ||
				event.Share.Usage == dto.UsageAsNone) {
			err := p.NetworkUnmountShare(event.Share.Name)
			if err != nil {
				slog.Error("Error unmounting share from ha_supervisor", "share", event.Share.Name, "err", err)
			}
		}
	})

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			return nil
		},
		OnStop: func(context.Context) error {
			for _, unsub := range unsubscribe {
				unsub()
			}
			return nil
		},
	})
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
			// If we get a 400 error, it might be because a stale systemd unit exists
			// Try to remove it and retry the mount creation
			if resp.StatusCode() == 400 {
				// Attempt to remove the potentially stale mount
				removeResp, removeErr := self.mount_client.RemoveMountWithResponse(self.apiContext, share.Name)
				if removeErr == nil && removeResp.StatusCode() == 200 {
					// Successfully removed, retry creation
					retryResp, retryErr := self.mount_client.CreateMountWithResponse(self.apiContext, rmount)
					if retryErr != nil {
						return errors.Errorf("Error creating mount %s from ha_supervisor after retry: %w", share.Name, retryErr)
					}
					if retryResp.StatusCode() == 200 {
						// Success on retry
						return nil
					}
					// Retry also failed
					rjson, _ := json.Marshal(rmount)
					return errors.Errorf("Error creating mount %s from ha_supervisor after removing stale mount: %d \nReq:%#v\nResp:%#v", *rmount.Name, retryResp.StatusCode(), string(rjson), string(retryResp.Body))
				}
			}
			// Original error or retry strategy didn't work
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
			// If we get a 400 error, it might be because the systemd unit is in a stale state
			// Try to remove it and recreate the mount (similar to create path)
			if resp.StatusCode() == 400 {
				// Attempt to remove the potentially stale mount
				removeResp, removeErr := self.mount_client.RemoveMountWithResponse(self.apiContext, share.Name)
				if removeErr == nil && removeResp.StatusCode() == 200 {
					// Successfully removed, retry by creating a new mount
					newMount := mount.Mount{}
					conv.SharedResourceToMount(share, &newMount)
					newMount.Username = mountUsername
					newMount.Password = mountPassword
					newMount.Server = &self.state.AddonIpAddress
					newMount.Type = pointer.Any(mount.MountType("cifs")).(*mount.MountType)

					retryResp, retryErr := self.mount_client.CreateMountWithResponse(self.apiContext, newMount)
					if retryErr != nil {
						return errors.Errorf("Error recreating mount %s from ha_supervisor after update failure: %w", share.Name, retryErr)
					}
					if retryResp.StatusCode() == 200 {
						// Success on retry
						return nil
					}
					// Retry also failed
					rjson, _ := json.Marshal(newMount)
					return errors.Errorf("Error recreating mount %s from ha_supervisor after removing stale mount: %d \nReq:%#v\nResp:%#v", share.Name, retryResp.StatusCode(), string(rjson), string(retryResp.Body))
				}
			}
			// Original error or retry strategy didn't work
			return errors.Errorf("Error updating mount %s from ha_supervisor: %d %#v", share.Name, resp.StatusCode(), string(resp.Body))
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

func (self *SupervisorService) mountHaStorage() errors.E {
	shares, err := self.share_service.ListShares()
	if err != nil {
		return errors.WithStack(err)
	}

	if self.state.HACoreReady && self.state.AddonIpAddress != "" {
		for _, share := range shares {
			if share.Disabled != nil && *share.Disabled {
				continue
			}
			if (share.Invalid != nil && *share.Invalid) || (share.MountPointData == nil || share.MountPointData.IsInvalid) {
				continue
			}
			switch share.Usage {
			case "media", "share", "backup":
				err = self.NetworkMountShare(share)
				if err != nil {
					slog.Error("Mounting error", "share", share, "err", err)
				}
			}
		}
	}
	return nil
}

func (self *SupervisorService) NetworkUnmountAllShares() errors.E {
	shares, err := self.share_service.ListShares()
	if err != nil {
		return errors.WithStack(err)
	}

	if self.state.HACoreReady {
		mounts, err := self.NetworkGetAllMounted()
		if err != nil {
			return errors.WithStack(err)
		}
		for _, share := range shares {
			if share.Disabled != nil && *share.Disabled {
				continue
			}
			switch share.Usage {
			case "media", "share", "backup":
				delete(mounts, share.Name)
			}
		}
		// Unmount any remaining mounts
		slog.Info("Unmounting remaining HA mounts", "count", len(mounts))
		for name := range mounts {
			err := self.NetworkUnmountShare(name)
			if err != nil {
				slog.Error("Unmounting error", "share", name, "err", err)
			}
		}
	}
	return nil
}
