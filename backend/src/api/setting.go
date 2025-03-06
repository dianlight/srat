package api

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/ztrue/tracerr"
)

type SettingsHanler struct {
	apiContext   *dto.ContextState
	dirtyService service.DirtyDataServiceInterface
}

func NewSettingsHanler(apiContext *dto.ContextState, dirtyService service.DirtyDataServiceInterface) *SettingsHanler {
	p := new(SettingsHanler)
	p.apiContext = apiContext
	p.dirtyService = dirtyService
	return p
}

/*
func (self *SettingsHanler) HumaRoute(api *huma.API) error {
	huma.Get(*api, "/settings", self.GetSettings)
	huma.Put(*api, "/settings", self.UpdateSettings)
	//	huma.Patch(*api, "/settings", self.UpdateSettings)
	return nil
}
*/

func (self *SettingsHanler) RegisterSettings(api huma.API) {
	//slog.Debug("Autoregister Settings")
	huma.Get(api, "/settings", self.GetSettings)
	huma.Put(api, "/settings", self.UpdateSettings)
	/*
		huma.Register(api, huma.Operation{
			OperationID: "ListItems",
			Method: http.MethodGet,
			Path: "/items",
		}, s.ListItems)
	*/
}

// UpdateSettings godoc
//
//	@Summary		Update the configuration for the global samba settings
//	@Description	Update the configuration for the global samba settings
//	@Tags			samba
//	@Accept			json
//	@Produce		json
//	@Param			config	body		dto.Settings	true	"Update model"
//	@Success		200		{object}	dto.Settings
//	@Failure		400		{object}	dto.ErrorInfo
//	@Failure		500		{object}	dto.ErrorInfo
//	@Router			/settings [put]
//	@Router			/settings [patch]
func (self *SettingsHanler) UpdateSettings(ctx context.Context, input *struct {
	//Name string `path:"name" maxLength:"30" example:"world" doc:"Name to greet"`
	Body dto.Settings
}) (*struct{ Body dto.Settings }, error) {
	config := input.Body

	var dbconfig dbom.Properties
	err := dbconfig.Load()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	var conv converter.DtoToDbomConverterImpl

	err = conv.SettingsToProperties(config, &dbconfig)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	err = dbconfig.Save()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	err = conv.PropertiesToSettings(dbconfig, &config)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	self.dirtyService.SetDirtySettings()
	return &struct{ Body dto.Settings }{Body: config}, nil
}

// GetSettings godoc
//
//	@Summary		Get the configuration for the global samba settings
//	@Description	Get the configuration for the global samba settings
//	@Tags			samba
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	dto.Settings
//	@Failure		400	{object}	dto.ErrorInfo
//	@Failure		500	{object}	dto.ErrorInfo
//	@Router			/settings [get]
func (self *SettingsHanler) GetSettings(ctx context.Context, input *struct{}) (*struct{ Body dto.Settings }, error) {
	var dbsettings dbom.Properties
	var conv converter.DtoToDbomConverterImpl
	err := dbsettings.Load()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	var settings dto.Settings
	err = conv.PropertiesToSettings(dbsettings, &settings)
	if err != nil {
		return nil, tracerr.Wrap(err)

	}
	return &struct{ Body dto.Settings }{Body: settings}, nil
}
