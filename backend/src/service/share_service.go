package service

import (
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/xorcare/pointer"
	"gitlab.com/tozd/go/errors"
	"gorm.io/gorm"
)

// ShareServiceInterface defines the operations for managing shared resources.
type ShareServiceInterface interface {
	ListShares() ([]dto.SharedResource, error)
	GetShare(name string) (*dto.SharedResource, error)
	GetShareFromPath(path string) (*dto.SharedResource, error)
	CreateShare(share dto.SharedResource) (*dto.SharedResource, error)
	UpdateShare(name string, shareUpdate dto.SharedResource) (*dto.SharedResource, error)
	DeleteShare(name string) error
	DisableShare(name string) (*dto.SharedResource, error)
	EnableShare(name string) (*dto.SharedResource, error)
	DisableShareFromPath(path string) (*dto.SharedResource, error)
	EnableShareFromPath(path string) (*dto.SharedResource, error)
}

type shareService struct {
	shareRepo  repository.ExportedShareRepositoryInterface
	userRepo   repository.SambaUserRepositoryInterface
	converter  converter.DtoToDbomConverterInterface
	supervisor SupervisorServiceInterface
	dirty      DirtyDataServiceInterface // Optional, if needed for dirty data handling
}

// NewShareService creates a new instance of ShareServiceInterface.
func NewShareService(
	shareRepo repository.ExportedShareRepositoryInterface,
	userRepo repository.SambaUserRepositoryInterface,
	supervisor SupervisorServiceInterface,
	dirty DirtyDataServiceInterface,
) ShareServiceInterface {
	return &shareService{
		shareRepo:  shareRepo,
		userRepo:   userRepo,
		supervisor: supervisor,
		converter:  &converter.DtoToDbomConverterImpl{},
		dirty:      dirty,
	}
}

// ListShares retrieves all shared resources.
func (s *shareService) ListShares() ([]dto.SharedResource, error) {
	dbShares, err := s.shareRepo.All()
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve all shares from repository")
	}

	var dtoShares []dto.SharedResource
	if dbShares != nil {
		for _, dbShare := range *dbShares {
			var dtoShare dto.SharedResource
			if err := s.converter.ExportedShareToSharedResource(dbShare, &dtoShare); err != nil {
				return nil, errors.Wrapf(err, "failed to convert dbom.ExportedShare to dto.SharedResource for share %s", dbShare.Name)
			}
			// Check Supervisor status
			if dtoShare.Usage != "internal" && dtoShare.Usage != "none" {
				mount, err := s.supervisor.NetworkGetMountByName(dtoShare.Name)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to get network mount for share %s", dtoShare.Name)
				}
				if mount == nil {
					dtoShare.IsHAMounted = pointer.Bool(false)
				} else {
					dtoShare.IsHAMounted = pointer.Bool(true)
				}
			}

			dtoShares = append(dtoShares, dtoShare)
		}
	}
	return dtoShares, nil
}

// GetShare retrieves a specific share by its name.
func (s *shareService) GetShare(name string) (*dto.SharedResource, error) {
	dbShare, err := s.shareRepo.FindByName(name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrapf(dto.ErrorShareNotFound, "share with name '%s' not found", name)
		}
		return nil, errors.Wrapf(err, "failed to find share '%s' in repository", name)
	}

	var dtoShare dto.SharedResource
	if err := s.converter.ExportedShareToSharedResource(*dbShare, &dtoShare); err != nil {
		return nil, errors.Wrapf(err, "failed to convert dbom.ExportedShare to dto.SharedResource for share '%s'", dbShare.Name)
	}
	return &dtoShare, nil
}

// GetShareFromPath retrieves a specific share by its mount path.
func (s *shareService) GetShareFromPath(path string) (*dto.SharedResource, error) {
	dbShare, err := s.shareRepo.FindByMountPath(path)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrapf(dto.ErrorShareNotFound, "no share found for path '%s'", path)
		}
		return nil, errors.Wrapf(err, "failed to find share by path '%s' in repository", path)
	}

	var dtoShare dto.SharedResource
	if err := s.converter.ExportedShareToSharedResource(*dbShare, &dtoShare); err != nil {
		return nil, errors.Wrapf(err, "failed to convert dbom.ExportedShare to dto.SharedResource for share %s (path %s)", dbShare.Name, path)
	}

	// Check supervisor status
	if dtoShare.Usage != "internal" && dtoShare.Usage != "none" {
		if mount, err := s.supervisor.NetworkGetMountByName(dtoShare.Name); err != nil {
			return nil, errors.Wrapf(err, "failed to get network mount for share %s (path %s)", dtoShare.Name, path)
		} else if mount == nil {
			dtoShare.IsHAMounted = pointer.Bool(false)
		} else {
			dtoShare.IsHAMounted = pointer.Bool(true)
		}
	}

	return &dtoShare, nil
}

