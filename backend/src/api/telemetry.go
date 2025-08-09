package api

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
)

type TelemetryHandler struct {
	apiContext       *dto.ContextState
	telemetryService service.TelemetryServiceInterface
}

// NewTelemetryHandler creates a new instance of TelemetryHandler
func NewTelemetryHandler(apiContext *dto.ContextState, telemetryService service.TelemetryServiceInterface) *TelemetryHandler {
	return &TelemetryHandler{
		apiContext:       apiContext,
		telemetryService: telemetryService,
	}
}

// RegisterTelemetry registers the telemetry-related endpoints with the provided API
func (self *TelemetryHandler) RegisterTelemetry(api huma.API) {
	huma.Get(api, "/telemetry/modes", self.GetTelemetryModes, huma.OperationTags("system"))
	huma.Get(api, "/telemetry/internet-connection", self.GetInternetConnection, huma.OperationTags("system"))
}

// GetTelemetryModes returns all available telemetry modes
func (self *TelemetryHandler) GetTelemetryModes(ctx context.Context, input *struct{}) (*struct{ Body []string }, error) {
	modes := []string{}

	// Use the generated enum container to get all modes
	for telemetryMode := range dto.TelemetryModes.All() {
		modes = append(modes, telemetryMode.String())
	}

	return &struct{ Body []string }{Body: modes}, nil
}

// GetInternetConnection returns whether internet connection is available
func (self *TelemetryHandler) GetInternetConnection(ctx context.Context, input *struct{}) (*struct{ Body bool }, error) {
	connected := self.telemetryService.IsConnectedToInternet()
	return &struct{ Body bool }{Body: connected}, nil
}
