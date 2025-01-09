package api

import (
	"net/http"
	"time"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
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
//	@Success		204
//	@Failure		400	{object}	dto.ResponseError
//	@Failure		500	{object}	dto.ResponseError
//	@Router			/global [put]
//	@Router			/global [patch]
func UpdateSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var config dto.Settings
	err := config.FromJSONBody(w, r)
	if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "A0 Internal error", err)
		return
	}

	var dbconfig dbom.Properties
	err = dbconfig.Load()
	if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "A1 Internal error", err)
		return
	}

	err = config.ToArray(&dbconfig)
	if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "A2 Internal error", err)
		return
	}

	err = dbconfig.Save()
	if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "A3 Internal error", err)
		return
	}

	context_state := (&dto.ContextState{}).FromContext(r.Context())

	config.FromArray(dbconfig)
	context_state.DataDirtyTracker.Settings = true
	UpdateLimiter = rate.Sometimes{Interval: 30 * time.Minute}
	config.ToResponse(http.StatusOK, w)
}

// GetSettings godoc
//
//	@Summary		Get the configuration for the global samba settings
//	@Description	Get the configuration for the global samba settings
//	@Tags			samba
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	dto.Settings
//	@Failure		400	{object}	dto.ResponseError
//	@Failure		500	{object}	dto.ResponseError
//	@Router			/global [get]
func GetSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var dbsettings dbom.Properties
	err := dbsettings.Load()
	if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Internal error", err)
		return
	}
	var settings dto.Settings
	err = settings.FromArray(&dbsettings)
	if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Internal error", err)
		return
	}
	settings.ToResponse(http.StatusOK, w)
}

// PersistAllConfig godoc
//
//	@Summary		Persiste the current samba config
//	@Description	Save dirty changes to the disk
//	@Tags			samba
//	@Accept			json
//	@Produce		json
//	@Success		200 {object}	dto.Settings
//	@Failure		400	{object}	dto.ResponseError
//	@Failure		500	{object}	dto.ResponseError
//	@Router			/config [put]
//	@Router			/config [patch]
/*
func PersistAllConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	context_state := (&dto.ContextState{}).FromContext(r.Context())

	//config.SaveConfig(addon_config) // FIXME: Change to DB
	context_state.DataDirtyTracker.Settings = false
	context_state.DataDirtyTracker.Users = false

	/*
		err := PersistSharesState()
		if err != nil {
			DoResponseError(http.StatusInternalServerError, w, "Error Persisting Share States", err)
			return
		}
			 FIXME: Persist share state
	* /
	context_state.DataDirtyTracker.Shares = false

	/*
		err = PersistVolumesState()
		if err != nil {
			DoResponseError(http.StatusInternalServerError, w, "Error Persisting Volume States", err)
			return
		}
			FIXME: Persist volume state
	* /
	context_state.DataDirtyTracker.Volumes = false

	context_state.Settings.ToResponse(http.StatusOK, w)
}
*/

// RollbackConfig godoc
//
//	@Summary		Rollback the current samba config
//	@Description	Revert to the last saved samba config
//	@Tags			samba
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	dto.Settings
//	@Failure		400	{object}	dto.ResponseError
//	@Failure		500	{object}	dto.ResponseError
//	@Router			/config [delete]
/*
func RollbackConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	context_state := (&dto.ContextState{}).FromContext(r.Context())

	// FIXME: Implement rollback

	//	config, err := config.RollbackConfig(addon_config) // FIXME: Change to DB
	//	if err != nil {
	//		settings.ToResponseError(http.StatusInternalServerError, w, "Error rolling back config", err)
	//		return
	//	}

	//	copier.CopyWithOption(addon_config, config, copier.Option{IgnoreEmpty: false, DeepCopy: true})
	//	settings.From(config)

	//context_state.DataDirtyTracker.Settings = false
	//data_dirty_tracker.Users = false FIXME: Implement this
	//data_dirty_tracker.Shares = false FIXME: Implement this
	//data_dirty_tracker.Volumes = false FIXME: Implement this

	context_state.Settings.ToResponse(http.StatusOK, w)
}
*/
