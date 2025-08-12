package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/srat/tlog"
	"go.uber.org/fx"
)

// IssueAPI handles API requests for issues.
type IssueAPI struct {
	service *service.IssueService
}

// NewIssueAPI creates a new issue API.
func NewIssueAPI(service *service.IssueService) *IssueAPI {
	return &IssueAPI{service: service}
}

// GetIssuesInput defines the input for getting issues.
type GetIssuesInput struct{}

// GetIssuesOutput defines the output for getting issues.
type GetIssuesOutput struct {
	Body []*dto.Issue
}

// CreateIssueInput defines the input for creating an issue.
type CreateIssueInput struct {
	Body dto.Issue
}

// CreateIssueOutput defines the output for creating an issue.
type CreateIssueOutput struct {
	Body *dto.Issue
}

// ResolveIssueInput defines the input for resolving an issue.
type ResolveIssueInput struct {
	ID uint `path:"id"`
}

// ResolveIssueOutput defines the output for resolving an issue.
type ResolveIssueOutput struct {
	Body struct{}
}

// UpdateIssueInput defines the input for updating an issue.
type UpdateIssueInput struct {
	ID   uint `path:"id"`
	Body dto.Issue
}

// UpdateIssueOutput defines the output for updating an issue.
type UpdateIssueOutput struct {
	Body *dto.Issue
}

// Register registers the issue API endpoints.
func (a *IssueAPI) RegisterIssueHandler(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-issues",
		Summary:     "Get all open issues",
		Method:      http.MethodGet,
		Path:        "/issues",
		Tags:        []string{"Issues"},
	}, func(ctx context.Context, input *GetIssuesInput) (*GetIssuesOutput, error) {
		issues, err := a.service.FindOpen()
		if err != nil {
			tlog.Error("failed to get issues", err)
			return nil, huma.Error500InternalServerError("failed to get issues", err)
		}
		return &GetIssuesOutput{Body: issues}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "create-issue",
		Summary:     "Create a new issue",
		Method:      http.MethodPost,
		Path:        "/issues",
		Tags:        []string{"Issues"},
	}, func(ctx context.Context, input *CreateIssueInput) (*CreateIssueOutput, error) {
		newIssue, err := a.service.Create(&input.Body)
		if err != nil {
			tlog.Error("failed to create issue", err)
			return nil, huma.Error500InternalServerError("failed to create issue", err)
		}
		return &CreateIssueOutput{Body: newIssue}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "resolve-issue",
		Summary:     "Resolve an issue",
		Method:      http.MethodDelete,
		Path:        "/issues/{id}",
		Tags:        []string{"Issues"},
	}, func(ctx context.Context, input *ResolveIssueInput) (*ResolveIssueOutput, error) {
		if err := a.service.Resolve(input.ID); err != nil {
			tlog.Error("failed to resolve issue", err)
			return nil, huma.Error500InternalServerError("failed to resolve issue", err)
		}
		return &ResolveIssueOutput{}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-issue",
		Summary:     "Update an issue",
		Method:      http.MethodPut,
		Path:        "/issues/{id}",
		Tags:        []string{"Issues"},
	}, func(ctx context.Context, input *UpdateIssueInput) (*UpdateIssueOutput, error) {
		input.Body.ID = input.ID
		updatedIssue, err := a.service.Update(&input.Body)
		if err != nil {
			tlog.Error("failed to update issue", err)
			return nil, huma.Error500InternalServerError("failed to update issue", err)
		}
		return &UpdateIssueOutput{Body: updatedIssue}, nil
	})
}

// Fx provides the issue API as a dependency.
var Fx = fx.Provide(NewIssueAPI)
