package dto

import (
	"time"
)

type ResolutionIssue struct {
	Type        string    `json:"type"`
	Context     string    `json:"context"`
	Reference   string    `json:"reference,omitempty"`
	Suggestion  string    `json:"suggestion,omitempty"`
	Unhealthy   bool      `json:"unhealthy"`
	Unsupported bool      `json:"unsupported"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}
