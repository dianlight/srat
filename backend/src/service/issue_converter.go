package service

import (
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
)

// goverter:converter
// goverter:output:file ./issue_to_dto_conv_gen.go
// goverter:output:package service
// goverter:extend TimeToTime
type IssueToDtoConverter interface {
	// goverter:map CreatedAt Date
	ToDto(source *dbom.Issue) *dto.Issue
	ToDtoList(source []*dbom.Issue) []*dto.Issue
	// goverter:map Date CreatedAt
	// goverter:ignore UpdatedAt DeletedAt
	ToDbom(source *dto.Issue) *dbom.Issue
}
