package api

import (
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"
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

func (handler *SettingsHanler) Routers(srv *fuego.Server) error {
	fuego.Get(srv, "/settings", handler.GetSettings, option.Tags("samba"), option.Description("Get the configuration for the global samba settings"))
	fuego.Put(srv, "/settings", handler.UpdateSettings, option.Tags("samba"), option.Description("Update the configuration for the global samba settings"))
	fuego.Patch(srv, "/settings", handler.UpdateSettings, option.Tags("samba"), option.Description("Update the configuration for the global samba settings"))
	return nil
}

func (self *SettingsHanler) UpdateSettings(c fuego.ContextWithBody[dto.Settings]) (*dto.Settings, error) {
	config, err := c.Body()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	var dbconfig dbom.Properties
	err = dbconfig.Load()
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
	return &config, nil
}

func (self *SettingsHanler) GetSettings(c fuego.ContextNoBody) (*dto.Settings, error) {
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
	return &settings, nil

}
