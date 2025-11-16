package service

import (
	"context"
	"log/slog"
	"os"
	"sync"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/tlog"
	"github.com/xorcare/pointer"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type ShareServiceInterface interface {
	//	SaveAll(*[]dto.SharedResource) errors.E
	ListShares() ([]dto.SharedResource, errors.E)
	GetShare(name string) (*dto.SharedResource, errors.E)
	CreateShare(share dto.SharedResource) (*dto.SharedResource, errors.E)
	UpdateShare(name string, share dto.SharedResource) (*dto.SharedResource, errors.E)
	DeleteShare(name string) errors.E
	DisableShare(name string) (*dto.SharedResource, errors.E)
	EnableShare(name string) (*dto.SharedResource, errors.E)
	//GetShareFromPath(path string) (*dto.SharedResource, errors.E)
	SetShareFromPathEnabled(path string, enabled bool) (*dto.SharedResource, errors.E)
	//NotifyClient()
	VerifyShare(share *dto.SharedResource) errors.E
}

type ShareService struct {
	exported_share_repo repository.ExportedShareRepositoryInterface
	user_service        UserServiceInterface
	mount_repo          repository.MountPointPathRepositoryInterface
	eventBus            events.EventBusInterface
	sharesQueueMutex    *sync.RWMutex
	dbomConv            converter.DtoToDbomConverterImpl
	defaultConfig       *config.DefaultConfig
}

type ShareServiceParams struct {
	fx.In
	ExportedShareRepo repository.ExportedShareRepositoryInterface
	UserService       UserServiceInterface
	MountRepo         repository.MountPointPathRepositoryInterface
	EventBus          events.EventBusInterface
	DefaultConfig     *config.DefaultConfig
}

func NewShareService(lc fx.Lifecycle, in ShareServiceParams) ShareServiceInterface {
	s := &ShareService{
		exported_share_repo: in.ExportedShareRepo,
		user_service:        in.UserService,
		mount_repo:          in.MountRepo,
		eventBus:            in.EventBus,
		defaultConfig:       in.DefaultConfig,
		sharesQueueMutex:    &sync.RWMutex{},
		dbomConv:            converter.DtoToDbomConverterImpl{},
	}
	unsubscribe := s.eventBus.OnMountPoint(func(event events.MountPointEvent) {
		slog.Info("Received MountPointEvent", "type", event.Type, "mountpoint", event.MountPoint)
		_, errs := s.SetShareFromPathEnabled(event.MountPoint.Path, event.MountPoint.IsMounted)
		if errs != nil {
			if errors.Is(errs, gorm.ErrRecordNotFound) {
				tlog.Trace("No share found for mount point", "path", event.MountPoint.Path)
				return
			}
			slog.Error("Error updating share status from mount event", "err", errs)
		}
	})
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			if os.Getenv("SRAT_MOCK") == "true" {
				return nil
			}
			allusers, err := s.user_service.ListUsers()
			if err != nil {
				return errors.WithStack(err)
			}
			// Create all default Shares if don't exists
			cconv := converter.ConfigToDtoConverterImpl{}
			for _, defCShare := range s.defaultConfig.Shares {
				_, err := s.GetShare(defCShare.Name)
				if err != nil {
					if errors.Is(err, dto.ErrorShareNotFound) {
						defShare, errConv := cconv.ShareToSharedResource(defCShare, allusers)
						if errConv != nil {
							slog.Error("Error converting default share", "name", defCShare.Name, "err", errConv)
							return errors.WithStack(errConv)
						}
						slog.Info("Creating default share", "name", defShare.Name, "path", defShare.MountPointData.Path, "device_id", defShare.MountPointData.DeviceId)
						_, createErr := s.CreateShare(defShare)
						if createErr != nil {
							slog.Error("Error creating default share", "name", defShare.Name, "err", createErr)
							return createErr
						}
					} else {
						slog.Error("Error checking for default share", "name", defCShare.Name, "err", err)
					}
				}
			}
			return nil
		},
		OnStop: func(_ context.Context) error {
			if unsubscribe != nil {
				unsubscribe()
			}
			return nil
		},
	})
	return s
}

/*
func (s *ShareService) SaveAll(shares *[]dto.SharedResource) errors.E {
	sh, err := s.dbomConv.SharedResourcesToExportedShares(shares)
	if err != nil {
		return errors.WithStack(err)
	}
	return s.exported_share_repo.SaveAll(sh)
}
*/

