package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/homeassistant/mount"
	"github.com/dianlight/tlog"
)

var _supervisor_api_mutex sync.Mutex

type SupervisorServiceInterface interface {
	NetworkMountShare(ctx context.Context, share dto.SharedResource) errors.E
	NetworkUnmountShare(ctx context.Context, shareName string) errors.E
	NetworkGetAllMounted(ctx context.Context) (mounts map[string]mount.Mount, err errors.E)
	NetworkMountAllShares(ctx context.Context) errors.E
	NetworkUnmountAllShares(ctx context.Context) errors.E
}

type SupervisorService struct {
	//prop_repo repository.PropertyRepositoryInterface
	//apiContext       context.Context
	apiContextCancel   context.CancelFunc
	mount_client       mount.ClientWithResponsesInterface
	state              *dto.ContextState
	share_service      ShareServiceInterface
	dirty_data_service DirtyDataServiceInterface
	settingService     SettingServiceInterface
	eventBus           events.EventBusInterface
}

type SupervisorServiceParams struct {
	fx.In
	ApiContext       context.Context
	ApiContextCancel context.CancelFunc
	MountClient      mount.ClientWithResponsesInterface `optional:"true"`
	//PropertyRepo     repository.PropertyRepositoryInterface
	State            *dto.ContextState
	ShareService     ShareServiceInterface
	DirtyDataService DirtyDataServiceInterface
	SettingService   SettingServiceInterface
	EventBus         events.EventBusInterface
}

func NewSupervisorService(lc fx.Lifecycle, in SupervisorServiceParams) SupervisorServiceInterface {
	p := &SupervisorService{}
	//p.apiContext = in.ApiContext
	p.dirty_data_service = in.DirtyDataService
	p.apiContextCancel = in.ApiContextCancel
	p.mount_client = in.MountClient
	//p.prop_repo = in.PropertyRepo
	p.state = in.State
	p.share_service = in.ShareService
	p.settingService = in.SettingService
	p.eventBus = in.EventBus
	unsubscribe := make([]func(), 3)
	unsubscribe[0] = p.eventBus.OnServerProccess(func(ctx context.Context, event events.ServerProcessEvent) errors.E {
		slog.DebugContext(ctx, "SupervisorService received ServerProcess event", "tracker", event.DataDirtyTracker)
		if event.Type == events.EventTypes.CLEAN {
			err := p.NetworkMountAllShares(ctx)
			if err != nil {
				slog.ErrorContext(ctx, "Error mounting HA storage shares", "err", err)
				p.eventBus.EmitHomeAssistant(events.HomeAssistantEvent{
					Event: events.Event{
						Type: events.EventTypes.ERROR,
					},
					Error: &dto.ErrorDataModel{
						Title:  "Error mounting HA storage shares",
						Detail: err.Error(),
					},
				})
				return err
			}
		}
		return nil
	})
	unsubscribe[1] = p.eventBus.OnHomeAssistant(func(ctx context.Context, event events.HomeAssistantEvent) errors.E {
		if event.Type == events.EventTypes.START && p.dirty_data_service.IsClean() {
			err := p.NetworkMountAllShares(ctx)
			if err != nil {
				slog.ErrorContext(ctx, "Error mounting HA storage shares", "err", err)
				p.eventBus.EmitHomeAssistant(events.HomeAssistantEvent{
					Event: events.Event{
						Type: events.EventTypes.ERROR,
					},
					Error: &dto.ErrorDataModel{
						Title:  "Error mounting HA storage shares",
						Detail: err.Error(),
					},
				})
				return err
			}
		}
		return nil
	})
	unsubscribe[2] = p.eventBus.OnShare(func(ctx context.Context, event events.ShareEvent) errors.E {
		if event.Type == events.EventTypes.REMOVE {
			err := p.NetworkUnmountShare(ctx, event.Share.Name)
			if err != nil {
				slog.ErrorContext(ctx, "Error unmounting share from ha_supervisor", "share", event.Share.Name, "err", err)
				p.eventBus.EmitHomeAssistant(events.HomeAssistantEvent{
					Event: events.Event{
						Type: events.EventTypes.ERROR,
					},
					Error: &dto.ErrorDataModel{
						Title:  "Error unmounting share from ha_supervisor",
						Detail: err.Error(),
					},
				})
				return err
			}
		} else if event.Type == events.EventTypes.UPDATE &&
			(event.Share.Disabled != nil && *event.Share.Disabled == true) {
			err := p.NetworkUnmountShare(ctx, event.Share.Name)
			if err != nil {
				slog.ErrorContext(ctx, "Error unmounting share from ha_supervisor", "share", event.Share.Name, "err", err)
				p.eventBus.EmitHomeAssistant(events.HomeAssistantEvent{
					Event: events.Event{
						Type: events.EventTypes.ERROR,
					},
					Error: &dto.ErrorDataModel{
						Title:  "Error unmounting share from ha_supervisor",
						Detail: err.Error(),
					},
				})
				return err
			}
		} else if event.Type == events.EventTypes.UPDATE &&
			(event.Share.Usage == dto.UsageAsInternal ||
				event.Share.Usage == dto.UsageAsNone) {
			err := p.NetworkUnmountShare(ctx, event.Share.Name)
			if err != nil {
				slog.ErrorContext(ctx, "Error unmounting share from ha_supervisor", "share", event.Share.Name, "err", err)
				p.eventBus.EmitHomeAssistant(events.HomeAssistantEvent{
					Event: events.Event{
						Type: events.EventTypes.ERROR,
					},
					Error: &dto.ErrorDataModel{
						Title:  "Error unmounting share from ha_supervisor",
						Detail: err.Error(),
					},
				})
				return err
			}
		}
		/*
			err := p.NetworkMountAllShares(ctx)
			if err != nil {
				slog.ErrorContext(ctx, "Error mounting HA storage shares", "err", err)
				return err
			}
		*/
		return nil
	})

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			serviceStart := time.Now()
			tlog.TraceContext(ctx, "=== SERVICE INIT: SupervisorService Starting ===")
			defer func() {
				tlog.TraceContext(ctx, "=== SERVICE INIT: SupervisorService Complete ===", "duration", time.Since(serviceStart))
			}()
			tlog.DebugContext(ctx, "Starting Supervisor Service")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			tlog.DebugContext(ctx, "Stopping Supervisor Service")
			for _, unsub := range unsubscribe {
				unsub()
			}
			p.NetworkUnmountAllShares(ctx)
			return nil
		},
	})
	return p
}

