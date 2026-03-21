package dto

import "time"

type ManagedRepair struct {
	RepairID      string                 `json:"repair_id"`
	LastCommandID string                 `json:"last_command_id,omitempty"`
	LastAction    RepairCommandAction    `json:"last_action,omitempty"`
	Status        RepairLifecycleStatus  `json:"status"`
	LastError     *string                `json:"last_error,omitempty"`
	UpdatedAt     time.Time              `json:"updated_at"`
	Command       RepairCommandMessage   `json:"command"`
	Lifecycle     *RepairLifecycleMessage `json:"lifecycle,omitempty"`
}
