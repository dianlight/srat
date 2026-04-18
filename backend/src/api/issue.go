package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/tlog"
	"go.uber.org/fx"
)

// IssueAPI handles API requests for issues.
type IssueAPI struct {
	reportService   service.IssueReportServiceInterface
	templateService service.IssueTemplateServiceInterface
}

// NewIssueAPI creates a new issue API.
func NewIssueAPI(
	reportService service.IssueReportServiceInterface,
	templateService service.IssueTemplateServiceInterface,
) *IssueAPI {
	return &IssueAPI{
		reportService:   reportService,
		templateService: templateService,
	}
}

// ResolveIssueInput defines the input for resolving an issue.
type ResolveIssueInput struct {
	ID uint `path:"id"`
}

// ResolveIssueOutput defines the output for resolving an issue.
// ResolveIssueOutput defines the output for resolving an issue.
// Returning a struct with a Status field allows explicit 204 No Content.
type ResolveIssueOutput struct {
	Status int `json:"-"`
}

// Register registers the issue API endpoints.
func (a *IssueAPI) RegisterIssueHandler(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "generate-issue-report",
		Summary:     "Generate diagnostic data for GitHub issue reporting",
		Method:      http.MethodPost,
		Path:        "/issues/report",
		Tags:        []string{"Issues"},
	}, func(ctx context.Context, input *struct{ Body dto.IssueReportRequest }) (*struct{ Body dto.IssueReportResponse }, error) {
		report, err := a.reportService.GenerateIssueReport(ctx, &input.Body)
		if err != nil {
			tlog.ErrorContext(ctx, "failed to generate issue report", err)
			return nil, huma.Error500InternalServerError("failed to generate issue report", err)
		}
		return &struct{ Body dto.IssueReportResponse }{Body: *report}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-issue-template",
		Summary:     "Get GitHub issue template",
		Method:      http.MethodGet,
		Path:        "/issues/template",
		Tags:        []string{"Issues"},
	}, func(ctx context.Context, input *struct{}) (*struct{ Body dto.IssueTemplateResponse }, error) {
		template, err := a.templateService.GetTemplate()
		if err != nil {
			tlog.WarnContext(ctx, "failed to fetch issue template", "error", errors.Unwrap(err))
			errMsg := err.Error()
			return &struct{ Body dto.IssueTemplateResponse }{
				Body: dto.IssueTemplateResponse{
					Template: nil,
					Error:    &errMsg,
				},
			}, nil
		}
		return &struct{ Body dto.IssueTemplateResponse }{
			Body: dto.IssueTemplateResponse{
				Template: template,
				Error:    nil,
			},
		}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "repair",
		Summary:     "Repair Message Object for Home Assistant to trigger repairs from the frontend",
		Method:      http.MethodTrace,
		Path:        "/repairMessage",
		Tags:        []string{"Issues", "internal"},
	}, func(ctx context.Context, input *struct{}) (*struct {
		Body dto.RepairCommandMessage
	}, error) {
		return nil, huma.Error500InternalServerError("failed to repair issue", nil)
	})
}

// Fx provides the issue API as a dependency.
var Fx = fx.Provide(NewIssueAPI)
