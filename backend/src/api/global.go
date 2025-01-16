package api

import (
	"context"
	"net/http"
	"time"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/mapper"
	"golang.org/x/time/rate"
)

// UpdateSettings godoc
//
//	@Summary		Update the configuration for the global samba settings
//	@Description	Update the configuration for the global samba settings
//	@Tags			samba
//	@Accept			json
//	@Produce		json
//	@Param			config	body		dto.Settings	true	"Update model"
//	@Success		200		{object}	dto.Settings
//	@Failure		400		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/global [put]
//	@Router			/global [patch]
func UpdateSettings(w http.ResponseWriter, r *http.Request) {
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

	err = mapper.Map(context.Background(), &dbconfig, config)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	err = dbconfig.Save()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	context_state := (&dto.ContextState{}).FromContext(r.Context())

	err = mapper.Map(context.Background(), &config, dbconfig)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	context_state.DataDirtyTracker.Settings = true
	UpdateLimiter = rate.Sometimes{Interval: 30 * time.Minute}
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
//	@Failure		400	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/global [get]
func GetSettings(w http.ResponseWriter, r *http.Request) {
	var dbsettings dbom.Properties
	err := dbsettings.Load()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	var settings dto.Settings
	err = mapper.Map(context.Background(), &settings, dbsettings)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	HttpJSONReponse(w, settings, nil)
}
