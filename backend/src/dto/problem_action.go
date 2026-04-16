package dto

// ProblemAction defines a UI/API action associated with a problem.
type ProblemAction struct {
	Key       string  `json:"key"`
	Label     string  `json:"label"`
	URL       *string `json:"url,omitempty"`
	IsDefault bool    `json:"is_default,omitempty"`
}
