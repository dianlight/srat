package dbom_test

import (
	"testing"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
)

func TestIssueFields(t *testing.T) {
	// Test Issue struct instantiation and field assignment
	severity := dto.IssueSeverities.ISSUESEVERITYERROR
	issue := dbom.Issue{
		Title:          "Test Issue",
		Description:    "This is a test issue",
		DetailLink:     "https://example.com/details",
		ResolutionLink: "https://example.com/resolution",
		Repeating:      2,
		Ignored:        false,
		Severity:       &severity,
	}

	assert.Equal(t, "Test Issue", issue.Title)
	assert.Equal(t, "This is a test issue", issue.Description)
	assert.Equal(t, "https://example.com/details", issue.DetailLink)
	assert.Equal(t, "https://example.com/resolution", issue.ResolutionLink)
	assert.Equal(t, uint(2), issue.Repeating)
	assert.False(t, issue.Ignored)
	assert.NotNil(t, issue.Severity)
	assert.Equal(t, dto.IssueSeverities.ISSUESEVERITYERROR, *issue.Severity)
}

func TestIssueDefaults(t *testing.T) {
	// Test Issue struct with zero values
	issue := dbom.Issue{
		Title: "Default Issue",
	}

	assert.Equal(t, "Default Issue", issue.Title)
	assert.Empty(t, issue.Description)
	assert.Equal(t, uint(0), issue.Repeating)
	assert.False(t, issue.Ignored)
	assert.Nil(t, issue.Severity)
}

func TestIssueSeverityLevels(t *testing.T) {
	// Test all severity levels
	tests := []struct {
		name     string
		severity dto.IssueSeverity
	}{
		{"Error", dto.IssueSeverities.ISSUESEVERITYERROR},
		{"Warning", dto.IssueSeverities.ISSUESEVERITYWARNING},
		{"Info", dto.IssueSeverities.ISSUESEVERITYINFO},
		{"Success", dto.IssueSeverities.ISSUESEVERITYSUCCESS},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := dbom.Issue{
				Title:    "Test " + tt.name,
				Severity: new(tt.severity),
			}
			assert.NotNil(t, issue.Severity)
			assert.Equal(t, tt.severity, *issue.Severity)
		})
	}
}
