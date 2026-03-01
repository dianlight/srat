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
	"github.com/dianlight/tlog/sanitizer"
	"github.com/google/go-github/v84/github"
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
	params.Set("title", request.ProblemType.IssueTitle+" "+request.Title)

	attachs := map[string]*string{}

	// Include sanitized config if requested
	if request.IncludeSRATConfig {
		sanitizedConfig, err := s.exportSanitizedConfig(ctx)
		if err != nil {
			tlog.WarnContext(ctx, "Failed to export sanitized config", "error", err)
		} else {
			maskedSRATConfig := sanitizer.MaskNestedValue(sanitizedConfig, "").(string) // Assuming sanitizedConfig is a JSON string; adjust as needed
			attachs["srat_config"] = &maskedSRATConfig
		}
	}

	// Include addon config if requested
	if request.IncludeAddonConfig {
		addonConfig, err := s.exportAddonConfig(ctx)
		if err != nil {
			tlog.WarnContext(ctx, "Failed to export addon config", "error", err)
		} else {
			maskedAddonConfig := sanitizer.MaskNestedValue(addonConfig, "").(string) // Assuming addonConfig is a JSON string; adjust as needed
			attachs["addon_config"] = &maskedAddonConfig
		}
	}

	// Include addon logs if requested
	if request.IncludeAddonLogs {
		addonLogs, err := s.addonService.GetLatestLogs(ctx)
		if err != nil {
			tlog.WarnContext(ctx, "Failed to export addon logs", "error", err)
		} else {
			maskedAddonLogs := sanitizer.MaskNestedValue(addonLogs, "").(string) // Assuming addonLogs is a string; adjust as needed
			attachs["logs"] = &maskedAddonLogs
		}
	}

	var consoleErrors strings.Builder
	if request.IncludeConsoleErrors {
		if len(request.ConsoleErrors) > 0 {
			consoleErrors.WriteString("- **Console Errors**:\n```javascript\n")
			for _, err := range request.ConsoleErrors {
				fmt.Fprintf(&consoleErrors, "%s\n", sanitizer.MaskNestedValue(err, ""))
			}
			consoleErrors.WriteString("```\n")
		}
		attachs["console"] = new(consoleErrors.String())
	}

	/* if request.IncludeDatabaseDump {
		attachs["database"] = new("\n\n(Database dump is not included in the issue report due to size constraints. Please attach a database dump manually if relevant.)")
	} */

	switch request.ProblemType.Template {
	case "bug_report.yaml":
		params.Set("template", "bug_report.yaml")

		params.Set("problem_type", request.ProblemType.Description)
		params.Set("version", config.Version)
		params.Set("arch", runtime.GOARCH)
		params.Set("os", runtime.GOOS)
		params.Set("description", request.Description)
		params.Set("reprod", request.ReproducingSteps)
		if attachs["srat_config"] != nil {
			params.Set("srat_config", *attachs["srat_config"])
		}
		if attachs["addon_config"] != nil {
			params.Set("addon_config", *attachs["addon_config"])
		}
		if attachs["logs"] != nil {
			params.Set("logs", *attachs["logs"])
		}
		if attachs["console"] != nil {
			params.Set("console", *attachs["console"])
		}
		/* if attachs["database"] != nil {
			params.Set("database", *attachs["database"])
		} */
	case "BUG-REPORT.yml":
		params.Set("template", "BUG-REPORT.yml")

		params.Set("addon", "SambaNas2")
		params.Set("description", request.Description)
		params.Set("reprod", request.ReproducingSteps)
		if attachs["logs"] != nil {
			params.Set("logs", *attachs["logs"])
		}
		if attachs["addon_config"] != nil {
			params.Set("config", *attachs["addon_config"])
		}
		params.Set("browsers", runtime.GOARCH)
		params.Set("os", runtime.GOOS)
	default:
		tlog.WarnContext(ctx, "Unknown issue template specified", "template", request.ProblemType.Template)
		return nil, errors.New("unknown issue template specified")
	}

	response.GitHubURL = fmt.Sprintf("%s/issues/new?%s", request.ProblemType.RepositoryUrl, params.Encode())
	files := map[github.GistFilename]github.GistFile{}

	for len(params.Encode()) > 7000 {
		tlog.WarnContext(ctx, "Generated GitHub issue URL is too long", "length", len(params.Encode()), "max_length", 7000)

		/* if attachs["database"] != nil {
			files["database_dump.txt"] = github.GistFile{
				Content: attachs["database"],
			}
			params.Del("database")
			continue
		} */
		if attachs["logs"] != nil {
			files["logs.txt"] = github.GistFile{
				Content: attachs["logs"],
			}
			params.Del("logs")
			continue
		}
		if attachs["console"] != nil {
			files["console.txt"] = github.GistFile{
				Content: attachs["console"],
			}
			params.Del("console")
			continue
		}
		tlog.WarnContext(ctx, "Unable to reduce GitHub issue URL length further by offloading attachments to Gist", "current_length", len(params.Encode()))
		break
	}

	if len(files) > 0 {
		gist, _, err := s.gh.Gists.Create(s.ctx, &github.Gist{
			Description: new("Issue report attachments for URL length constraints"),
			Public:      new(true),
			Files:       files,
		})

		if err == nil {
			for filename := range files {
				tlog.InfoContext(ctx, "Created GitHub Gist for issue report attachment", "filename", filename, "gist_id", gist.GetID(), "raw_url", *gist.GetFiles()[filename].RawURL)
				params.Set(strings.TrimSuffix(string(filename), ".txt"), *gist.GetFiles()[filename].RawURL)
			}
		} else {
			tlog.WarnContext(ctx, "Failed to create GitHub Gist for issue report attachments", "error", err)
			for filename := range files {
				tlog.WarnContext(ctx, "Attachment omitted due to URL length constraints", "filename", filename)
				params.Set(strings.TrimSuffix(string(filename), ".txt"), fmt.Sprintf("(Attachment omitted due to URL length constraints: %s)", filename))
			}
		}
	}
	response.GitHubURL = fmt.Sprintf("%s/issues/new?%s", request.ProblemType.RepositoryUrl, params.Encode())

	return response, nil
}

// exportSanitizedConfig exports configuration with sensitive data removed
func (s *IssueReportService) exportSanitizedConfig(_ context.Context) (string, error) {
	settings, err := s.settingService.Load()
	if err != nil {
		return "", errors.Wrap(err, "failed to load settings")
	}

	// Create a sanitized copy
	sanitized := sanitizer.MaskNestedValue(*settings, "")

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
