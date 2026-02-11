package service

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/urlutil"
	"github.com/dianlight/tlog"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"gopkg.in/yaml.v3"
)

const (
	issueTemplateURL   = "https://raw.githubusercontent.com/dianlight/srat/main/.github/ISSUE_TEMPLATE/bug_report.yaml"
	issueTemplateFetch = 30 * time.Second
)

// IssueTemplateServiceInterface defines the interface for issue template management
type IssueTemplateServiceInterface interface {
	GetTemplate() (*dto.IssueTemplate, error)
}

// IssueTemplateService handles fetching and parsing GitHub issue templates
type IssueTemplateService struct {
	httpClient *http.Client
	template   *dto.IssueTemplate
}

// NewIssueTemplateService creates a new issue template service
func NewIssueTemplateService(lc fx.Lifecycle) IssueTemplateServiceInterface {
	ret := &IssueTemplateService{
		httpClient: &http.Client{
			Timeout: issueTemplateFetch,
		},
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			tlog.DebugContext(ctx, "IssueTemplateService started")
			var err error
			if ret.template, err = ret.fetchIssueTemplate(ctx); err != nil {
				tlog.ErrorContext(ctx, "Failed to fetch issue template on start", "error", err)
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			tlog.DebugContext(ctx, "IssueTemplateService stopped")
			return nil
		},
	})
	return ret
}

func (s *IssueTemplateService) GetTemplate() (*dto.IssueTemplate, error) {
	if s.template == nil {
		return nil, errors.New("issue template not available")
	}
	return s.template, nil
}

// fetchIssueTemplate fetches and parses the GitHub issue template
func (s *IssueTemplateService) fetchIssueTemplate(ctx context.Context) (*dto.IssueTemplate, error) {
	tlog.InfoContext(ctx, "Fetching issue template from GitHub")

	if err := urlutil.ValidateURL(issueTemplateURL, []string{"raw.githubusercontent.com"}); err != nil {
		return nil, errors.Wrap(err, "untrusted issue template URL")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, issueTemplateURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	resp, err := s.httpClient.Do(req) // #nosec G704
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch issue template")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	var template dto.IssueTemplate
	if err := yaml.Unmarshal(body, &template); err != nil {
		return nil, errors.Wrap(err, "failed to parse YAML template")
	}

	tlog.InfoContext(ctx, "Successfully fetched and parsed issue template", "name", template.Name)
	return &template, nil
}
