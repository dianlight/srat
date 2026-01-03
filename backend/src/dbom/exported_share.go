package dbom

import (
	"time"

	"github.com/dianlight/srat/dto"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ExportedShare struct {
	Name               string `gorm:"primarykey"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          gorm.DeletedAt `gorm:"index"`
	Disabled           *bool
	Users              []SambaUser `gorm:"many2many:user_rw_share;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	RoUsers            []SambaUser `gorm:"many2many:user_ro_share;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	VetoFiles          datatypes.JSONSlice[string]
	TimeMachine        bool
	RecycleBin         bool `gorm:"default:false"`
	GuestOk            bool `gorm:"default:false"`
	TimeMachineMaxSize string
	Usage              dto.HAMountUsage
	MountPointDataPath string
	MountPointDataRoot string
	MountPointData     MountPointPath `gorm:"foreignKey:MountPointDataPath,MountPointDataRoot;references:Path,Root"`
}
