package dto

// IssueReportRequest represents a request to export diagnostic data for issue reporting
type IssueReportRequest struct {
	ProblemType      ProblemType `json:"problem_type" enum:"frontend_ui,ha_integration,addon,samba"`
	Title            string      `json:"title"`
	Description      string      `json:"description"`
	ReproducingSteps string      `json:"reproducing_steps"`

	IncludeContextData  bool `json:"include_context_data"`
	IncludeAddonLogs    bool `json:"include_addon_logs"`
	IncludeSRATConfig   bool `json:"include_srat_config"`
	IncludeAddonConfig  bool `json:"include_addon_config"`
	IncludeDatabaseDump bool `json:"include_database_dump"`

	// Context data from frontend
	CurrentURL        string   `json:"current_url,omitempty"`
	NavigationHistory []string `json:"navigation_history,omitempty"`
	BrowserInfo       string   `json:"browser_info,omitempty"`
	ConsoleErrors     []string `json:"console_errors,omitempty"`
}

// IssueReportResponse represents the response with diagnostic data
type IssueReportResponse struct {
	GitHubURL  string `json:"github_url"`
	IssueTitle string `json:"issue_title"`

	SanitizedSRATConfig  *string `json:"sanitized_srat_config,omitempty"`
	SanitizedAddonConfig *string `json:"sanitized_addon_config,omitempty"`
	AddonLogs            *string `json:"addon_logs,omitempty"`
	DatabaseDump         *string `json:"database_dump,omitempty"`
}
