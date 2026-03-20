package dto

import "fmt"

type RepairCommandAction string

const (
	RepairCommandActionUpsert    RepairCommandAction = "upsert"
	RepairCommandActionDelete    RepairCommandAction = "delete"
	RepairCommandActionReconcile RepairCommandAction = "reconcile"
)

type RepairIssueSeverity string

const (
	RepairIssueSeverityWarning  RepairIssueSeverity = "warning"
	RepairIssueSeverityError    RepairIssueSeverity = "error"
	RepairIssueSeverityCritical RepairIssueSeverity = "critical"
)

type RepairCommandMessage struct {
	CommandID               string              `json:"command_id"`
	RepairID                string              `json:"repair_id"`
	Action                  RepairCommandAction `json:"action"`
	TranslationKey          string              `json:"translation_key,omitempty"`
	TranslationPlaceholders map[string]string   `json:"translation_placeholders,omitempty"`
	Data                    map[string]any      `json:"data,omitempty"`
	LearnMoreURL            *string             `json:"learn_more_url,omitempty"`
	BreaksInHAVersion       *string             `json:"breaks_in_ha_version,omitempty"`
	Severity                RepairIssueSeverity `json:"severity,omitempty"`
	IsFixable               bool                `json:"is_fixable"`
	IsPersistent            bool                `json:"is_persistent"`
}

func (msg RepairCommandMessage) Validate() error {
	if msg.CommandID == "" {
		return fmt.Errorf("command_id is required")
	}
	if msg.RepairID == "" {
		return fmt.Errorf("repair_id is required")
	}

	switch msg.Action {
	case RepairCommandActionUpsert, RepairCommandActionDelete, RepairCommandActionReconcile:
	default:
		return fmt.Errorf("invalid repair action %q", msg.Action)
	}

	if msg.Action != RepairCommandActionDelete {
		if msg.TranslationKey == "" {
			return fmt.Errorf("translation_key is required")
		}
		switch msg.Severity {
		case RepairIssueSeverityWarning, RepairIssueSeverityError, RepairIssueSeverityCritical:
		default:
			return fmt.Errorf("invalid repair severity %q", msg.Severity)
		}
	}

	return nil
}

type RepairLifecycleStatus string

const (
	RepairLifecycleStatusCreated   RepairLifecycleStatus = "created"
	RepairLifecycleStatusUpdated   RepairLifecycleStatus = "updated"
	RepairLifecycleStatusIgnored   RepairLifecycleStatus = "ignored"
	RepairLifecycleStatusFixed     RepairLifecycleStatus = "fixed"
	RepairLifecycleStatusDismissed RepairLifecycleStatus = "dismissed"
	RepairLifecycleStatusDeleted   RepairLifecycleStatus = "deleted"
	RepairLifecycleStatusError     RepairLifecycleStatus = "error"
)

type RepairLifecycleMessage struct {
	Type      string                `json:"type"`
	CommandID string                `json:"command_id,omitempty"`
	RepairID  string                `json:"repair_id"`
	Status    RepairLifecycleStatus `json:"status"`
	Error     *string               `json:"error,omitempty"`
	Details   map[string]any        `json:"details,omitempty"`
}

func (msg RepairLifecycleMessage) Validate() error {
	if msg.Type != ClientEventTypes.CLIENTEVENTTYPEREPAIRLIFECYCLE.String() {
		return fmt.Errorf("invalid lifecycle type %q", msg.Type)
	}
	if msg.RepairID == "" {
		return fmt.Errorf("repair_id is required")
	}

	switch msg.Status {
	case RepairLifecycleStatusCreated,
		RepairLifecycleStatusUpdated,
		RepairLifecycleStatusIgnored,
		RepairLifecycleStatusFixed,
		RepairLifecycleStatusDismissed,
		RepairLifecycleStatusDeleted,
		RepairLifecycleStatusError:
		return nil
	default:
		return fmt.Errorf("invalid repair lifecycle status %q", msg.Status)
	}
}