func (s *ShareService) ListShares() ([]dto.SharedResource, errors.E) {
	shares, err := s.exported_share_repo.All()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list shares")
	}
	var conv converter.DtoToDbomConverterImpl
	var dtoShares []dto.SharedResource
	for _, share := range *shares {
		var dtoShare dto.SharedResource
		err := conv.ExportedShareToSharedResource(share, &dtoShare, nil)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert share")
		}

		// Verify share validity
		err = s.VerifyShare(&dtoShare)
		if err != nil {
			slog.Error("Error verifying share", "share", dtoShare.Name, "err", err)
			continue
		}

		dtoShares = append(dtoShares, dtoShare)
	}
	return dtoShares, nil
}

func (s *ShareService) GetShare(name string) (*dto.SharedResource, errors.E) {
	share, err := s.exported_share_repo.FindByName(name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get share")
	}
	if share == nil {
		return nil, errors.WithStack(dto.ErrorShareNotFound)
	}
	var conv converter.DtoToDbomConverterImpl
	var dtoShare dto.SharedResource
	err = conv.ExportedShareToSharedResource(*share, &dtoShare, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert share")
	}

	if err := s.VerifyShare(&dtoShare); err != nil {
		slog.Warn("Share verification failed", "share", dtoShare.Name, "err", err)
	}

	return &dtoShare, nil
}

func (s *ShareService) CreateShare(share dto.SharedResource) (*dto.SharedResource, errors.E) {
	existing, err := s.exported_share_repo.FindByName(share.Name)
	if err != nil && (!errors.Is(err, gorm.ErrRecordNotFound) && !errors.Is(err, dto.ErrorShareNotFound)) {
		slog.Error("Failed to check for existing share", "share_name", share.Name, "error", err)
		return nil, errors.Wrap(err, "failed to check for existing share")
	}
	if existing != nil {
		return nil, errors.WithStack(dto.ErrorShareAlreadyExists)
	}

	var conv converter.DtoToDbomConverterImpl
	var dbShare dbom.ExportedShare
	err = conv.SharedResourceToExportedShare(share, &dbShare)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert share")
	}

	if len(dbShare.Users) == 0 {
		admin, err := s.user_service.GetAdmin()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get admin user")
		}
		var dbomAdmin dbom.SambaUser
		err = s.dbomConv.UserToSambaUser(*admin, &dbomAdmin)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert admin user to dbom.SambaUser")
		}
		dbShare.Users = []dbom.SambaUser{dbomAdmin}
	}

	errS := s.mount_repo.Save(&dbShare.MountPointData)
	if errS != nil {
		return nil, errors.Wrap(errS, "failed to save mount point")
	}

	err = s.exported_share_repo.Save(&dbShare)
	if err != nil {
		return nil, errors.Wrap(err, "failed to save share")
	}

	var dtoShare dto.SharedResource
	var convOut converter.DtoToDbomConverterImpl
	err = convOut.ExportedShareToSharedResource(dbShare, &dtoShare, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert share")
	}

	if err := s.VerifyShare(&dtoShare); err != nil {
		slog.Warn("Share verification failed", "share", dtoShare.Name, "err", err)
	}

	s.eventBus.EmitShare(events.ShareEvent{
		Event: events.Event{Type: events.EventTypes.ADD},
		Share: &dtoShare,
	})

	return &dtoShare, nil
}

