package converter

import (
	"time"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
)

// goverter:converter
// goverter:output:file ./issue_to_dto_conv_gen.go
type IssueToDtoConverter interface {
	// goverter:map CreatedAt Date | timeToTime
	ToDto(source *dbom.Issue) *dto.Issue
	ToDtoList(source []*dbom.Issue) []*dto.Issue
	// goverter:map Date CreatedAt | timeToTime
	// goverter:ignore UpdatedAt DeletedAt
	ToDbom(source *dto.Issue) *dbom.Issue
}

func timeToTime(source time.Time) time.Time {
	return source
}
