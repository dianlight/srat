package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/tlog"
)

// ProblemAPI handles API requests for unified problems.
type ProblemAPI struct {
	service service.ProblemServiceInterface
}

// NewProblemAPI creates a new problem API.
func NewProblemAPI(service service.ProblemServiceInterface) *ProblemAPI {
	return &ProblemAPI{service: service}
}

type GetProblemsOutput struct {
	Body []*dto.Problem
}

type GetProblemInput struct {
	ProblemKey string `path:"problemKey"`
}

type GetProblemOutput struct {
	Body *dto.Problem
}

type UpsertProblemInput struct {
	Body dto.Problem
}

type UpsertProblemOutput struct {
	Body *dto.Problem
}

type UpsertProblemByKeyInput struct {
	ProblemKey string      `path:"problemKey"`
	Body       dto.Problem `json:"body"`
}

type ExecuteProblemActionInput struct {
	ProblemKey string `path:"problemKey"`
	ActionKey  string `path:"actionKey"`
}

type ExecuteProblemActionOutput struct {
	Body *dto.Problem
}

type DismissProblemInput struct {
	ProblemKey string `path:"problemKey"`
}

type DismissProblemOutput struct {
	Status int `json:"-"`
}

// RegisterProblemHandler registers problem endpoints.
func (a *ProblemAPI) RegisterProblemHandler(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-problems",
		Summary:     "Get all open problems",
		Method:      http.MethodGet,
		Path:        "/problems",
		Tags:        []string{"Problems"},
	}, func(ctx context.Context, input *struct{}) (*GetProblemsOutput, error) {
		items, err := a.service.List()
		if err != nil {
			tlog.ErrorContext(ctx, "failed to list problems", "error", err)
			return nil, huma.Error500InternalServerError("failed to list problems", err)
		}
		return &GetProblemsOutput{Body: items}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-problem",
		Summary:     "Get a single problem by key",
		Method:      http.MethodGet,
		Path:        "/problems/{problemKey}",
		Tags:        []string{"Problems"},
	}, func(ctx context.Context, input *GetProblemInput) (*GetProblemOutput, error) {
		item, err := a.service.Get(input.ProblemKey)
		if err != nil {
			tlog.ErrorContext(ctx, "failed to get problem", "problem_key", input.ProblemKey, "error", err)
			return nil, huma.Error500InternalServerError("failed to get problem", err)
		}
		return &GetProblemOutput{Body: item}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "upsert-problem",
		Summary:     "Create or update a problem",
		Method:      http.MethodPost,
		Path:        "/problems",
		Tags:        []string{"Problems"},
	}, func(ctx context.Context, input *UpsertProblemInput) (*UpsertProblemOutput, error) {
		item, err := a.service.Upsert(&input.Body)
		if err != nil {
			tlog.ErrorContext(ctx, "failed to upsert problem", "error", err)
			return nil, huma.Error500InternalServerError("failed to upsert problem", err)
		}
		return &UpsertProblemOutput{Body: item}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "dismiss-problem",
		Summary:     "Dismiss a problem by key",
		Method:      http.MethodDelete,
		Path:        "/problems/{problemKey}",
		Tags:        []string{"Problems"},
	}, func(ctx context.Context, input *DismissProblemInput) (*DismissProblemOutput, error) {
		if err := a.service.Dismiss(input.ProblemKey); err != nil {
			tlog.ErrorContext(ctx, "failed to dismiss problem", "problem_key", input.ProblemKey, "error", err)
			return nil, huma.Error500InternalServerError("failed to dismiss problem", err)
		}
		return &DismissProblemOutput{Status: http.StatusNoContent}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "upsert-problem-by-key",
		Summary:     "Create or update a problem by key",
		Method:      http.MethodPut,
		Path:        "/problems/{problemKey}",
		Tags:        []string{"Problems"},
	}, func(ctx context.Context, input *UpsertProblemByKeyInput) (*UpsertProblemOutput, error) {
		payload := input.Body
		payload.ProblemKey = input.ProblemKey
		item, err := a.service.Upsert(&payload)
		if err != nil {
			tlog.ErrorContext(ctx, "failed to upsert problem by key", "problem_key", input.ProblemKey, "error", err)
			return nil, huma.Error500InternalServerError("failed to upsert problem by key", err)
		}
		return &UpsertProblemOutput{Body: item}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "execute-problem-action",
		Summary:     "Execute an action for a problem",
		Method:      http.MethodPost,
		Path:        "/problems/{problemKey}/actions/{actionKey}",
		Tags:        []string{"Problems"},
	}, func(ctx context.Context, input *ExecuteProblemActionInput) (*ExecuteProblemActionOutput, error) {
		problem, err := a.service.Get(input.ProblemKey)
		if err != nil {
			tlog.ErrorContext(ctx, "failed to get problem before action execution", "problem_key", input.ProblemKey, "error", err)
			return nil, huma.Error500InternalServerError("failed to get problem", err)
		}

		actionFound := false
		for _, action := range problem.Actions {
			if action.Key == input.ActionKey {
				actionFound = true
				break
			}
		}
		if !actionFound {
			return nil, huma.Error404NotFound("problem action not found", nil)
		}

		updated, err := a.service.ApplyLifecycle(
			input.ProblemKey,
			dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSFIXED,
			nil,
		)
		if err != nil {
			tlog.ErrorContext(ctx, "failed to apply lifecycle while executing problem action", "problem_key", input.ProblemKey, "action_key", input.ActionKey, "error", err)
			return nil, huma.Error500InternalServerError("failed to execute problem action", err)
		}

		return &ExecuteProblemActionOutput{Body: updated}, nil
	})
}
