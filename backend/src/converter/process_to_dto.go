package converter

import (
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

// goverter:converter
// goverter:output:file ./process_to_dto_conv_gen.go
// goverter:output:package github.com/dianlight/srat/converter
// goverter:useZeroValueOnPointerInconsistency
// goverter:update:ignoreZeroValueField
// goverter:extend int64ToTime
// goverter:default:update
type ProcessToDto interface {
	// goverter:useZeroValueOnPointerInconsistency
	// goverter:map OpenFiles OpenFiles | sliceToLen
	// goverter:map Connections Connections | sliceToLen
	ProcessToProcessStatus(source *process.Process) (target *dto.ProcessStatus, err error)
}

func int64ToTime(source int64) (time.Time, error) {
	return time.Unix(source/1000, 0), nil
}

func sliceToLen(source any) (int, error) {
	switch v := source.(type) {
	case []process.OpenFilesStat:
		return len(v), nil
	case []net.ConnectionStat:
		return len(v), nil
	}
	return 0, nil
}
