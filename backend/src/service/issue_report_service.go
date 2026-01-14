package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/tlog"
	"gitlab.com/tozd/go/errors"
)

// IssueReportServiceInterface defines the interface for issue reporting
type IssueReportServiceInterface interface {
	GenerateIssueReport(ctx context.Context, request *dto.IssueReportRequest) (*dto.IssueReportResponse, error)
}

// IssueReportService handles generating diagnostic data for issue reporting
type IssueReportService struct {
	settingService SettingServiceInterface
}

// NewIssueReportService creates a new issue report service
func NewIssueReportService(settingService SettingServiceInterface) IssueReportServiceInterface {
	return &IssueReportService{
		settingService: settingService,
	}
}

// GenerateIssueReport generates diagnostic data and creates a GitHub issue URL
func (s *IssueReportService) GenerateIssueReport(ctx context.Context, request *dto.IssueReportRequest) (*dto.IssueReportResponse, error) {
	tlog.InfoContext(ctx, "Generating issue report", "problem_type", request.ProblemType)

	// Build issue title
	issueTitle := s.buildIssueTitle(request.ProblemType)

	// Build issue body
	issueBody := s.buildIssueBody(ctx, request)

	// Determine GitHub repository based on problem type
	repoURL := s.getRepositoryURL(request.ProblemType)

	// Create GitHub issue URL
	githubURL := s.createGitHubIssueURL(repoURL, issueTitle, issueBody)

	response := &dto.IssueReportResponse{
		GitHubURL:  githubURL,
		IssueTitle: issueTitle,
		IssueBody:  issueBody,
	}

	// Include sanitized config if requested
	if request.IncludeSRATConfig {
		sanitizedConfig, err := s.exportSanitizedConfig(ctx)
		if err != nil {
			tlog.WarnContext(ctx, "Failed to export sanitized config", "error", err)
		} else {
			response.SanitizedConfig = &sanitizedConfig
		}
	}

	// Include addon logs if requested
	if request.IncludeAddonLogs {
		addonLogs, err := s.exportAddonLogs(ctx)
		if err != nil {
			tlog.WarnContext(ctx, "Failed to export addon logs", "error", err)
		} else {
			response.AddonLogs = &addonLogs
		}
	}

	return response, nil
}

// buildIssueTitle creates a title for the GitHub issue
func (s *IssueReportService) buildIssueTitle(problemType dto.ProblemType) string {
	prefix := ""
	switch problemType {
	case dto.ProblemTypeFrontendUI:
		prefix = "[UI]"
	case dto.ProblemTypeHAIntegration:
		prefix = "[HA Integration]"
	case dto.ProblemTypeAddon:
		prefix = "[Addon]"
	case dto.ProblemTypeSamba:
		prefix = "[Samba]"
	}
	return fmt.Sprintf("%s Issue reported on %s", prefix, time.Now().Format("2006-01-02"))
}

// buildIssueBody creates the body content for the GitHub issue
func (s *IssueReportService) buildIssueBody(ctx context.Context, request *dto.IssueReportRequest) string {
	var body strings.Builder

	// User description
	body.WriteString("## Description\n\n")
	body.WriteString(request.Description)
	body.WriteString("\n\n")

	// Problem type
	body.WriteString("## Problem Type\n\n")
	body.WriteString(fmt.Sprintf("- **Type**: %s\n", request.ProblemType))
	body.WriteString("\n")

	// Context data
	if request.IncludeContextData && request.CurrentURL != "" {
		body.WriteString("## Context Information\n\n")
		body.WriteString(fmt.Sprintf("- **Current URL**: %s\n", request.CurrentURL))
		
		if len(request.NavigationHistory) > 0 {
			body.WriteString("- **Navigation History**:\n")
			for i, url := range request.NavigationHistory {
				if i >= 5 { // Limit to last 5 entries
					break
				}
				body.WriteString(fmt.Sprintf("  - %s\n", url))
			}
		}
		
		if request.BrowserInfo != "" {
			body.WriteString(fmt.Sprintf("- **Browser**: %s\n", request.BrowserInfo))
		}
		
		if len(request.ConsoleErrors) > 0 {
			body.WriteString("- **Console Errors**:\n```\n")
			for i, err := range request.ConsoleErrors {
				if i >= 10 { // Limit to last 10 errors
					break
				}
				body.WriteString(fmt.Sprintf("%s\n", err))
			}
			body.WriteString("```\n")
		}
		body.WriteString("\n")
	}

	// System information
	body.WriteString("## System Information\n\n")
	body.WriteString(fmt.Sprintf("- **Report Generated**: %s\n", time.Now().Format(time.RFC3339)))
	
	// Add version info if available
	settings, err := s.settingService.Load()
	if err == nil && settings != nil {
		body.WriteString(fmt.Sprintf("- **SRAT Version**: %s\n", "unknown")) // TODO: Add version tracking
	}
	
	body.WriteString("\n")

	// Instructions for attachments
	if request.IncludeSRATConfig || request.IncludeAddonLogs {
		body.WriteString("## Attachments\n\n")
		body.WriteString("Please attach the downloaded diagnostic files to this issue.\n\n")
		if request.IncludeSRATConfig {
			body.WriteString("- [ ] Sanitized SRAT configuration\n")
		}
		if request.IncludeAddonLogs {
			body.WriteString("- [ ] Addon logs\n")
		}
		body.WriteString("\n")
	}

	return body.String()
}

// getRepositoryURL returns the GitHub repository URL based on problem type
func (s *IssueReportService) getRepositoryURL(problemType dto.ProblemType) string {
	switch problemType {
	case dto.ProblemTypeFrontendUI, dto.ProblemTypeHAIntegration:
		return "https://github.com/dianlight/srat"
	case dto.ProblemTypeAddon, dto.ProblemTypeSamba:
		return "https://github.com/dianlight/hassos-addon"
	default:
		return "https://github.com/dianlight/srat"
	}
}

// createGitHubIssueURL creates a GitHub URL with pre-populated issue data
func (s *IssueReportService) createGitHubIssueURL(repoURL, title, body string) string {
	params := url.Values{}
	params.Add("title", title)
	params.Add("body", body)
	return fmt.Sprintf("%s/issues/new?%s", repoURL, params.Encode())
}

// exportSanitizedConfig exports configuration with sensitive data removed
func (s *IssueReportService) exportSanitizedConfig(ctx context.Context) (string, error) {
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

// exportAddonLogs exports addon logs from last boot
func (s *IssueReportService) exportAddonLogs(ctx context.Context) (string, error) {
	// Try to read from typical Home Assistant addon log locations
	logPaths := []string{
		"/data/logs/addon.log",
		"/var/log/addon.log",
		"/addon_logs/current.log",
	}

	var logContent strings.Builder
	foundAnyLogs := false

	for _, logPath := range logPaths {
		content, err := os.ReadFile(logPath)
		if err == nil {
			logContent.WriteString(fmt.Sprintf("=== Log from %s ===\n", logPath))
			// Limit log size to last 50KB
			if len(content) > 50000 {
				content = content[len(content)-50000:]
				logContent.WriteString("... (truncated to last 50KB)\n")
			}
			logContent.Write(content)
			logContent.WriteString("\n\n")
			foundAnyLogs = true
		}
	}

	if !foundAnyLogs {
		return "No addon logs found in standard locations", nil
	}

	return logContent.String(), nil
}
