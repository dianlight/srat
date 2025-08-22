package dbom

import (
	"time"

	"gorm.io/gorm"
)

// Issue defines a problem or action that needs attention.
type Issue struct {
	ID             uint `gorm:"primarykey"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
	Title          string         `gorm:"size:255,uniqueIndex"`
	Description    string
	DetailLink     string `gorm:"size:2048"`
	ResolutionLink string `gorm:"size:2048"`
	Repeating      uint   `gorm:"default:1"`
}
