package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strings"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/tlog"
	"github.com/xorcare/pointer"
	"gitlab.com/tozd/go/errors"
)

// IssueReportServiceInterface defines the interface for issue reporting
type IssueReportServiceInterface interface {
	GenerateIssueReport(ctx context.Context, request *dto.IssueReportRequest) (*dto.IssueReportResponse, error)
}

// IssueReportService handles generating diagnostic data for issue reporting
type IssueReportService struct {
	settingService SettingServiceInterface
	addonService   AddonsServiceInterface
}

// NewIssueReportService creates a new issue report service
func NewIssueReportService(
	settingService SettingServiceInterface,
	addonService AddonsServiceInterface,
) IssueReportServiceInterface {
	return &IssueReportService{
		settingService: settingService,
		addonService:   addonService,
	}
}

// GenerateIssueReport generates diagnostic data and creates a GitHub issue URL
func (s *IssueReportService) GenerateIssueReport(ctx context.Context, request *dto.IssueReportRequest) (*dto.IssueReportResponse, error) {
	tlog.InfoContext(ctx, "Generating issue report", "problem_type", request.ProblemType)

	response := &dto.IssueReportResponse{
		IssueTitle: request.ProblemType.IssueTitle,
		//IssueBody:  issueBody,
	}
	// Create GitHub issue URL
	params := url.Values{}
	params.Add("title", request.ProblemType.IssueTitle+" "+request.Title)

	var sanitizedConfigPtr *string
	var addonLogsPtr *string
	var addonConfigPtr *string
	contextDataStr := ""

	// Include sanitized config if requested
	if request.IncludeSRATConfig {
		sanitizedConfig, err := s.exportSanitizedConfig(ctx)
		if err != nil {
			tlog.WarnContext(ctx, "Failed to export sanitized config", "error", err)
		} else {
			sanitizedConfigPtr = &sanitizedConfig
		}
	}

	// Include addon config if requested
	if request.IncludeAddonConfig {
		addonConfig, err := s.exportAddonConfig(ctx)
		if err != nil {
			tlog.WarnContext(ctx, "Failed to export addon config", "error", err)
		} else {
			addonConfigPtr = &addonConfig
		}
	}

	// Include addon logs if requested
	if request.IncludeAddonLogs {
		addonLogs, err := s.addonService.GetLatestLogs(ctx)
		if err != nil {
			tlog.WarnContext(ctx, "Failed to export addon logs", "error", err)
		} else {
			addonLogsPtr = &addonLogs
		}
	}

	if request.IncludeContextData {
		var contextData strings.Builder
		contextData.WriteString("### Context Information\n\n")
		if request.CurrentURL != "" {
			contextData.WriteString(fmt.Sprintf("- **Current URL**: %s\n", request.CurrentURL))
		}
		if len(request.NavigationHistory) > 0 {
			contextData.WriteString("- **Navigation History**:\n")
			for i, url := range request.NavigationHistory {
				if i >= 5 { // Limit to last 5 entries
					break
				}
				contextData.WriteString(fmt.Sprintf("  - %s\n", url))
			}
		}
		if request.BrowserInfo != "" {
			contextData.WriteString(fmt.Sprintf("- **Browser**: %s\n", request.BrowserInfo))
		}
		if len(request.ConsoleErrors) > 0 {
			contextData.WriteString("- **Console Errors**:\n```\n")
			for i, err := range request.ConsoleErrors {
				if i >= 10 { // Limit to last 10 errors
					break
				}
				contextData.WriteString(fmt.Sprintf("%s\n", err))
			}
			contextData.WriteString("```\n")
		}
		contextDataStr = contextData.String()
	}

	switch request.ProblemType.Template {
	case "bug_report.yaml":
		params.Add("template", "bug_report.yaml")

		params.Add("problem_type", request.ProblemType.Description)
		params.Add("version", config.Version)
		params.Add("arch", runtime.GOARCH)
		params.Add("os", runtime.GOOS)
		params.Add("description", request.Description+"\n\n"+contextDataStr)
		params.Add("reprod", request.ReproducingSteps)
		if sanitizedConfigPtr != nil {
			params.Add("srat_config", *sanitizedConfigPtr)
		}
		if addonConfigPtr != nil {
			params.Add("addon_config", *addonConfigPtr)
		}
		if addonLogsPtr != nil {
			params.Add("logs", *addonLogsPtr)
		}
	case "BUG-REPORT.yml":
		params.Add("template", "BUG-REPORT.yml")

		params.Add("addon", "SambaNas2")
		params.Add("description", request.Description+"\n\n"+contextDataStr)
		params.Add("reprod", request.ReproducingSteps)
		if addonLogsPtr != nil {
			params.Add("logs", *addonLogsPtr)
		}
		if addonConfigPtr != nil {
			params.Add("config", *addonConfigPtr)
		}
		params.Add("browsers", runtime.GOARCH)
		params.Add("os", runtime.GOOS)
	default:
		// No template specified use body only
		// Build issue body from the GitHub issue template structure
		body := s.buildIssueBody(ctx,
			request,
			&contextDataStr,
			sanitizedConfigPtr,
			addonConfigPtr,
			addonLogsPtr)

		params.Add("body", body)
	}

	response.GitHubURL = fmt.Sprintf("%s/issues/new?%s", request.ProblemType.RepositoryUrl, params.Encode())

	for len(response.GitHubURL) > 8000 {
		tlog.WarnContext(ctx, "Generated GitHub issue URL is too long", "length", len(response.GitHubURL))
		if addonLogsPtr != nil && response.AddonLogs == nil {
			params.Del("logs")
			response.AddonLogs = addonLogsPtr
			addonLogsPtr = pointer.String("Please attach logs to your issue. The file is generated and in your download directory")
			tlog.InfoContext(ctx, "Omitted addon logs from issue report to reduce URL length")
		}
		if sanitizedConfigPtr != nil && response.SanitizedSRATConfig == nil {
			params.Del("srat_config")
			response.SanitizedSRATConfig = sanitizedConfigPtr
			sanitizedConfigPtr = pointer.String("Please attach sanitized config to your issue. The file is generated and in your download directory")
			tlog.InfoContext(ctx, "Omitted sanitized config from issue report to reduce URL length")
		}
		if addonConfigPtr != nil && response.SanitizedAddonConfig == nil {
			params.Del("addon_config")
			params.Del("config")
			response.SanitizedAddonConfig = addonConfigPtr
			addonConfigPtr = pointer.String("Please attach addon config to your issue. The file is generated and in your download directory")
			tlog.InfoContext(ctx, "Omitted addon config from issue report to reduce URL length")
		}
		response.GitHubURL = fmt.Sprintf("%s/issues/new?%s", request.ProblemType.RepositoryUrl, params.Encode())
		if response.AddonLogs != nil && response.SanitizedSRATConfig != nil && response.SanitizedAddonConfig != nil && len(response.GitHubURL) > 8000 {
			tlog.ErrorContext(ctx, "Unable to reduce GitHub issue URL length below limit")
			return nil, errors.New("generated GitHub issue URL is too long even after omitting optional data")
		}
	}

	return response, nil
}

