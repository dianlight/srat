package converter

import (
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
)

// goverter:converter
// goverter:output:file ./problem_to_dto_conv_gen.go
// goverter:useZeroValueOnPointerInconsistency
// goverter:skipCopySameType
type ProblemToDtoConverter interface {
	ToDto(source *dbom.Problem) *dto.Problem

	ToDtoList(source []*dbom.Problem) []*dto.Problem

	// goverter:ignore DeletedAt
	ToDbom(source *dto.Problem) *dbom.Problem
}

/*
func problemTimeToTime(source time.Time) time.Time {
	return source
}
*/
