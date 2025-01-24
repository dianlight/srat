package converter

import (
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/shirou/gopsutil/v4/process"
)

// goverter:converter
// goverter:output:file ./process_to_dto_conv_gen.go
// goverter:output:package github.com/dianlight/srat/converter
// goverter:useZeroValueOnPointerInconsistency
// goverter:update:ignoreZeroValueField
// -goverter:extend funcToInt64
// -goverter:extend sliceToInt
// goverter:extend int64ToTime
// goverter:default:update
type ProcessToDto interface {
	// goverter:update target
	// goverter:useZeroValueOnPointerInconsistency
	// -goverter:map CreateTime CreateTime | funcToTime
	// goverter:map OpenFiles OpenFiles | sliceToLen
	// goverter:map Connections Connections | sliceToLen
	ProcessToSambaProcessStatus(source *process.Process, target *dto.SambaProcessStatus) error
}

/*
func funcToInt64(source func() (int64, error)) (int64, error) {
	return source()
}

func funcToTime(source func() (int64, error)) (time.Time, error) {
	createTime, err := source()
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(createTime/1000, 0), nil
}
*/

func int64ToTime(source int64) (time.Time, error) {
	return time.Unix(source/1000, 0), nil
}

func sliceToLen(source any) (int, error) {
	return len(source.([]any)), nil
}

/*
	createTime, _ := spid.CreateTime()

	sambaP.Name = gog.Must(spid.Name())
	sambaP.CreateTime = time.Unix(createTime/1000, 0)
	sambaP.CPUPercent = gog.Must(spid.CPUPercent())
	sambaP.MemoryPercent = gog.Must(spid.MemoryPercent())
	sambaP.OpenFiles = int32(len(gog.Must(spid.OpenFiles())))
	sambaP.Connections = int32(len(gog.Must(spid.Connections())))
	sambaP.Status = gog.Must(spid.Status())
	sambaP.IsRunning = gog.Must(spid.IsRunning())

	HttpJSONReponse(w, sambaP, nil)
*/