func (s *ShareService) UpdateShare(name string, share dto.SharedResource) (*dto.SharedResource, errors.E) {
	dbShare, err := s.exported_share_repo.FindByName(name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get share")
	}
	if dbShare == nil {
		return nil, errors.WithStack(dto.ErrorShareNotFound)
	}

	var conv converter.DtoToDbomConverterImpl
	err = conv.SharedResourceToExportedShare(share, dbShare)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert share")
	}

	if len(dbShare.Users) == 0 {
		adminUser, adminErr := s.user_service.GetAdmin()
		if adminErr != nil {
			return nil, errors.Wrap(adminErr, "failed to get admin user for new share")
		}
		var dbomAdmin dbom.SambaUser
		errS := s.dbomConv.UserToSambaUser(*adminUser, &dbomAdmin)
		if errS != nil {
			return nil, errors.Wrap(errS, "failed to convert admin user to dbom.SambaUser")
		}
		dbShare.Users = append(dbShare.Users, dbomAdmin)
	}

	if err := s.exported_share_repo.Save(dbShare); err != nil {
		// Note: gorm.ErrDuplicatedKey might not be standard across all GORM dialects/drivers.
		// Checking for a more generic "constraint violation" or relying on the FindByName check might be more robust.
		return nil, errors.Wrapf(err, "failed to save share '%s' to repository", share.Name)
	}

	var createdDtoShare dto.SharedResource
	if err := conv.ExportedShareToSharedResource(*dbShare, &createdDtoShare, nil); err != nil {
		return nil, errors.Wrapf(err, "failed to convert created dbom.ExportedShare back to dto.SharedResource for share '%s'", dbShare.Name)
	}

	if err := s.VerifyShare(&createdDtoShare); err != nil {
		slog.Warn("New share verification failed", "share", createdDtoShare.Name, "err", err)
	}

	err = s.mount_repo.Save(&dbShare.MountPointData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to save mount point")
	}

	err = s.exported_share_repo.Save(dbShare)
	if err != nil {
		return nil, errors.Wrap(err, "failed to save share")
	}

	var dtoShare dto.SharedResource
	var convOut converter.DtoToDbomConverterImpl
	err = convOut.ExportedShareToSharedResource(*dbShare, &dtoShare, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert share")
	}

	s.eventBus.EmitShare(events.ShareEvent{
		Event: events.Event{Type: events.EventTypes.UPDATE},
		Share: &dtoShare,
	})

	return &dtoShare, nil
}

// DeleteShare deletes a shared resource by its name.
func (s *ShareService) DeleteShare(name string) errors.E {
	var ashare *dto.SharedResource
	ashare, err := s.GetShare(name)
	if err != nil { // Leverage GetShare for not-found check
		return err
	}
	s.eventBus.EmitShare(events.ShareEvent{
		Event: events.Event{Type: events.EventTypes.REMOVE},
		Share: ashare,
	})
	err = s.exported_share_repo.Delete(name)
	if err != nil {
		return errors.Wrap(err, "failed to delete share")
	}
	err = s.mount_repo.Delete(ashare.MountPointData.Path)
	if err != nil {
		return errors.Wrap(err, "failed to delete mount point")
	}
	return nil
}

/*
func (s *ShareService) GetShareFromPath(path string) (*dto.SharedResource, errors.E) {
	share, err := s.exported_share_repo.FindByMountPath(path)
	if err != nil {
		return nil, err // This will propagate ErrorShareNotFound
	}

	var conv converter.DtoToDbomConverterImpl
	var dtoShare dto.SharedResource
	err = conv.ExportedShareToSharedResource(*share, &dtoShare, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert share")
	}
	return &dtoShare, nil
}
*/

func (s *ShareService) SetShareFromPathEnabled(path string, enabled bool) (*dto.SharedResource, errors.E) {
	share, err := s.exported_share_repo.FindByMountPath(path)
	if err != nil {
		return nil, err // This will propagate ErrorShareNotFound
	}
	if share.Disabled != nil && *share.Disabled == !enabled {
		// No change needed
		var conv converter.DtoToDbomConverterImpl
		var dtoShare dto.SharedResource
		err = conv.ExportedShareToSharedResource(*share, &dtoShare, nil)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert share")
		}
		return &dtoShare, nil
	}

	disabled := !enabled
	share.Disabled = &disabled
	err = s.exported_share_repo.Save(share)
	if err != nil {
		return nil, errors.Wrap(err, "failed to save share")
	}

	var conv converter.DtoToDbomConverterImpl
	var dtoShare dto.SharedResource
	err = conv.ExportedShareToSharedResource(*share, &dtoShare, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert share")
	}
	s.eventBus.EmitShare(events.ShareEvent{
		Event: events.Event{Type: events.EventTypes.UPDATE},
		Share: &dtoShare,
	})

	return &dtoShare, nil
}

