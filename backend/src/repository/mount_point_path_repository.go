package repository

/*
import (
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/repository/dao"
	"gitlab.com/tozd/go/errors"
	"gorm.io/gorm"
)

type MountPointPathRepository struct {
	db *gorm.DB
}

type MountPointPathRepositoryInterface interface {
	All() ([]*dbom.MountPointPath, errors.E)
	Save(mp *dbom.MountPointPath) errors.E
	FindByPath(path string) (*dbom.MountPointPath, errors.E)
	FindByDevice(device string) ([]*dbom.MountPointPath, errors.E)
	Delete(path string) errors.E
}

func NewMountPointPathRepository(db *gorm.DB) MountPointPathRepositoryInterface {
	dao.SetDefault(db)
	return &MountPointPathRepository{
		db: db,
	}
}

func (r *MountPointPathRepository) Save(mp *dbom.MountPointPath) errors.E {
	return errors.WithStack(dao.MountPointPath.Save(mp))
}

func (r *MountPointPathRepository) FindByPath(path string) (*dbom.MountPointPath, errors.E) {
	mount, err := dao.MountPointPath.Where(dao.MountPointPath.Path.Eq(path)).First()
	return mount, errors.WithStack(err)
}

func (r *MountPointPathRepository) FindByDevice(device string) ([]*dbom.MountPointPath, errors.E) {
	mount, err := dao.MountPointPath.Where(dao.MountPointPath.DeviceId.Eq(device)).Find()
	return mount, errors.WithStack(err)
}
func (r *MountPointPathRepository) All() (Data []*dbom.MountPointPath, Error errors.E) {
	mounts, err := dao.MountPointPath.Find()
	return mounts, errors.WithStack(err)
}

func (r *MountPointPathRepository) Delete(path string) errors.E {
	_, err := dao.MountPointPath.Where(dao.MountPointPath.Path.Eq(path)).Delete()
	return errors.WithStack(err)
}
*/