func (self *SupervisorService) NetworkGetAllMounted(ctx context.Context) (mounts map[string]mount.Mount, err errors.E) {
	_supervisor_api_mutex.Lock()
	defer _supervisor_api_mutex.Unlock()
	if self.state.HACoreReady == false {
		return nil, errors.Errorf("HA Core is not ready")
	}

	if self.state.SupervisorURL != "demo" {
		resp, err := self.mount_client.GetMountsWithResponse(ctx)
		if err != nil {
			return nil, errors.Errorf("Error getting mounts from ha_supervisor: %w", err)
		}
		if resp == nil {
			return nil, errors.Errorf("Error getting mounts from ha_supervisor: response is nil")
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

func (self *SupervisorService) NetworkMountShare(ctx context.Context, share dto.SharedResource) errors.E {
	return self.networkMountShareWithRetry(ctx, share, 3)
}

func (self *SupervisorService) networkMountShareWithRetry(ctx context.Context, share dto.SharedResource, retries int) errors.E {

	if retries <= 0 {
		return errors.Errorf("Exceeded maximum retries to mount share %s", share.Name)
	}

	if self.state.HACoreReady == false {
		return errors.Errorf("HA Core is not ready")
	}

	mounts, err := self.NetworkGetAllMounted(ctx)
	if err != nil {
		return err
	}
	conv := converter.HaSupervisorToDtoImpl{}

	mountUsername := new("_ha_mount_user_")
	setting, err := self.settingService.Load()
	if err != nil {
		return errors.Errorf("Error getting password for mount %s from ha_supervisor: %w", share.Name, err)
	}
	mountPassword := setting.HASmbPassword.Expose()
	useNfs := setting.HAUseNFS
	if useNfs == nil {
		return errors.Errorf("Error getting HAUseNFS setting from ha_supervisor: value is nil")
	}

	rmount, ok := mounts[share.Name]
	if !ok {
		// new mount
		rmount = mount.Mount{}
		conv.SharedResourceToMount(share, &rmount)
		rmount.Server = &self.state.AddonIpAddress

		if *useNfs {
			rmount.Type = new(mount.MountType("nfs"))
			rmount.Path = new(share.MountPointData.Path)
		} else {
			rmount.Type = new(mount.MountType("cifs"))
			rmount.Username = mountUsername
			rmount.Password = &mountPassword
		}

		resp, err := self.mount_client.CreateMountWithResponse(ctx, rmount)
		if err != nil {
			return errors.Errorf("Error creating mount %s from ha_supervisor: %w", share.Name, err)
		}
		if resp.StatusCode() != 200 {
			// If we get a 400 error, it might be because a stale systemd unit exists
			// Try to remove it and retry the mount creation
			if resp.StatusCode() == 400 {
				// Attempt to remove the potentially stale mount
				removeResp, removeErr := self.mount_client.RemoveMountWithResponse(ctx, share.Name)
				if removeErr == nil && removeResp.StatusCode() == 200 {
					// Successfully removed, retry creation
					retryResp, retryErr := self.mount_client.CreateMountWithResponse(ctx, rmount)
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
		return nil
	} else if string(share.Usage) != string(*rmount.Usage) ||
		*rmount.State != "active" ||
		(*useNfs && *rmount.Type == "cifs") ||
		(!*useNfs && *rmount.Type == "nfs") {
		conv.SharedResourceToMount(share, &rmount)
		if *useNfs {
			rmount.Type = new(mount.MountType("nfs"))
			rmount.Path = new(share.MountPointData.Path)
		} else {
			rmount.Type = new(mount.MountType("cifs"))
			rmount.Username = mountUsername
			rmount.Password = &mountPassword
		}
		resp, err := self.mount_client.UpdateMountWithResponse(ctx, *rmount.Name, rmount)
		if err != nil {
			return errors.Errorf("Error updating mount %s from ha_supervisor: %w", *rmount.Name, err)
		}
		if resp.StatusCode() != 200 {
			// If we get a 400 error, it might be because the systemd unit is in a stale state
			// Try to remove it and recreate the mount (similar to create path)
			if resp.StatusCode() == 400 {
				// Attempt to remove the potentially stale mount
				removeResp, removeErr := self.mount_client.RemoveMountWithResponse(ctx, share.Name)
				if removeErr == nil && removeResp.StatusCode() == 200 {
					return self.networkMountShareWithRetry(ctx, share, retries-1)
				}
			}
			// Original error or retry strategy didn't work
			return errors.Errorf("Error updating mount %s from ha_supervisor: %d %#v", share.Name, resp.StatusCode(), string(resp.Body))
		}
	}
	return nil
}

func (self *SupervisorService) NetworkUnmountShare(ctx context.Context, shareName string) errors.E {
	if self.state.HACoreReady == false {
		return errors.Errorf("HA Core is not ready")
	}

	mounts, errE := self.NetworkGetAllMounted(ctx)
	if errE != nil {
		return errE
	}

	_, ok := mounts[shareName]
	if !ok {
		slog.InfoContext(ctx, "Share not mounted in ha_supervisor, skipping unmount", "share", shareName)
		// not mounted
		return nil
	}

	resp, err := self.mount_client.RemoveMountWithResponse(ctx, shareName)
	if err != nil {
		return errors.Errorf("Error unmounting share %s from ha_supervisor: %w", shareName, err)
	}
	if resp.StatusCode() != 200 {
		return errors.Errorf("Error unmounting share %s from ha_supervisor: %d %#v", shareName, resp.StatusCode(), resp)
	}
	return nil
}

func (self *SupervisorService) NetworkMountAllShares(ctx context.Context) errors.E {
	if self.state.HACoreReady == false {
		slog.InfoContext(ctx, "HA Core is not ready, skipping mountHaStorage")
		return nil
	}
	shares, err := self.share_service.ListShares()
	if err != nil {
		return errors.WithStack(err)
	}

	if self.state.AddonIpAddress != "" {
		for _, share := range shares {
			if share.Disabled != nil && *share.Disabled {
				continue
			}

			if !share.Status.IsValid {
				continue
			}
			switch share.Usage {
			case "media", "share", "backup":
				err = self.NetworkMountShare(ctx, share)
				if err != nil {
					slog.ErrorContext(ctx, "Mounting error", "share", share, "err", err)
				}
			}
		}
		// Unmount lost shares
		err = self.networkUnmountLostShares(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Error unmounting lost shares", "err", err)
		}
	} else {
		slog.WarnContext(ctx, "Addon IP address is empty, skipping mountHaStorage")
	}
	return nil
}

func (self *SupervisorService) networkUnmountLostShares(ctx context.Context) errors.E {
	shares, err := self.share_service.ListShares()
	if err != nil {
		return errors.WithStack(err)
	}

	if self.state.HACoreReady {
		mounts, err := self.NetworkGetAllMounted(ctx)
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
		slog.InfoContext(ctx, "Unmounting remaining HA mounts", "count", len(mounts))
		for name := range mounts {
			err := self.NetworkUnmountShare(ctx, name)
			if err != nil {
				slog.ErrorContext(ctx, "Unmounting error", "share", name, "err", err)
			}
		}
	}
	return nil
}

func (self *SupervisorService) NetworkUnmountAllShares(ctx context.Context) (err errors.E) {
	shares, err := self.share_service.ListShares()
	if err != nil {
		return errors.WithStack(err)
	}
	for _, share := range shares {
		if share.Disabled != nil && *share.Disabled {
			continue
		}
		switch share.Usage {
		case "media", "share", "backup":
			err = self.NetworkUnmountShare(ctx, share.Name)
			if err != nil {
				slog.ErrorContext(ctx, "Unmounting error", "share", share, "err", err)
			}
		}
	}
	return err
}
