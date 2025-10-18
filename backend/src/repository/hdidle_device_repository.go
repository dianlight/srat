package repository

import (
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/repository/dao"
	"gitlab.com/tozd/go/errors"
	"gorm.io/gorm"
)

// HDIdleDeviceRepository handles database operations for HDIdleDevices.
type hDIdleDeviceRepository struct {
	db *gorm.DB
}

// HDIdleDeviceRepositoryInterface defines the methods for the HDIdleDevice repository.
type HDIdleDeviceRepositoryInterface interface {
	Create(device *dbom.HDIdleDevice) errors.E
	Update(device *dbom.HDIdleDevice) errors.E
	LoadAll() ([]*dbom.HDIdleDevice, errors.E)
	LoadByPath(path string) (*dbom.HDIdleDevice, errors.E)
	Delete(path string) errors.E
}

// NewHDIdleDeviceRepository creates a new HDIdleDevice repository.
func NewHDIdleDeviceRepository(db *gorm.DB) HDIdleDeviceRepositoryInterface {
	dao.SetDefault(db)
	return &hDIdleDeviceRepository{db: db}
}

// Create creates a new HDIdleDevice.
func (r *hDIdleDeviceRepository) Create(device *dbom.HDIdleDevice) errors.E {
	return errors.WithStack(dao.HDIdleDevice.Create(device))
}

// Update updates an existing HDIdleDevice.
func (r *hDIdleDeviceRepository) Update(device *dbom.HDIdleDevice) errors.E {
	_, err := dao.HDIdleDevice.Updates(device)
	return errors.WithStack(err)
}

func (r *hDIdleDeviceRepository) LoadAll() (ret []*dbom.HDIdleDevice, errE errors.E) {
	ret, err := dao.HDIdleDevice.Find()
	return ret, errors.WithStack(err)
}

// Delete deletes an HDIdleDevice.
func (r *hDIdleDeviceRepository) Delete(path string) errors.E {
	_, err := dao.HDIdleDevice.Where(dao.HDIdleDevice.DevicePath.Eq(path)).Delete()
	return errors.WithStack(err)
}

func (r *hDIdleDeviceRepository) LoadByPath(path string) (*dbom.HDIdleDevice, errors.E) {
	device, err := dao.HDIdleDevice.Where(dao.HDIdleDevice.DevicePath.Eq(path)).First()
	return device, errors.WithStack(err)
}
