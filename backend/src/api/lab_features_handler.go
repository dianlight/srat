package api

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"go.uber.org/fx"
)

type LabFeatureHandler struct {
	settingService service.SettingServiceInterface
	registry       *service.LabFeatureRegistry
}

type LabFeatureHandlerParams struct {
	fx.In
	SettingService service.SettingServiceInterface
}

func NewLabFeatureHandler(params LabFeatureHandlerParams) *LabFeatureHandler {
	return &LabFeatureHandler{
		settingService: params.SettingService,
		registry:       service.NewLabFeatureRegistry(),
	}
}

// RegisterLabFeatureHandler registers GET /lab_features.
// Alpha features are omitted entirely when the build environment is
// "production" — the frontend must not render what the release cannot serve.
func (h *LabFeatureHandler) RegisterLabFeatureHandler(api huma.API) {
	huma.Get(api, "/lab_features", h.HandleGetLabFeatures, huma.OperationTags("system"))
}

func (h *LabFeatureHandler) HandleGetLabFeatures(ctx context.Context, input *struct{}) (*struct{ Body []dto.LabFeature }, error) {
	env := config.Environment()

	settings, err := h.settingService.Load()
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to read settings", err)
	}
	labMode := settings != nil && settings.ExperimentalLabMode

	features := make([]dto.LabFeature, 0, len(h.registry.All()))
	for _, f := range h.registry.All() {
		available := false
		switch f.Status {
		case service.StatusAlpha:
			// Alpha features exist only in development/pre-release builds.
			if env == "production" {
				continue
			}
			available = true
		case service.StatusBeta:
			available = labMode
		}
		features = append(features, dto.LabFeature{
			Key:         f.Key,
			Name:        f.Name,
			Description: f.Description,
			Status:      string(f.Status),
			Available:   available,
		})
	}
	return &struct{ Body []dto.LabFeature }{Body: features}, nil
}
