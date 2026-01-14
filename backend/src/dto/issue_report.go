package dto

// IssueReportRequest represents a request to export diagnostic data for issue reporting
type IssueReportRequest struct {
	ProblemType        ProblemType `json:"problem_type" enum:"frontend_ui,ha_integration,addon,samba"`
	Description        string      `json:"description"`
	IncludeContextData bool        `json:"include_context_data"`
	IncludeAddonLogs   bool        `json:"include_addon_logs"`
	IncludeSRATConfig  bool        `json:"include_srat_config"`
	// Context data from frontend
	CurrentURL      string   `json:"current_url,omitempty"`
	NavigationHistory []string `json:"navigation_history,omitempty"`
	BrowserInfo     string   `json:"browser_info,omitempty"`
	ConsoleErrors   []string `json:"console_errors,omitempty"`
}

// IssueReportResponse represents the response with diagnostic data
type IssueReportResponse struct {
	GitHubURL        string  `json:"github_url"`
	IssueTitle       string  `json:"issue_title"`
	IssueBody        string  `json:"issue_body"`
	SanitizedConfig  *string `json:"sanitized_config,omitempty"`
	AddonLogs        *string `json:"addon_logs,omitempty"`
}

// ProblemType represents the type of issue being reported
type ProblemType string

const (
	ProblemTypeFrontendUI     ProblemType = "frontend_ui"
	ProblemTypeHAIntegration  ProblemType = "ha_integration"
	ProblemTypeAddon          ProblemType = "addon"
	ProblemTypeSamba          ProblemType = "samba"
)