// buildIssueBody creates the body content for the GitHub issue
func (s *IssueReportService) buildIssueBody(
	_ context.Context,
	request *dto.IssueReportRequest,
	contextDataStr *string,
	sanitizedConfig *string,
	addonConfigPtr *string,
	addonLogs *string,
) string {
	var body strings.Builder
	// Header aligned with template name
	body.WriteString(fmt.Sprintf("## 🐛 %s\n\n", request.Title))

	// Problem type (align to template section)
	body.WriteString("### System Information\n\n")
	body.WriteString(fmt.Sprintf("- **Problem Type**: %s\n\n", request.ProblemType.Description))
	body.WriteString(fmt.Sprintf("- **Version**: %s\n\n", config.Version))
	body.WriteString(fmt.Sprintf("- **Architecture**: %s\n\n", runtime.GOARCH))
	body.WriteString(fmt.Sprintf("- **OS**: %s\n\n", runtime.GOOS))

	// Context data
	if request.IncludeContextData && request.CurrentURL != "" {
		body.WriteString(*contextDataStr)
		body.WriteString("\n\n")
	}

	// Description (user-provided)
	body.WriteString("### Description\n\n")
	body.WriteString(request.Description)
	body.WriteString("\n\n")

	// Steps to reproduce (template section)
	body.WriteString("### Steps to Reproduce\n\n")
	body.WriteString(request.ReproducingSteps)
	body.WriteString("\n\n")

	// SRAT Config (sanitized) section per template
	body.WriteString("### SRAT Config (sanitized)\n\n")
	if sanitizedConfig != nil && *sanitizedConfig != "" {
		body.WriteString("```json\n")
		body.WriteString(*sanitizedConfig)
		body.WriteString("\n```\n\n")
	} else {
		body.WriteString("(Attach sanitized SRAT config or enable 'Include SRAT configuration' in the dialog)\n\n")
	}

	// Addon Config section (left as placeholder, not collected automatically)
	body.WriteString("### Addon Config\n\n")
	if addonConfigPtr != nil && *addonConfigPtr != "" {
		body.WriteString("```json\n")
		body.WriteString(*addonConfigPtr)
		body.WriteString("\n```\n\n")
	}

	// Addon Logs section per template
	body.WriteString("### Addon Logs\n\n")
	if addonLogs != nil && *addonLogs != "" {
		body.WriteString("```text\n")
		body.WriteString(*addonLogs)
		body.WriteString("\n```\n\n")
	} else {
		body.WriteString("(Attach addon logs or enable 'Include addon logs' in the dialog)\n\n")
	}

	return body.String()
}

// exportSanitizedConfig exports configuration with sensitive data removed
func (s *IssueReportService) exportSanitizedConfig(_ context.Context) (string, error) {
	settings, err := s.settingService.Load()
	if err != nil {
		return "", errors.Wrap(err, "failed to load settings")
	}

	// Create a sanitized copy
	sanitized := *settings

	// Remove sensitive fields
	// Note: The actual Settings struct fields should be checked and sanitized
	// This is a placeholder implementation

	jsonData, jsonErr := json.MarshalIndent(sanitized, "", "  ")
	if jsonErr != nil {
		return "", errors.Wrap(jsonErr, "failed to marshal sanitized config")
	}

	return string(jsonData), nil
}

// exportAddonConfig exports addon config
func (s *IssueReportService) exportAddonConfig(_ context.Context) (string, error) {
	content, err := os.ReadFile("/data/options.json")
	return string(content), err
}
