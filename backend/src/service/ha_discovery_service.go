package service

import (
	"context"
	"log/slog"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/discovery"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

// HaDiscoveryServiceInterface manages Supervisor discovery registration
// so that the SRAT custom component can auto-discover this addon.
type HaDiscoveryServiceInterface interface {
	// RegisterDiscovery posts a discovery message to the Supervisor API.
	RegisterDiscovery(ctx context.Context) error
	// UnregisterDiscovery removes the discovery message from the Supervisor API.
	UnregisterDiscovery(ctx context.Context) error
}

type haDiscoveryService struct {
	ctx             context.Context
	state           *dto.ContextState
	addonsService   AddonsServiceInterface
	discoveryClient discovery.ClientWithResponsesInterface
	discoveryUUID   *openapi_types.UUID // UUID returned by the Supervisor after registration
}

type HaDiscoveryServiceParams struct {
	fx.In
	Ctx             context.Context
	State           *dto.ContextState
	AddonsService   AddonsServiceInterface
	DiscoveryClient discovery.ClientWithResponsesInterface `optional:"true"`
}

func NewHaDiscoveryService(lc fx.Lifecycle, params HaDiscoveryServiceParams) HaDiscoveryServiceInterface {
	svc := &haDiscoveryService{
		ctx:             params.Ctx,
		state:           params.State,
		addonsService:   params.AddonsService,
		discoveryClient: params.DiscoveryClient,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := svc.RegisterDiscovery(ctx); err != nil {
				// Discovery registration failure is non-fatal — log and continue.
				slog.WarnContext(ctx, "Failed to register Supervisor discovery (addon may not have 'discovery' in config.yaml)", "err", err)
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if err := svc.UnregisterDiscovery(ctx); err != nil {
				slog.WarnContext(ctx, "Failed to unregister Supervisor discovery", "err", err)
			}
			return nil
		},
	})

	return svc
}

func (s *haDiscoveryService) RegisterDiscovery(ctx context.Context) error {
	if s.state.SupervisorURL == "" || s.state.SupervisorURL == "demo" {
		slog.DebugContext(ctx, "Skipping discovery registration — not running in Supervisor mode")
		return nil
	}

	if s.state.SupervisorToken == "" {
		slog.DebugContext(ctx, "Skipping discovery registration — no Supervisor token available")
		return nil
	}

	if s.discoveryClient == nil {
		return errors.New("discovery client is not initialized")
	}

	// Get the addon's hostname for the discovery config.
	// In HA Supervisor, addons are accessible via their container hostname.
	addonInfo, errInfo := s.addonsService.GetInfo(ctx)
	if errInfo != nil {
		return errors.Wrap(errInfo, "failed to get addon info for discovery")
	}

	host := ""
	if addonInfo != nil && addonInfo.Hostname != nil {
		host = *addonInfo.Hostname
	}
	if host == "" {
		// Fallback: use the addon IP address from context state
		host = s.state.AddonIpAddress
	}
	if host == "" {
		return errors.New("cannot determine addon host for discovery registration")
	}

	payload := discovery.CreateDiscoveryServiceJSONRequestBody{
		Service: "srat",
		Config: map[string]any{
			"host": host,
			"port": s.state.ServerPort,
		},
	}

	resp, err := s.discoveryClient.CreateDiscoveryServiceWithResponse(ctx, payload)
	if err != nil {
		return errors.Wrap(err, "failed to send discovery request")
	}

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return errors.Errorf("discovery registration failed: HTTP %d — %s", resp.StatusCode(), string(resp.Body))
	}

	if resp.JSON200 == nil || resp.JSON200.Data == nil || resp.JSON200.Data.Uuid == nil {
		return errors.New("discovery response missing data or uuid")
	}

	s.discoveryUUID = resp.JSON200.Data.Uuid
	slog.InfoContext(ctx, "Registered Supervisor discovery for SRAT custom component", "uuid", s.discoveryUUID.String(), "host", host)

	return nil
}

func (s *haDiscoveryService) UnregisterDiscovery(ctx context.Context) error {
	if s.discoveryUUID == nil {
		return nil // nothing to unregister
	}

	if s.state.SupervisorURL == "" || s.state.SupervisorURL == "demo" {
		return nil
	}

	if s.discoveryClient == nil {
		return errors.New("discovery client is not initialized")
	}

	resp, err := s.discoveryClient.DeleteDiscoveryServiceWithResponse(ctx, *s.discoveryUUID)
	if err != nil {
		return errors.Wrap(err, "failed to send discovery delete request")
	}

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return errors.Errorf("discovery unregistration failed: HTTP %d — %s", resp.StatusCode(), string(resp.Body))
	}

	slog.InfoContext(ctx, "Unregistered Supervisor discovery for SRAT", "uuid", s.discoveryUUID.String())
	s.discoveryUUID = nil

	return nil
}
