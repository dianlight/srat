package repository

import (
	"sync"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"gitlab.com/tozd/go/errors"
	"gorm.io/gorm"
)

type HDIdleConfigRepository struct {
	db    *gorm.DB
	mutex sync.RWMutex
}

type HDIdleConfigRepositoryInterface interface {
	Get() (*dbom.HDIdleConfig, errors.E)
	Save(config *dbom.HDIdleConfig) errors.E
	Delete() errors.E
}

func NewHDIdleConfigRepository(db *gorm.DB) HDIdleConfigRepositoryInterface {
	return &HDIdleConfigRepository{
		mutex: sync.RWMutex{},
		db:    db,
	}
}

// Get retrieves the HDIdle configuration (there should only be one)
func (r *HDIdleConfigRepository) Get() (*dbom.HDIdleConfig, errors.E) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var config dbom.HDIdleConfig
	err := r.db.Preload("Devices").First(&config).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.WithStack(dto.ErrorNotFound)
		}
		return nil, errors.WithStack(err)
	}
	return &config, nil
}

// Save saves the HDIdle configuration
func (r *HDIdleConfigRepository) Save(config *dbom.HDIdleConfig) errors.E {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Start a transaction
	tx := r.db.Begin()
	if tx.Error != nil {
		return errors.WithStack(tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// If config has an ID, update it, otherwise create
	if config.ID != 0 {
		// Delete existing devices first
		if err := tx.Where("config_id = ?", config.ID).Delete(&dbom.HDIdleDevice{}).Error; err != nil {
			tx.Rollback()
			return errors.WithStack(err)
		}

		// Update config
		if err := tx.Save(config).Error; err != nil {
			tx.Rollback()
			return errors.WithStack(err)
		}
	} else {
		// Create new config
		if err := tx.Create(config).Error; err != nil {
			tx.Rollback()
			return errors.WithStack(err)
		}
	}

	return errors.WithStack(tx.Commit().Error)
}

// Delete removes the HDIdle configuration
func (r *HDIdleConfigRepository) Delete() errors.E {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var config dbom.HDIdleConfig
	if err := r.db.First(&config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // Already deleted
		}
		return errors.WithStack(err)
	}

	return errors.WithStack(r.db.Delete(&config).Error)
}