// CreateShare creates a new shared resource.
func (s *shareService) CreateShare(share dto.SharedResource) (*dto.SharedResource, error) {
	_, err := s.shareRepo.FindByName(share.Name)
	if err == nil {
		return nil, errors.Wrapf(dto.ErrorShareAlreadyExists, "share with name '%s' already exists", share.Name)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.Wrapf(err, "failed to check existence of share '%s'", share.Name)
	}

	dbShare := &dbom.ExportedShare{}
	if err := s.converter.SharedResourceToExportedShare(share, dbShare); err != nil {
		return nil, errors.Wrapf(err, "failed to convert dto.SharedResource to dbom.ExportedShare for share '%s'", share.Name)
	}

	if len(dbShare.Users) == 0 {
		adminUser, adminErr := s.userRepo.GetAdmin()
		if adminErr != nil {
			return nil, errors.Wrap(adminErr, "failed to get admin user for new share")
		}
		dbShare.Users = append(dbShare.Users, adminUser)
	}

	if err := s.shareRepo.Save(dbShare); err != nil {
		// Note: gorm.ErrDuplicatedKey might not be standard across all GORM dialects/drivers.
		// Checking for a more generic "constraint violation" or relying on the FindByName check might be more robust.
		return nil, errors.Wrapf(err, "failed to save share '%s' to repository", share.Name)
	}

	var createdDtoShare dto.SharedResource
	if err := s.converter.ExportedShareToSharedResource(*dbShare, &createdDtoShare); err != nil {
		return nil, errors.Wrapf(err, "failed to convert created dbom.ExportedShare back to dto.SharedResource for share '%s'", dbShare.Name)
	}
	return &createdDtoShare, nil
}

// UpdateShare updates an existing shared resource.
func (s *shareService) UpdateShare(currentName string, shareUpdate dto.SharedResource) (*dto.SharedResource, error) {
	dbShare, err := s.shareRepo.FindByName(currentName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrapf(dto.ErrorShareNotFound, "share with name '%s' not found for update", currentName)
		}
		return nil, errors.Wrapf(err, "failed to find share '%s' for update", currentName)
	}

	if err := s.converter.SharedResourceToExportedShare(shareUpdate, dbShare); err != nil {
		return nil, errors.Wrapf(err, "failed to convert dto.SharedResource to dbom.ExportedShare for updating share '%s'", currentName)
	}

	if currentName != dbShare.Name { // Name has changed
		_, findErr := s.shareRepo.FindByName(dbShare.Name) // Check if new name exists
		if findErr == nil {
			return nil, errors.Wrapf(dto.ErrorShareAlreadyExists, "cannot rename share to '%s', as it already exists", dbShare.Name)
		}
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return nil, errors.Wrapf(findErr, "failed to check existence of new share name '%s' during rename", dbShare.Name)
		}
		if err := s.shareRepo.UpdateName(currentName, dbShare.Name); err != nil {
			return nil, errors.Wrapf(err, "failed to update share name from '%s' to '%s'", currentName, dbShare.Name)
		}
	}

	if err := s.shareRepo.Save(dbShare); err != nil {
		return nil, errors.Wrapf(err, "failed to save updated share '%s'", dbShare.Name)
	}

	var updatedDtoShare dto.SharedResource
	if err := s.converter.ExportedShareToSharedResource(*dbShare, &updatedDtoShare); err != nil {
		return nil, errors.Wrapf(err, "failed to convert updated dbom.ExportedShare back to dto.SharedResource for share '%s'", dbShare.Name)
	}
	return &updatedDtoShare, nil
}

// DeleteShare deletes a shared resource by its name.
func (s *shareService) DeleteShare(name string) error {
	if _, err := s.GetShare(name); err != nil { // Leverage GetShare for not-found check
		return err
	}
	if err := s.shareRepo.Delete(name); err != nil {
		return errors.Wrapf(err, "failed to delete share '%s' from repository", name)
	}
	return nil
}

func (s *shareService) setShareDisabledStatus(name string, disabled bool) (*dto.SharedResource, error) {
	dbShare, err := s.shareRepo.FindByName(name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrapf(dto.ErrorShareNotFound, "share with name '%s' not found", name)
		}
		return nil, errors.Wrapf(err, "failed to find share '%s'", name)
	}

	dbShare.Disabled = disabled
	if err := s.shareRepo.Save(dbShare); err != nil {
		return nil, errors.Wrapf(err, "failed to save share '%s' with disabled status %t", name, disabled)
	}

	var dtoShare dto.SharedResource
	if err := s.converter.ExportedShareToSharedResource(*dbShare, &dtoShare); err != nil {
		return nil, errors.Wrapf(err, "failed to convert dbom.ExportedShare to dto.SharedResource for share '%s'", dbShare.Name)
	}
	s.dirty.SetDirtyShares()
	return &dtoShare, nil
}

// DisableShare disables a shared resource.
func (s *shareService) DisableShare(name string) (*dto.SharedResource, error) {
	return s.setShareDisabledStatus(name, true)
}

// EnableShare enables a shared resource.
func (s *shareService) EnableShare(name string) (*dto.SharedResource, error) {
	return s.setShareDisabledStatus(name, false)
}

// DisableShareFromPath disables a shared resource identified by its mount path.
func (s *shareService) DisableShareFromPath(path string) (*dto.SharedResource, error) {
	share, err := s.GetShareFromPath(path)
	if err != nil {
		// GetShareFromPath already wraps dto.ErrorShareNotFound
		return nil, errors.Wrapf(err, "failed to get share from path '%s' for disabling", path)
	}
	if share.Disabled != nil && *share.Disabled && share.IsHAMounted != nil && *share.IsHAMounted {
		err = s.supervisor.NetworkUnmountShare(dbom.ExportedShare{Name: share.Name})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmount share '%s' from supervisor before disabling", share.Name)
		}
	}

	return s.setShareDisabledStatus(share.Name, true)
}

// EnableShareFromPath enables a shared resource identified by its mount path.
func (s *shareService) EnableShareFromPath(path string) (*dto.SharedResource, error) {
	share, err := s.GetShareFromPath(path)
	if err != nil {
		// GetShareFromPath already wraps dto.ErrorShareNotFound
		return nil, errors.Wrapf(err, "failed to get share from path '%s' for enabling", path)
	}
	return s.setShareDisabledStatus(share.Name, false)
}
