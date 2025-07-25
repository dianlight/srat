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
	All() (*[]dbom.ExportedShare, error)
	SaveAll(*[]dbom.ExportedShare) error
	ListShares() ([]dto.SharedResource, error)
	GetShare(name string) (*dto.SharedResource, error)
	CreateShare(share dto.SharedResource) (*dto.SharedResource, error)
	UpdateShare(name string, share dto.SharedResource) (*dto.SharedResource, error)
	DeleteShare(name string) error
	DisableShare(name string) (*dto.SharedResource, error)
	EnableShare(name string) (*dto.SharedResource, error)
	GetShareFromPath(path string) (*dto.SharedResource, error)
	DisableShareFromPath(path string) (*dto.SharedResource, error)
	NotifyClient()
	VerifyShare(share *dto.SharedResource) error
	//SetVolumeService(v VolumeServiceInterface)
}

type ShareService struct {
	exported_share_repo repository.ExportedShareRepositoryInterface
	samba_user_repo     repository.SambaUserRepositoryInterface
	mount_repo          repository.MountPointPathRepositoryInterface
	broadcaster         BroadcasterServiceInterface
	sharesQueueMutex    *sync.RWMutex
	// volumeService       VolumeServiceInterface
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
	}
}

func (s *ShareService) All() (*[]dbom.ExportedShare, error) {
	return s.exported_share_repo.All()
}

func (s *ShareService) SaveAll(shares *[]dbom.ExportedShare) error {
	return s.exported_share_repo.SaveAll(shares)
}

func (s *ShareService) ListShares() ([]dto.SharedResource, error) {
	shares, err := s.exported_share_repo.All()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list shares")
	}
	var conv converter.DtoToDbomConverterImpl
	var dtoShares []dto.SharedResource
	for _, share := range *shares {
		var dtoShare dto.SharedResource
		err := conv.ExportedShareToSharedResource(share, &dtoShare)
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

func (s *ShareService) GetShare(name string) (*dto.SharedResource, error) {
	share, err := s.exported_share_repo.FindByName(name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get share")
	}
	if share == nil {
		return nil, dto.ErrorShareNotFound
	}
	var conv converter.DtoToDbomConverterImpl
	var dtoShare dto.SharedResource
	err = conv.ExportedShareToSharedResource(*share, &dtoShare)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert share")
	}

	if err := s.VerifyShare(&dtoShare); err != nil {
		slog.Warn("Share verification failed", "share", dtoShare.Name, "err", err)
	}

	return &dtoShare, nil
}

func (s *ShareService) CreateShare(share dto.SharedResource) (*dto.SharedResource, error) {
	existing, err := s.exported_share_repo.FindByName(share.Name)
	if err != nil && !errors.Is(err, dto.ErrorShareNotFound) {
		return nil, errors.Wrap(err, "failed to check for existing share")
	}
	if existing != nil {
		return nil, dto.ErrorShareAlreadyExists
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

	err = s.mount_repo.Save(&dbShare.MountPointData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to save mount point")
	}

	err = s.exported_share_repo.Save(&dbShare)
	if err != nil {
		return nil, errors.Wrap(err, "failed to save share")
	}

	var dtoShare dto.SharedResource
	var convOut converter.DtoToDbomConverterImpl
	err = convOut.ExportedShareToSharedResource(dbShare, &dtoShare)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert share")
	}

	if err := s.VerifyShare(&dtoShare); err != nil {
		slog.Warn("Share verification failed", "share", dtoShare.Name, "err", err)
	}

	return &dtoShare, nil
}

func (s *ShareService) UpdateShare(name string, share dto.SharedResource) (*dto.SharedResource, error) {
	dbShare, err := s.exported_share_repo.FindByName(name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get share")
	}
	if dbShare == nil {
		return nil, dto.ErrorShareNotFound
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
	if err := conv.ExportedShareToSharedResource(*dbShare, &createdDtoShare); err != nil {
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
	err = convOut.ExportedShareToSharedResource(*dbShare, &dtoShare)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert share")
	}

	return &dtoShare, nil
}

// DeleteShare deletes a shared resource by its name.
func (s *ShareService) DeleteShare(name string) error {
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

func (s *ShareService) findByPath(path string) (*dbom.ExportedShare, error) {
	shares, err := s.exported_share_repo.All()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list shares")
	}

	for i := range *shares {
		if (*shares)[i].MountPointData.Path == path {
			return &(*shares)[i], nil
		}
	}

	return nil, dto.ErrorShareNotFound
}

func (s *ShareService) GetShareFromPath(path string) (*dto.SharedResource, error) {
	share, err := s.findByPath(path)
	if err != nil {
		return nil, err // This will propagate ErrorShareNotFound
	}

	var conv converter.DtoToDbomConverterImpl
	var dtoShare dto.SharedResource
	err = conv.ExportedShareToSharedResource(*share, &dtoShare)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert share")
	}
	return &dtoShare, nil
}

func (s *ShareService) DisableShareFromPath(path string) (*dto.SharedResource, error) {
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
	err = conv.ExportedShareToSharedResource(*share, &dtoShare)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert share")
	}
	return &dtoShare, nil
}

func (s *ShareService) setShareEnabled(name string, enabled bool) (*dto.SharedResource, error) {
	share, err := s.exported_share_repo.FindByName(name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get share")
	}
	if share == nil {
		return nil, dto.ErrorShareNotFound
	}
	disabled := !enabled
	share.Disabled = &disabled
	err = s.exported_share_repo.Save(share)
	if err != nil {
		return nil, errors.Wrap(err, "failed to save share")
	}
	var conv converter.DtoToDbomConverterImpl
	var dtoShare dto.SharedResource
	err = conv.ExportedShareToSharedResource(*share, &dtoShare)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert share")
	}
	return &dtoShare, nil
}

func (s *ShareService) DisableShare(name string) (*dto.SharedResource, error) {
	return s.setShareEnabled(name, false)
}

func (s *ShareService) EnableShare(name string) (*dto.SharedResource, error) {
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
func (s *ShareService) VerifyShare(share *dto.SharedResource) error {
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
