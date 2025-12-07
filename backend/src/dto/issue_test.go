package dto_test

import (
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
)

func TestIssue_Fields(t *testing.T) {
	now := time.Now()
	severity := dto.IssueSeverities.ISSUESEVERITYERROR

	issue := dto.Issue{
		ID:             1,
		Date:           now,
		Title:          "Test Issue",
		Description:    "This is a test issue",
		DetailLink:     "/issues/1",
		ResolutionLink: "/resolve/1",
		Repeating:      3,
		Ignored:        false,
		Severity:       &severity,
	}

	assert.Equal(t, uint(1), issue.ID)
	assert.Equal(t, now, issue.Date)
	assert.Equal(t, "Test Issue", issue.Title)
	assert.Equal(t, "This is a test issue", issue.Description)
	assert.Equal(t, "/issues/1", issue.DetailLink)
	assert.Equal(t, "/resolve/1", issue.ResolutionLink)
	assert.Equal(t, uint(3), issue.Repeating)
	assert.False(t, issue.Ignored)
	assert.NotNil(t, issue.Severity)
	assert.Equal(t, dto.IssueSeverities.ISSUESEVERITYERROR, *issue.Severity)
}

func TestIssue_ZeroValues(t *testing.T) {
	issue := dto.Issue{}

	assert.Equal(t, uint(0), issue.ID)
	assert.True(t, issue.Date.IsZero())
	assert.Empty(t, issue.Title)
	assert.Empty(t, issue.Description)
	assert.Empty(t, issue.DetailLink)
	assert.Empty(t, issue.ResolutionLink)
	assert.Equal(t, uint(0), issue.Repeating)
	assert.False(t, issue.Ignored)
	assert.Nil(t, issue.Severity)
}

func TestIssue_AllSeverities(t *testing.T) {
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
			issue := dto.Issue{
				Severity: &tt.severity,
			}
			assert.Equal(t, tt.severity, *issue.Severity)
		})
	}
}

func TestIssue_Ignored(t *testing.T) {
	issue := dto.Issue{
		ID:      1,
		Title:   "Ignored Issue",
		Ignored: true,
	}

	assert.True(t, issue.Ignored)
	assert.Equal(t, uint(1), issue.ID)
	assert.Equal(t, "Ignored Issue", issue.Title)
}

func TestIssue_Repeating(t *testing.T) {
	issue := dto.Issue{
		ID:        1,
		Title:     "Repeating Issue",
		Repeating: 5,
	}

	assert.Equal(t, uint(1), issue.ID)
	assert.Equal(t, "Repeating Issue", issue.Title)
	assert.Equal(t, uint(5), issue.Repeating)
}
