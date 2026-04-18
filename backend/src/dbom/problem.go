package dbom

import (
	"time"

	"github.com/dianlight/srat/dto"
	"gorm.io/gorm"
)

// Problem stores the unified problem entity in the database.
type Problem struct {
	ID                      uint `gorm:"primarykey"`
	CreatedAt               time.Time
	UpdatedAt               time.Time
	DeletedAt               gorm.DeletedAt `gorm:"index"`
	ProblemKey              string         `gorm:"size:255,uniqueIndex"`
	Title                   string         `gorm:"size:255,index"`
	Description             string
	Severity                dto.ProblemSeverity
	Status                  dto.ProblemLifecycleStatus
	Repeating               uint                `gorm:"default:1"`
	Ignored                 bool                `gorm:"default:false"`
	Actions                 []dto.ProblemAction `gorm:"serializer:json"`
	TranslationKey          string              `gorm:"size:255"`
	TranslationPlaceholders map[string]string   `gorm:"serializer:json"`
	Data                    map[string]any      `gorm:"serializer:json"`
	LearnMoreURL            string              `gorm:"size:2048"`
	DetailLink              string              `gorm:"size:2048"`
	ResolutionLink          string              `gorm:"size:2048"`
	IsFixable               bool
	IsPersistent            bool
	LastError               string
}
