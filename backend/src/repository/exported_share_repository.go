package repository

import (
	"log/slog"
	"sync"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"gitlab.com/tozd/go/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ExportedShareRepository struct {
	db    *gorm.DB
	mutex sync.RWMutex
}

type ExportedShareRepositoryInterface interface {
	All() (*[]dbom.ExportedShare, error)
	Save(share *dbom.ExportedShare) error
	SaveAll(shares *[]dbom.ExportedShare) error
	FindByName(name string) (*dbom.ExportedShare, error)
	FindByMountPath(path string) (*dbom.ExportedShare, error)
	Delete(name string) error
	UpdateName(old_name string, new_name string) error
}

func (p *ExportedShareRepository) UpdateName(old_name string, new_name string) error {
	// Get / Save Users end RoUsers association

	err := p.db.
		Model(&dbom.ExportedShare{Name: old_name}).Update("name", new_name).Error
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func NewExportedShareRepository(db *gorm.DB) ExportedShareRepositoryInterface {
	return &ExportedShareRepository{
		mutex: sync.RWMutex{},
		db:    db,
	}
}

func (r *ExportedShareRepository) All() (*[]dbom.ExportedShare, error) {
	var shares []dbom.ExportedShare
	err := r.db.Preload(clause.Associations).Find(&shares).Error
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &shares, nil
}

func (p *ExportedShareRepository) FindByName(name string) (*dbom.ExportedShare, error) {
	var share dbom.ExportedShare
	err := p.db.Preload(clause.Associations).First(&share, "name = ?", name).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.WithStack(err)
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, dto.ErrorShareNotFound
	}
	return &share, nil
}

// FindByMountPath retrieves a specific share by its associated mount path.
func (p *ExportedShareRepository) FindByMountPath(path string) (*dbom.ExportedShare, error) {
	var share dbom.ExportedShare
	// The ExportedShare table has a MountPointDataPath column which is the foreign key to MountPointPath.Path
	err := p.db.Preload(clause.Associations).Where("mount_point_data_path = ?", path).First(&share).Error
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &share, nil
}
func (p *ExportedShareRepository) SaveAll(shares *[]dbom.ExportedShare) error {
	err := p.db.Session(&gorm.Session{FullSaveAssociations: true}).Select("*").Save(shares).Error
	if err != nil {
		return errors.WithDetails(err, "shares", shares)
	}
	return nil
}

func (p *ExportedShareRepository) Save(share *dbom.ExportedShare) error {
	return p.db.Transaction(func(tx *gorm.DB) error {
		var existingShare dbom.ExportedShare // Used to check existence
		// Check if a record with the same name (primary key) exists.
		// The BeforeSave hook in dbom.ExportedShare will validate if share.Name is empty.
		err := tx.Unscoped().First(&existingShare, "name = ?", share.Name).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Record does not exist, so create.
			slog.Debug("Record not found, creating new ExportedShare", "name", share.Name)
			// The BeforeSave hook will execute its logic for new records.
			// GORM's Create with FullSaveAssociations will handle associations.
			createErr := tx. /*.Debug()*/ /*.Session(&gorm.Session{FullSaveAssociations: true})*/ Create(share).Error
			if createErr != nil {
				slog.Error("Failed to create share", "share_name", share.Name, "error", createErr)
				return errors.WithDetails(createErr, "share_name", share.Name, "details", share)
			}
			return nil
		} else if err != nil {
			// Another error occurred during the First check (not ErrRecordNotFound)
			slog.Error("Failed to check share existence", "share_name", share.Name, "error", err)
			return errors.WithDetails(err, "share_name", share.Name, "details", share)
		} else {
			// Record exists, so update.
			slog.Debug("Record found, updating existing ExportedShare", "name", share.Name)

			// If the existing record in the DB is soft-deleted (existingShare.DeletedAt.Valid is true)
			// AND the incoming 'share' object intends for the record to be active (share.DeletedAt.Valid is false),
			// then explicitly set the 'deleted_at' column to NULL.
			// This is necessary because tx.Updates(share) with a struct argument typically only updates non-zero fields
			// from the 'share' struct by default, and would not clear 'deleted_at' if 'share.DeletedAt' is its zero value.
			if existingShare.DeletedAt.Valid && !share.DeletedAt.Valid {
				if err := tx.Model(&dbom.ExportedShare{}).Unscoped().Where("name = ?", share.Name).UpdateColumn("deleted_at", gorm.Expr("NULL")).Error; err != nil {
					slog.Error("Failed to explicitly undelete share before update", "share_name", share.Name, "error", err)
					return errors.WithDetails(err, "share_name", share.Name, "details", share)
				}
				slog.Debug("Explicitly set DeletedAt to NULL for existing share", "name", share.Name)
			}

			if len(share.RoUsers) == 0 {
				err := tx.Model(&share).Association("RoUsers").Clear()
				if err != nil {
					slog.Error("Failed to clear RoUsers association", "share_name", share.Name, "error", err)
				}
			}

			if len(share.Users) == 0 {
				err := tx.Model(&share).Association("Users").Clear()
				if err != nil {
					slog.Error("Failed to clear Users association", "share_name", share.Name, "error", err)
				}
			}

			// The BeforeSave hook will see the record exists and skip its specific creation logic.
			// Note: tx.Updates(share_struct) only updates non-zero fields by default.
			updateErr := tx. /*Debug().*/ /*.Session(&gorm.Session{FullSaveAssociations: true}).*/ Select("*").Updates(share).Error
			if updateErr != nil {
				slog.Error("Failed to update share", "share_name", share.Name, "error", updateErr)
				return errors.WithDetails(updateErr, "share_name", share.Name, "details", share)
			}
			return nil
		}
	})
}

func (p *ExportedShareRepository) Delete(name string) error {
	return p.db.Select(clause.Associations).Delete(&dbom.ExportedShare{Name: name}).Error
}
