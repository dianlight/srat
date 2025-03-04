package repository

import (
	"sync"

	"github.com/dianlight/srat/dbom"
	"github.com/ztrue/tracerr"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ExportedShareRepository struct {
	db    *gorm.DB
	mutex sync.RWMutex
}

type ExportedShareRepositoryInterface interface {
	All(shares *[]dbom.ExportedShare) error
	Save(share *dbom.ExportedShare) error
	SaveAll(shares *[]dbom.ExportedShare) error
	FindByName(name string) (*dbom.ExportedShare, error)
	Delete(name string) error
	UpdateName(old_name string, new_name string) error
}

func (p *ExportedShareRepository) UpdateName(old_name string, new_name string) error {
	// Get / Save Users end RoUsers association

	err := p.db.
		Model(&dbom.ExportedShare{Name: old_name}).Update("name", new_name).Error
	if err != nil {
		return tracerr.Wrap(err)
	}
	return nil
}

func NewExportedShareRepository(db *gorm.DB) ExportedShareRepositoryInterface {
	return &ExportedShareRepository{
		mutex: sync.RWMutex{},
		db:    db,
	}
}

func (r *ExportedShareRepository) All(shares *[]dbom.ExportedShare) error {
	err := r.db.Preload(clause.Associations).Find(shares).Error
	if err != nil {
		return tracerr.Wrap(err)
	}
	return err
}

func (p *ExportedShareRepository) FindByName(name string) (*dbom.ExportedShare, error) {
	var share dbom.ExportedShare
	err := p.db.Preload(clause.Associations).First(&share, "name = ?", name).Error
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	return &share, nil
}

func (p *ExportedShareRepository) SaveAll(shares *[]dbom.ExportedShare) error {
	err := p.db.Session(&gorm.Session{FullSaveAssociations: true}).Save(shares).Error
	if err != nil {
		return tracerr.Wrap(err)
	}
	return nil
}

func (p *ExportedShareRepository) Save(share *dbom.ExportedShare) error {
	err := p.db.Session(&gorm.Session{FullSaveAssociations: true}).Save(share).Error
	if err != nil {
		return tracerr.Wrap(err)
	}
	return nil
}

func (p *ExportedShareRepository) Delete(name string) error {
	return p.db.Select(clause.Associations).Delete(&dbom.ExportedShare{Name: name}).Error
}
