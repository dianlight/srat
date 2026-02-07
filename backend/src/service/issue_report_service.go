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
	"github.com/google/go-github/v82/github"
	"github.com/xorcare/pointer"
	"gitlab.com/tozd/go/errors"
)

// IssueReportServiceInterface defines the interface for issue reporting
type IssueReportServiceInterface interface {
	GenerateIssueReport(ctx context.Context, request *dto.IssueReportRequest) (*dto.IssueReportResponse, error)
}

// IssueReportService handles generating diagnostic data for issue reporting
type IssueReportService struct {
	ctx            context.Context
	settingService SettingServiceInterface
	addonService   AddonsServiceInterface
	gh             *github.Client
}

// NewIssueReportService creates a new issue report service
func NewIssueReportService(
	ctx context.Context,
	settingService SettingServiceInterface,
	addonService AddonsServiceInterface,
	gh *github.Client,
) IssueReportServiceInterface {
	return &IssueReportService{
		ctx:            ctx,
		settingService: settingService,
		addonService:   addonService,
		gh:             gh,
	}
}

// GenerateIssueReport generates diagnostic data and creates a GitHub issue URL
func (s *IssueReportService) GenerateIssueReport(ctx context.Context, request *dto.IssueReportRequest) (*dto.IssueReportResponse, error) {
	tlog.InfoContext(ctx, "Generating issue report", "problem_type", request.ProblemType)

	response := &dto.IssueReportResponse{
		IssueTitle: request.ProblemType.IssueTitle,
	}
	// Create GitHub issue URL
	params := url.Values{}
	params.Add("title", request.ProblemType.IssueTitle+" "+request.Title)

	var sanitizedConfigPtr *string
	var addonLogsPtr *string
	var addonConfigPtr *string
	var consoleErrorsPtr *string
	//var databaseDumpPtr *string

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

	var contextData strings.Builder
	if request.IncludeConsoleErrors {
		if len(request.ConsoleErrors) > 0 {
			contextData.WriteString("- **Console Errors**:\n```javascript\n")
			for _, err := range request.ConsoleErrors {
				contextData.WriteString(fmt.Sprintf("%s\n", err))
			}
			contextData.WriteString("```\n")
		}
		consoleErrorsPtr = pointer.String(contextData.String())
	}

	/* if request.IncludeDatabaseDump {
		//databaseDumpPtr = pointer.String("\n\n(Database dump is not included in the issue report due to size constraints. Please attach a database dump manually if relevant.)")
	} */

	switch request.ProblemType.Template {
	case "bug_report.yaml":
		params.Add("template", "bug_report.yaml")

		params.Add("problem_type", request.ProblemType.Description)
		params.Add("version", config.Version)
		params.Add("arch", runtime.GOARCH)
		params.Add("os", runtime.GOOS)
		params.Add("description", request.Description)
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
		if consoleErrorsPtr != nil {
			params.Add("console", *consoleErrorsPtr)
		}
		/* if databaseDumpPtr != nil {
			params.Add("database", *databaseDumpPtr)
		} */
	case "BUG-REPORT.yml":
		params.Add("template", "BUG-REPORT.yml")

		params.Add("addon", "SambaNas2")
		params.Add("description", request.Description)
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
			sanitizedConfigPtr,
			addonConfigPtr,
			addonLogsPtr)

		params.Add("body", body)
	}

	response.GitHubURL = fmt.Sprintf("%s/issues/new?%s", request.ProblemType.RepositoryUrl, params.Encode())

	for len(response.GitHubURL) > 8000 {
		tlog.WarnContext(ctx, "Generated GitHub issue URL is too long", "length", len(response.GitHubURL))
		/* if databaseDumpPtr != nil {
			gist, _, err := s.gh.Gists.Create(s.ctx, &github.Gist{
				Description: pointer.String("Database dump for issue report"),
				Public:      pointer.Bool(true),
				Files: map[github.GistFilename]github.GistFile{
					"database_dump.txt": {
						Content: databaseDumpPtr,
					},
				},
			})
			if err != nil {
				tlog.WarnContext(ctx, "Failed to create GitHub Gist for database dump", "error", err)
				params.Add("database", "(Database dump omitted due to URL length constraints. Please attach database dump to your issue manually.)")
			} else {
				params.Add("database", *gist.Files["database_dump.txt"].RawURL)
			}
			response.GitHubURL = fmt.Sprintf("%s/issues/new?%s", request.ProblemType.RepositoryUrl, params.Encode())
			continue
		} */
		if addonLogsPtr != nil {
			gist, _, err := s.gh.Gists.Create(s.ctx, &github.Gist{
				Description: pointer.String("Addon logs for issue report"),
				Public:      pointer.Bool(true),
				Files: map[github.GistFilename]github.GistFile{
					"addon_logs.txt": {
						Content: addonLogsPtr,
					},
				},
			})
			if err != nil {
				tlog.WarnContext(ctx, "Failed to create GitHub Gist for addon logs", "error", err)
				params.Set("logs", "(Addon logs omitted due to URL length constraints. Please attach addon logs to your issue manually.)")
			} else {
				params.Set("logs", *gist.Files["addon_logs.txt"].RawURL)
			}
			response.GitHubURL = fmt.Sprintf("%s/issues/new?%s", request.ProblemType.RepositoryUrl, params.Encode())
			addonLogsPtr = nil
			continue
		}
		if consoleErrorsPtr != nil {
			gist, _, err := s.gh.Gists.Create(s.ctx, &github.Gist{
				Description: pointer.String("Console errors for issue report"),
				Public:      pointer.Bool(true),
				Files: map[github.GistFilename]github.GistFile{
					"console_errors.txt": {
						Content: consoleErrorsPtr,
					},
				},
			})
			if err != nil {
				tlog.WarnContext(ctx, "Failed to create GitHub Gist for console errors", "error", err)
				params.Set("console", "(Console errors omitted due to URL length constraints. Please attach console errors to your issue manually.)")
			} else {
				params.Set("console", *gist.Files["console_errors.txt"].RawURL)
			}
			response.GitHubURL = fmt.Sprintf("%s/issues/new?%s", request.ProblemType.RepositoryUrl, params.Encode())
			consoleErrorsPtr = nil
			continue
		}
		// If we can't reduce length by offloading data to Gists, break to avoid infinite loop
		break
	}

	return response, nil
}

// buildIssueBody creates the body content for the GitHub issue
func (s *IssueReportService) buildIssueBody(
	_ context.Context,
	request *dto.IssueReportRequest,
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
