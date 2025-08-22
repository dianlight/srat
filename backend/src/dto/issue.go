package dto

import "time"

// Issue defines a problem or action that needs attention.
type Issue struct {
	ID             uint      `json:"id"`
	Date           time.Time `json:"date"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	DetailLink     string    `json:"detailLink,omitempty"`
	ResolutionLink string    `json:"resolutionLink,omitempty"`
	Repeating      uint      `json:"repeating"`
}
