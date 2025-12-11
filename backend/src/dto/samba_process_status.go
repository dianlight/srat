package dto

import (
	"time"
)

// ProcessStatus represents the status of a process or subprocess.
//
// PID Convention:
//   - Positive PIDs: Real OS processes (e.g., smbd, nmbd, srat-server)
//   - Negative PIDs: Virtual subprocesses or monitoring threads
//     The absolute value of a negative PID indicates the parent process PID.
//     For example, if srat-server has PID 1234, its subprocess (like powersave-monitor)
//     will have PID -1234. This convention allows distinguishing between real processes
//     and internal monitoring threads/subprocesses in the UI.
type ProcessStatus struct {
	Pid           int32     `json:"pid"` // Process ID (negative for subprocesses, see above)
	Name          string    `json:"name"`
	CreateTime    time.Time `json:"create_time"`
	CPUPercent    float64   `json:"cpu_percent"`
	MemoryPercent float32   `json:"memory_percent"`
	OpenFiles     int       `json:"open_files"`
	Connections   int       `json:"connections"` // For subprocesses: number of monitored entities
	Status        []string  `json:"status"`
	IsRunning     bool      `json:"is_running"`
	Managed       bool      `json:"managed"` // Whether the process is managed by the srat daemon or not
}

// SambaProcessStatus represents the status of all Samba-related processes and subprocesses.
type SambaProcessStatus map[string]*ProcessStatus
