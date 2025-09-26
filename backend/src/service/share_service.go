package service

import (
	"log"
	"log/slog"
	"sync"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

type ShareServiceInterface interface {
	All() (*[]dto.SharedResource, errors.E)
	SaveAll(*[]dto.SharedResource) errors.E
	ListShares() ([]dto.SharedResource, errors.E)
	GetShare(name string) (*dto.SharedResource, errors.E)
	CreateShare(share dto.SharedResource) (*dto.SharedResource, errors.E)
	UpdateShare(name string, share dto.SharedResource) (*dto.SharedResource, errors.E)
	DeleteShare(name string) errors.E
	DisableShare(name string) (*dto.SharedResource, errors.E)
	EnableShare(name string) (*dto.SharedResource, errors.E)
	GetShareFromPath(path string) (*dto.SharedResource, errors.E)
	DisableShareFromPath(path string) (*dto.SharedResource, errors.E)
	NotifyClient()
	VerifyShare(share *dto.SharedResource) errors.E
}

type ShareService struct {
	exported_share_repo repository.ExportedShareRepositoryInterface
	samba_user_repo     repository.SambaUserRepositoryInterface
	mount_repo          repository.MountPointPathRepositoryInterface
	broadcaster         BroadcasterServiceInterface
	sharesQueueMutex    *sync.RWMutex
	dbomConv            converter.DtoToDbomConverterImpl
}

type ShareServiceParams struct {
	fx.In
	ExportedShareRepo repository.ExportedShareRepositoryInterface
	SambaUserRepo     repository.SambaUserRepositoryInterface
	MountRepo         repository.MountPointPathRepositoryInterface
	Broadcaster       BroadcasterServiceInterface
}

func NewShareService(in ShareServiceParams) ShareServiceInterface {
	return &ShareService{
		exported_share_repo: in.ExportedShareRepo,
		samba_user_repo:     in.SambaUserRepo,
		mount_repo:          in.MountRepo,
		broadcaster:         in.Broadcaster,
		sharesQueueMutex:    &sync.RWMutex{},
		dbomConv:            converter.DtoToDbomConverterImpl{},
	}
}

func (s *ShareService) All() (*[]dto.SharedResource, errors.E) {
	sh, errE := s.exported_share_repo.All()
	if errE != nil {
		return nil, errE
	}
	ret, err := s.dbomConv.ExportedSharesToSharedResources(sh)
	return ret, errors.WithStack(err)
}

func (s *ShareService) SaveAll(shares *[]dto.SharedResource) errors.E {
	sh, err := s.dbomConv.SharedResourcesToExportedShares(shares)
	if err != nil {
		return errors.WithStack(err)
	}
	return s.exported_share_repo.SaveAll(sh)
}

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
	if err != nil && !errors.Is(err, dto.ErrorShareNotFound) {
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
		admin, err := s.samba_user_repo.GetAdmin()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get admin user")
		}
		dbShare.Users = []dbom.SambaUser{admin}
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
		adminUser, adminErr := s.samba_user_repo.GetAdmin()
		if adminErr != nil {
			return nil, errors.Wrap(adminErr, "failed to get admin user for new share")
		}
		dbShare.Users = append(dbShare.Users, adminUser)
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

	go s.NotifyClient()

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

	return &dtoShare, nil
}

// DeleteShare deletes a shared resource by its name.
func (s *ShareService) DeleteShare(name string) errors.E {
	var ashare *dto.SharedResource
	ashare, err := s.GetShare(name)
	if err != nil { // Leverage GetShare for not-found check
		return err
	}
	err = s.exported_share_repo.Delete(name)
	if err != nil {
		return errors.Wrap(err, "failed to delete share")
	}
	err = s.mount_repo.Delete(ashare.MountPointData.Path)
	if err != nil {
		return errors.Wrap(err, "failed to delete mount point")
	}
	go s.NotifyClient()
	return nil
}

func (s *ShareService) findByPath(path string) (*dbom.ExportedShare, errors.E) {
	shares, err := s.exported_share_repo.All()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list shares")
	}

	for i := range *shares {
		if (*shares)[i].MountPointData.Path == path {
			return &(*shares)[i], nil
		}
	}

	return nil, errors.WithStack(dto.ErrorShareNotFound)
}

func (s *ShareService) GetShareFromPath(path string) (*dto.SharedResource, errors.E) {
	share, err := s.findByPath(path)
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

func (s *ShareService) DisableShareFromPath(path string) (*dto.SharedResource, errors.E) {
	share, err := s.findByPath(path)
	if err != nil {
		return nil, err // This will propagate ErrorShareNotFound
	}

	disabled := true
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
	return &dtoShare, nil
}

func (s *ShareService) DisableShare(name string) (*dto.SharedResource, errors.E) {
	return s.setShareEnabled(name, false)
}

func (s *ShareService) EnableShare(name string) (*dto.SharedResource, errors.E) {
	return s.setShareEnabled(name, true)
}

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

// VerifyShare checks the validity of a share and disables it if invalid
func (s *ShareService) VerifyShare(share *dto.SharedResource) errors.E {
	if share == nil {
		return errors.New("share cannot be nil")
	}

	// Check if MountPointData exists and has a valid path
	if share.MountPointData == nil || share.MountPointData.Path == "" {
		slog.Warn("Share has no valid MountPointData", "share", share.Name)
		_, err := s.DisableShare(share.Name)
		if err != nil {
			return errors.Wrapf(err, "failed to disable invalid share '%s'", share.Name)
		}
		return nil
	}

	// Check if mount is active in Home Assistant
	if share.Usage != "internal" && share.Usage != "none" {
		if share.IsHAMounted != nil && !*share.IsHAMounted {
			slog.Warn("Share mount point is not mounted in Home Assistant",
				"share", share.Name,
				"status", share.HaStatus)
			_, err := s.DisableShare(share.Name)
			if err != nil {
				return errors.Wrapf(err, "failed to disable share '%s' with invalid mount point", share.Name)
			}
			return nil
		}
	}

	return nil
}
