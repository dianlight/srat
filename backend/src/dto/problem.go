package dto

import "time"

// Problem is the unified domain entity for issues/repairs/notifications.
type Problem struct {
	ID                      uint                   `json:"id"`
	ProblemKey              string                 `json:"problem_key"`
	Title                   string                 `json:"title"`
	Description             string                 `json:"description"`
	Severity                ProblemSeverity        `json:"severity" enum:"info,warning,error,critical"`
	Status                  ProblemLifecycleStatus `json:"status" enum:"created,updated,ignored,fixed,dismissed,deleted,error"`
	Repeating               uint                   `json:"repeating"`
	Ignored                 bool                   `json:"ignored"`
	Actions                 []ProblemAction        `json:"actions,omitempty"`
	TranslationKey          string                 `json:"translation_key,omitempty"`
	TranslationPlaceholders map[string]string      `json:"translation_placeholders,omitempty"`
	Data                    map[string]any         `json:"data,omitempty"`
	LearnMoreURL            *string                `json:"learn_more_url,omitempty"`
	DetailLink              string                 `json:"detail_link,omitempty"`
	ResolutionLink          string                 `json:"resolution_link,omitempty"`
	IsFixable               bool                   `json:"is_fixable,omitempty"`
	IsPersistent            bool                   `json:"is_persistent,omitempty"`
	LastError               *string                `json:"last_error,omitempty"`
	CreatedAt               time.Time              `json:"created_at"`
	UpdatedAt               time.Time              `json:"updated_at"`
}
