package api

import (
	"net/http"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/server"
)

type SettingsHanler struct {
	apiContext *ContextState
}

func NewSettingsHanler(apiContext *ContextState) *SettingsHanler {
	p := new(SettingsHanler)
	p.apiContext = apiContext
	return p
}

func (broker *SettingsHanler) Patterns() []server.RouteDetail {
	return []server.RouteDetail{
		{Pattern: "/settings", Method: "GET", Handler: broker.GetSettings},
		{Pattern: "/settings", Method: "PUT", Handler: broker.UpdateSettings},
		{Pattern: "/settings", Method: "PATCH", Handler: broker.UpdateSettings},
	}
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
func (self *SettingsHanler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var config dto.Settings
	err := HttpJSONRequest(&config, w, r)
	if err != nil {
		return
	}

	var dbconfig dbom.Properties
	err = dbconfig.Load()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	var conv converter.DtoToDbomConverterImpl

	//err = mapper.Map(context.Background(), &dbconfig, config)
	err = conv.SettingsToProperties(config, &dbconfig)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	err = dbconfig.Save()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	//context_state := StateFromContext(r.Context())

	//err = mapper.Map(context.Background(), &config, dbconfig)
	err = conv.PropertiesToSettings(dbconfig, &config)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	self.apiContext.DataDirtyTracker.Settings = true
	//UpdateLimiter = rate.Sometimes{Interval: 30 * time.Minute}
	HttpJSONReponse(w, config, nil)
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
func (self *SettingsHanler) GetSettings(w http.ResponseWriter, r *http.Request) {
	var dbsettings dbom.Properties
	var conv converter.DtoToDbomConverterImpl
	err := dbsettings.Load()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	var settings dto.Settings
	err = conv.PropertiesToSettings(dbsettings, &settings)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	HttpJSONReponse(w, settings, nil)
}
