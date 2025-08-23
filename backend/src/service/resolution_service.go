package service

import (
	"context"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/resolution"
	"github.com/oapi-codegen/runtime/types"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

type ResolutionServiceParams struct {
	fx.In
	ApiContext       context.Context
	ApiContextCancel context.CancelFunc
	ResolutionClient resolution.ClientWithResponsesInterface
	State            *dto.ContextState
}

type ResolutionServiceInterface interface {
	CreateIssue(issue dto.ResolutionIssue) error
	DeleteIssue(uuid types.UUID) error
}

type ResolutionService struct {
	apiContext       context.Context
	resolutionClient resolution.ClientWithResponsesInterface
	state            *dto.ContextState
}

func NewResolutionService(in ResolutionServiceParams) ResolutionServiceInterface {
	p := &ResolutionService{}
	p.apiContext = in.ApiContext
	p.resolutionClient = in.ResolutionClient
	p.state = in.State
	return p
}

func (s *ResolutionService) CreateIssue(issue dto.ResolutionIssue) error {
	if s.state.SupervisorURL == "demo" || s.state.HACoreReady == false {
		return nil
	}
	/*
		resp, err := s.resolutionClient.CreateIssueWithResponse(
			s.apiContext,
			resolution.CreateIssueJSONBody{
				Type:        issue.Type,
				Context:     issue.Context,
				Reference:   issue.Reference,
				Suggestion:  issue.Suggestion,
				Unhealthy:   issue.Unhealthy,
				Unsupported: issue.Unsupported,
			},
		)

		if err != nil {
			return errors.Errorf("Error creating issue: %w", err)
		}

		if resp.StatusCode() != 200 {
			return errors.Errorf("Error creating issue: %d %s", resp.StatusCode(), string(resp.Body))
		}
	*/
	return nil
}

func (s *ResolutionService) DeleteIssue(uuid types.UUID) error {
	if s.state.SupervisorURL == "demo" || s.state.HACoreReady == false {
		return nil
	}

	resp, err := s.resolutionClient.DismissIssueWithResponse(s.apiContext, uuid)

	if err != nil {
		return errors.Errorf("Error deleting issue: %w", err)
	}

	if resp.StatusCode() != 200 {
		return errors.Errorf("Error deleting issue: %d %s", resp.StatusCode(), string(resp.Body))
	}

	return nil
}
