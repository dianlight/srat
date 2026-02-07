package dto

// IssueReportRequest represents a request to export diagnostic data for issue reporting
type IssueReportRequest struct {
	_                struct{}    `json:"-" additionalProperties:"true"`
	ProblemType      ProblemType `json:"problem_type"`
	Title            string      `json:"title,omitempty"`
	Description      string      `json:"description"`
	ReproducingSteps string      `json:"reproducing_steps"`

	IncludeSRATConfig    bool `json:"include_srat_config"`
	IncludeAddonConfig   bool `json:"include_addon_config"`
	IncludeAddonLogs     bool `json:"include_addon_logs"`
	IncludeConsoleErrors bool `json:"include_console_errors"`
	//IncludeDatabaseDump  bool `json:"include_database_dump"`

	ConsoleErrors []string `json:"console_errors,omitempty"`
}

// IssueReportResponse represents the response with diagnostic data
type IssueReportResponse struct {
	GitHubURL  string `json:"github_url"`
	IssueTitle string `json:"issue_title"`

	SanitizedSRATConfig  *string `json:"sanitized_srat_config,omitempty"`
	SanitizedAddonConfig *string `json:"sanitized_addon_config,omitempty"`
}
