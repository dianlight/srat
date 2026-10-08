package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"gorm.io/gorm"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/homeassistant/mount"
	"github.com/dianlight/tlog"
)

var _supervisor_api_mutex sync.Mutex

// supervisorMountTargetNotEmptyErrorKey is the supervisor error_key returned
// when a mount target already holds leftover data (issue #1358).
const supervisorMountTargetNotEmptyErrorKey = "mount_target_not_empty_error"

var supervisorMountTargetPathRe = regexp.MustCompile(`/data/\S+`)

// supervisorMountTargetNotEmptyProblemKey returns the stable per-share problem
// key for the leftover-data-at-target condition so ignores and dismissals stay
// scoped to the affected share.
func supervisorMountTargetNotEmptyProblemKey(mountName string) string {
	return "supervisor_mount_target_not_empty_" + mountName
}

// parseSupervisorMountTargetNotEmpty inspects a supervisor error body for the
// mount_target_not_empty_error key. It returns the leftover-data target path
// extracted from the message, or an empty path with true when the key matches
// but no path could be parsed.
func parseSupervisorMountTargetNotEmpty(body []byte) (string, bool) {
	var payload struct {
		ErrorKey    string `json:"error_key"`
		ErrorKeyAlt string `json:"errorKey"`
		Message     string `json:"message"`
		MessageAlt  string `json:"msg"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", false
	}
	key := payload.ErrorKey
	if key == "" {
		key = payload.ErrorKeyAlt
	}
	if key != supervisorMountTargetNotEmptyErrorKey {
		return "", false
	}
	message := payload.Message
	if message == "" {
		message = payload.MessageAlt
	}
	path := supervisorMountTargetPathRe.FindString(message)
	path = strings.TrimRight(path, ".,;:\"'")
	return path, true
}

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
	problemService     ProblemServiceInterface
	eventBus           events.EventBusInterface
	disks              *dto.DiskMap
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
	ProblemService   ProblemServiceInterface `optional:"true"`
	EventBus         events.EventBusInterface
	Disks            *dto.DiskMap `optional:"true"`
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
	p.problemService = in.ProblemService
	p.eventBus = in.EventBus
	p.disks = in.Disks
	unsubscribe := make([]func(), 3)
	unsubscribe[0] = p.eventBus.OnServerProccess(func(ctx context.Context, event events.ServerProcessEvent) errors.E {
		slog.DebugContext(ctx, "SupervisorService received ServerProcess event", "tracker", event.DataDirtyTracker)
		if event.Type == events.EventTypes.CLEAN {
			err := p.NetworkMountAllShares(ctx)
			if err != nil {
				slog.ErrorContext(ctx, "Error mounting HA storage shares", "err", err)
				p.eventBus.EmitHomeAssistant(events.HomeAssistantEvent{
					Type: events.EventTypes.ERROR,
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
					Type: events.EventTypes.ERROR,
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
					Type: events.EventTypes.ERROR,
					Error: &dto.ErrorDataModel{
						Title:  "Error unmounting share from ha_supervisor",
						Detail: err.Error(),
					},
				})
				return err
			}
		} else if event.Type == events.EventTypes.UPDATE &&
			(event.Share.Disabled != nil && *event.Share.Disabled) {
			err := p.NetworkUnmountShare(ctx, event.Share.Name)
			if err != nil {
				slog.ErrorContext(ctx, "Error unmounting share from ha_supervisor", "share", event.Share.Name, "err", err)
				p.eventBus.EmitHomeAssistant(events.HomeAssistantEvent{
					Type: events.EventTypes.ERROR,
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
					Type: events.EventTypes.ERROR,
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
			if err := p.NetworkUnmountAllShares(ctx); err != nil {
				slog.WarnContext(ctx, "Error while unmounting shares on supervisor stop", "error", err)
			}
			return nil
		},
	})
	return p
}

func (self *SupervisorService) NetworkGetAllMounted(ctx context.Context) (mounts map[string]mount.Mount, err errors.E) {
	_supervisor_api_mutex.Lock()
	defer _supervisor_api_mutex.Unlock()
	if !self.state.HACoreReady {
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

// sanitizeSupervisorMountName maps SRAT share names to HA Supervisor mount names.
// Supervisor validates mount name against ^[A-Za-z0-9_]+$ (issue #1299) while SRAT
// allows hyphens. Only Name is sanitized; Share keeps the raw SMB name.
func sanitizeSupervisorMountName(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}

func (self *SupervisorService) networkMountShareWithRetry(ctx context.Context, share dto.SharedResource, retries int) errors.E {

	if retries <= 0 {
		return errors.Errorf("Exceeded maximum retries to mount share %s", share.Name)
	}

	if !self.state.HACoreReady {
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

	rmount, ok := mounts[sanitizeSupervisorMountName(share.Name)]
	enrichSharePartitionFromCache(&share, self.disks)
	if !ok {
		// new mount
		rmount = mount.Mount{}
		if err := conv.SharedResourceToMount(share, &rmount); err != nil {
			return errors.Wrap(err, "failed converting share to HA mount payload")
		}
		sanitized := sanitizeSupervisorMountName(share.Name)
		rmount.Name = &sanitized
		rmount.Share = &share.Name
		rmount.Server = &self.state.AddonIpAddress

		if *useNfs && isShareNFSExportable(ctx, share) {
			nfsPath := resolveActualMountPointPath(share)
			if nfsPath == "" {
				nfsPath = share.Name
			}
			// strip /mnt prefix if exists to avoid issues with ha_supervisor path handling
			nfsPath = strings.TrimPrefix(nfsPath, "/mnt/")
			rmount.Type = new(mount.MountType("nfs"))
			rmount.Path = &nfsPath
			rmount.Username = nil
			rmount.Password = nil
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
				// Leftover data at the target path (issue #1358) is never
				// touched: warn, raise a notice naming the path and stop
				// without an error so RESTARTs do not spam Sentry. The full
				// mount request (with password) is deliberately not logged.
				if targetPath, ok := parseSupervisorMountTargetNotEmpty(resp.Body); ok {
					self.handleSupervisorMountTargetNotEmpty(ctx, *rmount.Name, share.Name, targetPath)
					return nil
				}
				// Attempt to remove the potentially stale mount
				removeResp, removeErr := self.mount_client.RemoveMountWithResponse(ctx, sanitizeSupervisorMountName(share.Name))
				if removeErr == nil && removeResp.StatusCode() == 200 {
					// Successfully removed, retry creation
					retryResp, retryErr := self.mount_client.CreateMountWithResponse(ctx, rmount)
					if retryErr != nil {
						return errors.Errorf("Error creating mount %s from ha_supervisor after retry: %w", share.Name, retryErr)
					}
					if retryResp.StatusCode() == 200 {
						// Success on retry
						self.dismissSupervisorMountTargetNotEmpty(ctx, *rmount.Name)
						return nil
					}
					if targetPath, ok := parseSupervisorMountTargetNotEmpty(retryResp.Body); ok {
						self.handleSupervisorMountTargetNotEmpty(ctx, *rmount.Name, share.Name, targetPath)
						return nil
					}
					// Retry also failed
					rjson, _ := json.Marshal(rmount)
					return errors.Errorf("Error creating mount %s from ha_supervisor after removing stale mount: %d \nReq:%#v\nResp:%#v", *rmount.Name, retryResp.StatusCode(), string(rjson), string(retryResp.Body))
				}
			}
			// Original error or retry strategy didn't work
			rjson, _ := json.Marshal(rmount)
			return errors.Errorf("Error creating mount %s from ha_supervisor: %d \nReq:%#v\nResp:%#v", *rmount.Name, resp.StatusCode(), string(rjson), string(resp.Body))
		}
		self.dismissSupervisorMountTargetNotEmpty(ctx, *rmount.Name)
		return nil
	} else if string(share.Usage) != string(*rmount.Usage) ||
		*rmount.State != "active" ||
		(*useNfs && *rmount.Type == "cifs") ||
		(!*useNfs && *rmount.Type == "nfs") {
		if err := conv.SharedResourceToMount(share, &rmount); err != nil {
			return errors.Wrap(err, "failed converting share to HA mount update payload")
		}
		mountName := sanitizeSupervisorMountName(share.Name)
		rmount.Name = &mountName
		rmount.Share = &share.Name
		if *useNfs && isShareNFSExportable(ctx, share) {
			nfsPath := resolveActualMountPointPath(share)
			if nfsPath == "" {
				nfsPath = share.Name
			}
			rmount.Type = new(mount.MountType("nfs"))
			rmount.Path = &nfsPath
			rmount.Username = nil
			rmount.Password = nil
		} else {
			rmount.Type = new(mount.MountType("cifs"))
			rmount.Username = mountUsername
			rmount.Password = &mountPassword
		}
		resp, err := self.mount_client.UpdateMountWithResponse(ctx, mountName, rmount)
		if err != nil {
			return errors.Errorf("Error updating mount %s from ha_supervisor: %w", mountName, err)
		}
		if resp.StatusCode() != 200 {
			// If we get a 400 error, it might be because the systemd unit is in a stale state
			// Try to remove it and recreate the mount (similar to create path)
			if resp.StatusCode() == 400 {
				// Same leftover-data policy as the create path (issue #1358):
				// never touch the data, warn plus notice, no error, no
				// password-bearing request logging.
				if targetPath, ok := parseSupervisorMountTargetNotEmpty(resp.Body); ok {
					self.handleSupervisorMountTargetNotEmpty(ctx, mountName, share.Name, targetPath)
					return nil
				}
				// Attempt to remove the potentially stale mount
				removeResp, removeErr := self.mount_client.RemoveMountWithResponse(ctx, mountName)
				if removeErr == nil && removeResp.StatusCode() == 200 {
					return self.networkMountShareWithRetry(ctx, share, retries-1)
				}
			}
			// Original error or retry strategy didn't work
			return errors.Errorf("Error updating mount %s from ha_supervisor: %d %#v", mountName, resp.StatusCode(), string(resp.Body))
		}
		self.dismissSupervisorMountTargetNotEmpty(ctx, mountName)
	}
	self.dismissSupervisorMountTargetNotEmpty(ctx, sanitizeSupervisorMountName(share.Name))
	return nil
}

// handleSupervisorMountTargetNotEmpty downgrades the leftover-data-at-target
// condition (issue #1358) from a per-restart error to a warning plus a
// ProblemService notice naming the exact path. It never moves, removes or
// otherwise touches the leftover data, never logs the mount request with its
// password, and returns nothing so callers surface no error (Warn-level logs
// do not reach Sentry, unlike Error-level ones).
func (self *SupervisorService) handleSupervisorMountTargetNotEmpty(ctx context.Context, mountName, shareName, targetPath string) {
	if targetPath == "" {
		targetPath = "/data/share/" + mountName
	}
	slog.WarnContext(ctx, "Supervisor mount blocked by leftover data at target; manual move required",
		"share", shareName,
		"mount", mountName,
		"target_path", targetPath,
		"error_key", supervisorMountTargetNotEmptyErrorKey)
	if self.problemService == nil {
		return
	}
	problemKey := supervisorMountTargetNotEmptyProblemKey(mountName)
	if existing, err := self.problemService.Get(problemKey); err == nil && existing != nil && existing.Ignored {
		// Permanent ignore: never raise again until dismissed/re-enabled.
		return
	}
	lastError := supervisorMountTargetNotEmptyErrorKey + ": leftover data at " + targetPath
	_, err := self.problemService.Upsert(&dto.Problem{
		ProblemKey: problemKey,
		Title:      "Supervisor mount blocked by leftover data",
		Description: fmt.Sprintf("Cannot mount share %q because leftover data already exists at %s. "+
			"Move it away first, then retry (restart the addon or re-save the share). "+
			"SRAT will never move or delete this data automatically.", shareName, targetPath),
		Severity:                dto.ProblemSeverities.PROBLEMSEVERITYWARNING,
		Status:                  dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSCREATED,
		TranslationKey:          problemKey,
		TranslationPlaceholders: map[string]string{"share": shareName, "target_path": targetPath},
		Data: map[string]any{
			"share":       shareName,
			"mount_name":  mountName,
			"target_path": targetPath,
			"error_key":   supervisorMountTargetNotEmptyErrorKey,
		},
		LastError:    &lastError,
		IsFixable:    false,
		IsPersistent: true,
	})
	if err != nil {
		slog.DebugContext(ctx, "Could not upsert mount-target problem", "problem_key", problemKey, "error", err)
	}
}

// dismissSupervisorMountTargetNotEmpty clears the leftover-data notice
// best-effort once a mount succeeds again. Missing rows are expected after
// manual dismissal and are ignored.
func (self *SupervisorService) dismissSupervisorMountTargetNotEmpty(ctx context.Context, mountName string) {
	if self.problemService == nil {
		return
	}
	problemKey := supervisorMountTargetNotEmptyProblemKey(mountName)
	if err := self.problemService.Dismiss(problemKey); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		slog.DebugContext(ctx, "Could not dismiss stale mount-target problem", "problem_key", problemKey, "error", err)
	}
}

func (self *SupervisorService) NetworkUnmountShare(ctx context.Context, shareName string) errors.E {
	if !self.state.HACoreReady {
		return errors.Errorf("HA Core is not ready")
	}

	mounts, errE := self.NetworkGetAllMounted(ctx)
	if errE != nil {
		return errE
	}

	name := sanitizeSupervisorMountName(shareName)
	if _, ok := mounts[name]; ok {
		// sanitized mount exists
	} else if _, ok := mounts[shareName]; ok {
		// legacy hyphenated mount exists (pre-#1299); unmount raw name
		name = shareName
	} else {
		slog.InfoContext(ctx, "Share not mounted in ha_supervisor, skipping unmount", "share", shareName)
		// not mounted
		return nil
	}

	resp, err := self.mount_client.RemoveMountWithResponse(ctx, name)
	if err != nil {
		return errors.Errorf("Error unmounting share %s from ha_supervisor: %w", shareName, err)
	}
	if resp.StatusCode() != 200 {
		return errors.Errorf("Error unmounting share %s from ha_supervisor: %d %#v", shareName, resp.StatusCode(), resp)
	}
	return nil
}

func (self *SupervisorService) NetworkMountAllShares(ctx context.Context) errors.E {
	if !self.state.HACoreReady {
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
					slog.ErrorContext(ctx, "Mounting error", "share", share.Name, "err", err)
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
				delete(mounts, sanitizeSupervisorMountName(share.Name))
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
				slog.ErrorContext(ctx, "Unmounting error", "share", share.Name, "err", err)
			}
		}
	}
	return err
}
