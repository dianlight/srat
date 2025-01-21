package dto

import (
	"time"
)

type SambaProcessStatus struct {
	PID           int32     `json:"pid"`
	Name          string    `json:"name"`
	CreateTime    time.Time `json:"create_time"`
	CPUPercent    float64   `json:"cpu_percent"`
	MemoryPercent float32   `json:"memory_percent"`
	OpenFiles     int32     `json:"open_files"`
	Connections   int32     `json:"connections"`
	Status        []string  `json:"status"`
	IsRunning     bool      `json:"is_running"`
}
