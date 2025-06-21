package service

import (
	"context"
	"log/slog"
	"net/http"
	"sync"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/addons"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

// AddonsServiceInterface defines the contract for addon-related operations.
type AddonsServiceInterface interface {
	// CheckProtectedMode checks if a given Home Assistant addon is marked as protected.
	// It returns true if the addon is protected, false otherwise.
	// An error is returned if the check cannot be performed (e.g., API error, addon not found).
	CheckProtectedMode() (bool, error)
}

// AddonsService provides methods to interact with Home Assistant addons.
type AddonsService struct {
	ctx          context.Context
	apictx       *dto.ContextState // Context state for the API, can be used for logging or passing additional information.
	addonsClient addons.ClientWithResponsesInterface
}

// AddonsServiceParams holds the dependencies for AddonsService.
type AddonsServiceParams struct {
	fx.In
	Ctx          context.Context
	Apictx       *dto.ContextState
	AddonsClient addons.ClientWithResponsesInterface `optional:"true"`
}

// NewAddonsService creates a new instance of AddonsService.
func NewAddonsService(params AddonsServiceParams) AddonsServiceInterface {
	if params.AddonsClient == nil {
		slog.WarnContext(params.Ctx, "AddonsClient is not available for AddonsService. Operations requiring it will fail.")
	}
	p := &AddonsService{
		ctx:          params.Ctx,
		apictx:       params.Apictx,
		addonsClient: params.AddonsClient,
	}

	params.Ctx.Value("wg").(*sync.WaitGroup).Add(1)
	go func() {
		defer params.Ctx.Value("wg").(*sync.WaitGroup).Done()
		p.apictx.ProtectedMode, _ = p.CheckProtectedMode()
	}()
	return p

}

// CheckProtectedMode implements the AddonsServiceInterface.
func (s *AddonsService) CheckProtectedMode() (bool, error) {
	if s.addonsClient == nil {
		return false, errors.New("addons client is not initialized")
	}

	resp, err := s.addonsClient.GetSelfAddonInfoWithResponse(s.ctx)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get addon info for '%s'", "self")
	}

	if resp.StatusCode() != http.StatusOK {
		return false, errors.Errorf("failed to get addon info for '%s': status %d, body: %s", "self", resp.StatusCode(), string(resp.Body))
	}

	if resp.JSON200 == nil || resp.JSON200.Data.Protected == nil {
		// If protected status is not explicitly provided, assume not protected or data is incomplete.
		return false, errors.Errorf("protected status not available or data incomplete for addon '%s'", "self")
	}

	return *resp.JSON200.Data.Protected, nil
}