func (s *ShareService) setShareEnabled(name string, enabled bool) (*dto.SharedResource, errors.E) {
	share, err := s.exported_share_repo.FindByName(name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get share")
	}
	if share == nil {
		return nil, errors.WithStack(dto.ErrorShareNotFound)
	}
	disabled := !enabled
	share.Disabled = &disabled
	err = s.exported_share_repo.Save(share)
	if err != nil {
		return nil, errors.Wrap(err, "failed to save share")
	}
	var conv converter.DtoToDbomConverterImpl
	var dtoShare dto.SharedResource
	err = conv.ExportedShareToSharedResource(*share, &dtoShare, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert share")
	}
	s.eventBus.EmitShare(events.ShareEvent{
		Event: events.Event{Type: events.EventTypes.UPDATE},
		Share: &dtoShare,
	})
	return &dtoShare, nil
}

func (s *ShareService) DisableShare(name string) (*dto.SharedResource, errors.E) {
	return s.setShareEnabled(name, false)
}

func (s *ShareService) EnableShare(name string) (*dto.SharedResource, errors.E) {
	return s.setShareEnabled(name, true)
}

/*
func (s *ShareService) NotifyClient() {
	s.sharesQueueMutex.RLock()
	defer s.sharesQueueMutex.RUnlock()

	shares, err := s.ListShares()
	if err != nil {
		log.Printf("Error listing shares in notifyClient: %v", err)
		return
	}
	s.broadcaster.BroadcastMessage(shares)
}
*/

// VerifyShare checks the validity of a share and disables it if invalid
// It handles the following scenarios:
// 1. Volume mounted and RW -> share active or not active as DB value and RW
// 2. Volume mounted and RO -> share active or not active as DB value and RO. No RW users
// 3. Volume not mounted -> share is not active and marked as anomaly (Invalid=true)
// 4. Volume not exists -> share is not active and marked as anomaly (Invalid=true)
func (s *ShareService) VerifyShare(share *dto.SharedResource) errors.E {
	if share == nil {
		return errors.New("share cannot be nil")
	}

	// Case 4: Check if MountPointData exists and has a valid path
	if share.MountPointData == nil || share.MountPointData.Path == "" {
		slog.Warn("Share has no valid MountPointData", "share", share.Name)
		share.Invalid = pointer.Bool(true)
		share.Disabled = pointer.Bool(true)
		return nil
	}

	// Case 4: Volume doesn't exist (marked as invalid in mount point)
	if share.MountPointData.IsInvalid {
		slog.Warn("Share volume does not exist",
			"share", share.Name,
			"path", share.MountPointData.Path)
		share.Invalid = pointer.Bool(true)
		share.Disabled = pointer.Bool(true)
		return nil
	}

	// Case 3: Volume exists but is not mounted
	if !share.MountPointData.IsMounted {
		slog.Warn("Share volume is not mounted",
			"share", share.Name,
			"path", share.MountPointData.Path)
		share.Invalid = pointer.Bool(true)
		share.Disabled = pointer.Bool(true)
		return nil
	}

	// Cases 1 & 2: Volume is mounted - validate write support vs user permissions
	if share.MountPointData.IsWriteSupported != nil {
		if !*share.MountPointData.IsWriteSupported {
			// Case 2: Read-only volume - ensure no RW users
			for i := range share.Users {
				if len(share.Users[i].RwShares) > 0 {
					// Filter out this share from RW shares
					var newRwShares []string
					for _, rwShare := range share.Users[i].RwShares {
						if rwShare != share.Name {
							newRwShares = append(newRwShares, rwShare)
						}
					}
					share.Users[i].RwShares = newRwShares
					slog.Info("Removed RW permission from user on RO volume",
						"share", share.Name,
						"user", share.Users[i].Username,
						"path", share.MountPointData.Path)
				}
			}
		}
		// Case 1: RW volume - share state depends on DB disabled value (already set)
	}

	// Check if mount is active in Home Assistant (for non-internal shares)
	if share.Usage != "internal" && share.Usage != "none" {
		if share.IsHAMounted != nil && !*share.IsHAMounted {
			slog.Warn("Share mount point is not mounted in Home Assistant",
				"share", share.Name,
				"status", share.HaStatus)
			share.Invalid = pointer.Bool(true)
			share.Disabled = pointer.Bool(true)
			return nil
		}
	}

	// Share is valid
	if share.Invalid == nil || !*share.Invalid {
		share.Invalid = pointer.Bool(false)
	}

	return nil
}
