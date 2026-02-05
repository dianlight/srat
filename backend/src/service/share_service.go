package service

import (
	"context"
	"log/slog"
	"os"
	"sync"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dbom/g"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/tlog"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

var internalShares = []dto.SharedResource{
	{
		Name: "config",
		//MountPointData: &dto.MountPointData{},
		Usage: dto.UsageAsInternal,
	},
	{
		Name: "addons",
		//MountPointData: &dto.MountPointData{},
		Usage: dto.UsageAsInternal,
	},
	{
		Name:  "ssl",
		Usage: dto.UsageAsInternal,
	},
	{
		Name:  "share",
		Usage: dto.UsageAsInternal,
	},
	{
		Name:  "backup",
		Usage: dto.UsageAsInternal,
	},
	{
		Name:  "media",
		Usage: dto.UsageAsInternal,
	},
	{
		Name:  "addon_configs",
		Usage: dto.UsageAsInternal,
	},
}

/*
ShareServiceInterface defines the interface for managing shared resources.

Copilot file rules:
- Always preload related data when fetching shares.

*/

type ShareServiceInterface interface {
	//	SaveAll(*[]dto.SharedResource) errors.E
	ListShares() ([]dto.SharedResource, errors.E)
	GetShare(name string) (*dto.SharedResource, errors.E)
	CreateShare(share dto.SharedResource) (*dto.SharedResource, errors.E)
	UpdateShare(name string, share dto.SharedResource) (*dto.SharedResource, errors.E)
	DeleteShare(name string) errors.E
	DisableShare(name string) (*dto.SharedResource, errors.E)
	EnableShare(name string) (*dto.SharedResource, errors.E)
	GetShareFromPath(path string) (*dto.SharedResource, errors.E)
	SetShareFromPathEnabled(path string, enabled bool) (*dto.SharedResource, errors.E)
	//NotifyClient()
	VerifyShare(share *dto.SharedResource) errors.E
	SetSupervisorService(s SupervisorServiceInterface)
}

type ShareService struct {
	ctx                context.Context
	db                 *gorm.DB
	supervisor_service SupervisorServiceInterface
	//exported_share_repo repository.ExportedShareRepositoryInterface
	user_service UserServiceInterface
	//mount_repo          repository.MountPointPathRepositoryInterface
	eventBus         events.EventBusInterface
	sharesQueueMutex *sync.RWMutex
	dbomConv         converter.DtoToDbomConverterImpl
	//defaultConfig    *config.DefaultConfig
}

type ShareServiceParams struct {
	fx.In
	Ctx context.Context
	Db  *gorm.DB
	//	SupervisorService SupervisorServiceInterface `optional:"true"`
	//ExportedShareRepo repository.ExportedShareRepositoryInterface
	UserService UserServiceInterface
	//MountRepo         repository.MountPointPathRepositoryInterface
	EventBus events.EventBusInterface
	//DefaultConfig *config.DefaultConfig
}

func NewShareService(lc fx.Lifecycle, in ShareServiceParams) ShareServiceInterface {
	s := &ShareService{
		//exported_share_repo: in.ExportedShareRepo,
		user_service: in.UserService,
		//mount_repo:          in.MountRepo,
		//		supervisor_service: in.SupervisorService,
		ctx:      in.Ctx,
		db:       in.Db,
		eventBus: in.EventBus,
		//defaultConfig:    in.DefaultConfig,
		sharesQueueMutex: &sync.RWMutex{},
		dbomConv:         converter.DtoToDbomConverterImpl{},
	}
	unsubscribe := s.eventBus.OnMountPoint(func(ctx context.Context, event events.MountPointEvent) errors.E {
		slog.InfoContext(ctx, "Received MountPointEvent", "type", event.Type, "mountpoint", event.MountPoint)
		share, err := s.GetShareFromPath(event.MountPoint.Path)
		if err != nil {
			if errors.Is(err, dto.ErrorShareNotFound) {
				tlog.TraceContext(ctx, "No share found for mount point", "path", event.MountPoint.Path)
				return nil
			}
			return err
		}
		if share == nil {
			tlog.TraceContext(ctx, "No share found for mount point", "path", event.MountPoint.Path)
			return nil
		}
		s.eventBus.EmitShare(events.ShareEvent{
			Event: events.Event{Type: events.EventTypes.UPDATE},
			Share: share, // Let subscribers fetch the share if needed
		})
		return nil
	})

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			if os.Getenv("SRAT_MOCK") == "true" {
				return nil
			}
			admin, err := s.user_service.GetAdmin()
			if err != nil {
				return errors.WithStack(err)
			}
			// Create all default Shares if don't exists
			//cconv := converter.ConfigToDtoConverterImpl{}
			for _, defCShare := range internalShares {
				_, err := s.GetShare(defCShare.Name)
				if err != nil {
					if errors.Is(err, dto.ErrorShareNotFound) {
						// load and associate admin user.
						defCShare.Users = []dto.User{*admin}

						slog.Debug("Creating default share", "name", defCShare.Name)

						_, createErr := s.CreateShare(defCShare)
						if createErr != nil {
							slog.Error("Error creating default share", "name", defCShare.Name, "err", createErr)
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

func (s *ShareService) ListShares() ([]dto.SharedResource, errors.E) {
	shares, err := gorm.G[dbom.ExportedShare](s.db).
		Preload("MountPointData", nil).
		Preload("Users", nil).
		Preload("RoUsers", nil).
		Find(s.ctx)
	if err != nil {
		return nil, errors.Errorf("failed to list shares from repository: %w", err)
	}
	var conv converter.DtoToDbomConverterImpl
	var dtoShares []dto.SharedResource
	for _, share := range shares {
		dtoShare, err := conv.ExportedShareToSharedResource(share)
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
	share, err := gorm.G[dbom.ExportedShare](s.db).
		Preload("MountPointData", nil).
		Preload("Users", nil).
		Preload("RoUsers", nil).
		Where(g.ExportedShare.Name.Eq(name)).First(s.ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.WithStack(dto.ErrorShareNotFound)
		}
		return nil, errors.Wrap(err, "failed to get share")
	}
	var conv converter.DtoToDbomConverterImpl
	dtoShare, errS := conv.ExportedShareToSharedResource(share)

	if errS != nil {
		return nil, errors.Wrap(errS, "failed to convert share")
	}

	if err := s.VerifyShare(&dtoShare); err != nil {
		slog.Warn("Share verification failed", "share", dtoShare.Name, "err", err)
	}

	return &dtoShare, nil
}

func (s *ShareService) CreateShare(share dto.SharedResource) (*dto.SharedResource, errors.E) {
	check, err := gorm.G[dbom.ExportedShare](s.db).Scopes(dbom.IncludeSoftDeleted).Where("name = ? and deleted_at IS NOT NULL", share.Name).Update(s.ctx, "deleted_at", nil)
	if err != nil {
		slog.Error("Failed to check for existing share", "share_name", share.Name, "error", err)
		return nil, errors.Wrapf(err, "failed to check for existing share: %s", err.Error())
	} else if check > 0 {
		return s.UpdateShare(share.Name, share)
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

	err = gorm.G[dbom.ExportedShare](s.db).Create(s.ctx, &dbShare)
	if err != nil {
		return nil, errors.Errorf("failed to save share '%s' to repository: %w", share.Name, err)
	}

	var convOut converter.DtoToDbomConverterImpl
	dtoShare, errS := convOut.ExportedShareToSharedResource(dbShare)
	if errS != nil {
		return nil, errors.Wrap(errS, "failed to convert share")
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
	dbShare, err := gorm.G[dbom.ExportedShare](s.db).
		Preload("MountPointData", nil).
		Preload("Users", nil).
		Preload("RoUsers", nil).
		Where(g.ExportedShare.Name.Eq(name)).First(s.ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.WithStack(dto.ErrorShareNotFound)
		}
		return nil, errors.Wrap(err, "failed to get share")
	}

	// Clear associations before updating to avoid foreign key constraint violations
	if err := s.db.WithContext(s.ctx).Model(&dbShare).Association("Users").Clear(); err != nil {
		return nil, errors.Wrap(err, "failed to clear Users associations during update")
	}
	if err := s.db.WithContext(s.ctx).Model(&dbShare).Association("RoUsers").Clear(); err != nil {
		return nil, errors.Wrap(err, "failed to clear RoUsers associations during update")
	}

	var conv converter.DtoToDbomConverterImpl
	err = conv.SharedResourceToExportedShare(share, &dbShare)
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

	if _, err := gorm.G[dbom.ExportedShare](s.db).Updates(s.ctx, dbShare); err != nil {
		// Note: gorm.ErrDuplicatedKey might not be standard across all GORM dialects/drivers.
		// Checking for a more generic "constraint violation" or relying on the FindByName check might be more robust.
		return nil, errors.Wrapf(err, "failed to update share '%s' err %v", share.Name, err)
	}

	createdDtoShare, errS := conv.ExportedShareToSharedResource(dbShare)
	if errS != nil {
		return nil, errors.Wrapf(errS, "failed to convert created dbom.ExportedShare back to dto.SharedResource for share '%s'", dbShare.Name)
	}

	if err := s.VerifyShare(&createdDtoShare); err != nil {
		slog.Warn("New share verification failed", "share", createdDtoShare.Name, "err", err)
	}

	s.eventBus.EmitShare(events.ShareEvent{
		Event: events.Event{Type: events.EventTypes.UPDATE},
		Share: &createdDtoShare,
	})

	return &createdDtoShare, nil
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

	// Retrieve the share with associations to clear them before soft delete
	var dbShare dbom.ExportedShare
	if err := s.db.WithContext(s.ctx).
		Preload("Users").
		Preload("RoUsers").
		Where("name = ?", name).First(&dbShare).Error; err != nil {
		return errors.Wrap(err, "failed to retrieve share for deletion")
	}

	// Clear associations to avoid foreign key issues on recreation
	if err := s.db.WithContext(s.ctx).Model(&dbShare).Association("Users").Clear(); err != nil {
		return errors.Wrap(err, "failed to clear Users associations")
	}
	if err := s.db.WithContext(s.ctx).Model(&dbShare).Association("RoUsers").Clear(); err != nil {
		return errors.Wrap(err, "failed to clear RoUsers associations")
	}

	// Now perform the soft delete
	_, errS := gorm.G[dbom.ExportedShare](s.db).
		Where(g.ExportedShare.Name.Eq(name)).Delete(s.ctx)
	if errS != nil {
		return errors.Wrap(errS, "failed to delete share")
	}
	return nil
}

func (s *ShareService) GetShareFromPath(path string) (*dto.SharedResource, errors.E) {
	share, err := gorm.G[dbom.ExportedShare](s.db).
		Preload("MountPointData", nil).
		Preload("Users", nil).
		Preload("RoUsers", nil).
		Where(g.ExportedShare.MountPointDataPath.Eq(path)).First(s.ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.WithStack(dto.ErrorShareNotFound)
		}
		return nil, errors.Wrap(err, "failed to get share by mount path")
	}
	var conv converter.DtoToDbomConverterImpl
	dtoShare, errS := conv.ExportedShareToSharedResource(share)
	if errS != nil {
		return nil, errors.Wrap(errS, "failed to convert share")
	}

	if err := s.VerifyShare(&dtoShare); err != nil {
		slog.Warn("Share verification failed", "share", dtoShare.Name, "err", err)
	}

	return &dtoShare, nil
}

func (s *ShareService) SetShareFromPathEnabled(path string, enabled bool) (*dto.SharedResource, errors.E) {
	share, err := gorm.G[dbom.ExportedShare](s.db).
		Preload("MountPointData", nil).
		Preload("Users", nil).
		Preload("RoUsers", nil).
		Where(g.ExportedShare.MountPointDataPath.Eq(path)).First(s.ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.WithStack(dto.ErrorShareNotFound)
		}
		return nil, errors.Wrap(err, "failed to get share by mount path")
	}
	if share.Disabled != nil && *share.Disabled == !enabled {
		// No change needed
		tlog.Debug("No update on Share", "path", path, "share", share)
		var conv converter.DtoToDbomConverterImpl
		dtoShare, err := conv.ExportedShareToSharedResource(share)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert share")
		}
		return &dtoShare, nil
	}

	disabled := !enabled
	share.Disabled = &disabled
	_, err = gorm.G[dbom.ExportedShare](s.db).Updates(s.ctx, share)
	if err != nil {
		return nil, errors.Wrap(err, "failed to save share")
	}

	if *share.Disabled && (share.Usage == dto.UsageAsMedia || share.Usage == dto.UsageAsBackup || share.Usage == dto.UsageAsShare) {
		// Umount the volume if the share is being disabled and is of type media/backup/share
		errE := s.supervisor_service.NetworkUnmountShare(s.ctx, share.Name)
		if errE != nil {
			slog.Error("Failed to unmount volume for disabled share", "share", share.Name, "error", errE)
		}
	}

	var conv converter.DtoToDbomConverterImpl
	dtoShare, errS := conv.ExportedShareToSharedResource(share)
	if errS != nil {
		return nil, errors.Wrap(errS, "failed to convert share")
	}
	s.eventBus.EmitShare(events.ShareEvent{
		Event: events.Event{Type: events.EventTypes.UPDATE},
		Share: &dtoShare,
	})

	return &dtoShare, nil
}

func (s *ShareService) setShareEnabled(name string, enabled bool) (*dto.SharedResource, errors.E) {
	share, err := gorm.G[dbom.ExportedShare](s.db).
		Preload("MountPointData", nil).
		Preload("Users", nil).
		Preload("RoUsers", nil).
		Where(g.ExportedShare.Name.Eq(name)).First(s.ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.WithStack(dto.ErrorShareNotFound)
		}
		return nil, errors.Wrap(err, "failed to get share")
	}
	disabled := !enabled
	share.Disabled = &disabled
	_, err = gorm.G[dbom.ExportedShare](s.db).Updates(s.ctx, share)
	if err != nil {
		return nil, errors.Errorf("failed to save share %w", err)
	}
	var conv converter.DtoToDbomConverterImpl
	dtoShare, errS := conv.ExportedShareToSharedResource(share)
	if errS != nil {
		return nil, errors.Wrap(errS, "failed to convert share")
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
	if share.Status == nil {
		share.Status = &dto.SharedResourceStatus{}
	}

	// Case 4: Check if MountPointData exists and has a valid path
	if share.MountPointData == nil || share.MountPointData.Path == "" {
		slog.Warn("Share has no valid MountPointData", "share", share.Name)
		share.Status.IsValid = false
		return nil
	}

	// Case 4: Volume doesn't exist (marked as invalid in mount point)
	if share.MountPointData.IsInvalid {
		slog.Warn("Share volume does not exist",
			"share", share.Name,
			"path", share.MountPointData.Path)
		share.Status.IsValid = false
		return nil
	}

	// Case 3: Volume exists but is not mounted
	if !share.MountPointData.IsMounted {
		slog.Warn("Share volume is not mounted",
			"share", share.Name,
			"path", share.MountPointData.Path)
		share.Status.IsValid = false
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
	share.Status.IsValid = true
	return nil
}

func (s *ShareService) SetSupervisorService(svc SupervisorServiceInterface) {
	s.supervisor_service = svc
}
