package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/dianlight/srat/dto"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

// DiscoveryServiceInterface manages Supervisor discovery registration
// so that the SRAT custom component can auto-discover this addon.
type DiscoveryServiceInterface interface {
	// RegisterDiscovery posts a discovery message to the Supervisor API.
	RegisterDiscovery(ctx context.Context) error
	// UnregisterDiscovery removes the discovery message from the Supervisor API.
	UnregisterDiscovery(ctx context.Context) error
}

type discoveryService struct {
	ctx           context.Context
	state         *dto.ContextState
	addonsService AddonsServiceInterface
	discoveryUUID string // UUID returned by the Supervisor after registration
}

// discoveryRequest is the JSON payload for POST /discovery.
type discoveryRequest struct {
	Service string         `json:"service"`
	Config  map[string]any `json:"config"`
}

// discoveryResponse is the JSON response from POST /discovery.
type discoveryResponse struct {
	Result string `json:"result"`
	Data   struct {
		UUID string `json:"uuid"`
	} `json:"data"`
}

type DiscoveryServiceParams struct {
	fx.In
	Ctx           context.Context
	State         *dto.ContextState
	AddonsService AddonsServiceInterface
}

func NewDiscoveryService(lc fx.Lifecycle, params DiscoveryServiceParams) DiscoveryServiceInterface {
	svc := &discoveryService{
		ctx:           params.Ctx,
		state:         params.State,
		addonsService: params.AddonsService,
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

func (s *discoveryService) RegisterDiscovery(ctx context.Context) error {
	if s.state.SupervisorURL == "" || s.state.SupervisorURL == "demo" {
		slog.DebugContext(ctx, "Skipping discovery registration — not running in Supervisor mode")
		return nil
	}

	if s.state.SupervisorToken == "" {
		slog.DebugContext(ctx, "Skipping discovery registration — no Supervisor token available")
		return nil
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

	payload := discoveryRequest{
		Service: "srat",
		Config: map[string]any{
			"host": host,
			"port": 8099,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "failed to marshal discovery payload")
	}

	url := fmt.Sprintf("%s/discovery", s.state.SupervisorURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return errors.Wrap(err, "failed to create discovery request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.state.SupervisorToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to send discovery request")
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.Errorf("discovery registration failed: HTTP %d — %s", resp.StatusCode, string(respBody))
	}

	var result discoveryResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return errors.Wrap(err, "failed to parse discovery response")
	}

	s.discoveryUUID = result.Data.UUID
	slog.InfoContext(ctx, "Registered Supervisor discovery for SRAT custom component", "uuid", s.discoveryUUID, "host", host)

	return nil
}

func (s *discoveryService) UnregisterDiscovery(ctx context.Context) error {
	if s.discoveryUUID == "" {
		return nil // nothing to unregister
	}

	if s.state.SupervisorURL == "" || s.state.SupervisorURL == "demo" {
		return nil
	}

	url := fmt.Sprintf("%s/discovery/%s", s.state.SupervisorURL, s.discoveryUUID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return errors.Wrap(err, "failed to create discovery delete request")
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.state.SupervisorToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to send discovery delete request")
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return errors.Errorf("discovery unregistration failed: HTTP %d — %s", resp.StatusCode, string(respBody))
	}

	slog.InfoContext(ctx, "Unregistered Supervisor discovery for SRAT", "uuid", s.discoveryUUID)
	s.discoveryUUID = ""

	return nil
}
