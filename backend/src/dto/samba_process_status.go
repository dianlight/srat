package dto

import (
	"time"
)

type SambaProcessStatus struct {
	Pid           int32     `json:"pid"`
	Name          string    `json:"name"`
	CreateTime    time.Time `json:"create_time"`
	CPUPercent    float64   `json:"cpu_percent"`
	MemoryPercent float32   `json:"memory_percent"`
	OpenFiles     int       `json:"open_files"`
	Connections   int       `json:"connections"`
	Status        []string  `json:"status"`
	IsRunning     bool      `json:"is_running"`
}
