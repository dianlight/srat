package api

import (
	"net/http"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dm"
	"github.com/dianlight/srat/dto"
	"github.com/jinzhu/copier"
	"github.com/kr/pretty"
)

// UpdateGlobalConfig godoc
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
func UpdateGlobalConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var globalConfig dto.Settings

	err := globalConfig.FromJSONBody(w, r)
	//	err := json.NewDecoder(r.Body).Decode(&globalConfig)
	if err != nil {
		return
	}

	tmpConfig := config.Config{}
	tmpSettings := dto.Settings{}
	addon_config := r.Context().Value("addon_config").(*config.Config)
	//addon_option := r.Context().Value("addon_option").(*config.Options)

	//dto.Mapper.Map(&tmpConfig, globalConfig)
	tmpSettings.From(addon_config)
	tmpSettings.FromIgnoreEmpty(globalConfig)
	copier.CopyWithOption(&tmpConfig, &addon_config, copier.Option{IgnoreEmpty: false, DeepCopy: true})
	//pretty.Println("---------------", tmpConfig, addon_config)
	tmpSettings.ToIgnoreEmpty(&tmpConfig)

	//copier.CopyWithOption(&tmpConfig, &addon_config, copier.Option{IgnoreEmpty: true, DeepCopy: true})
	//copier.CopyWithOption(&tmpConfig.Options, &globalConfig, copier.Option{IgnoreEmpty: true, DeepCopy: true})
	//copier.CopyWithOption(&tmpConfig, &globalConfig, copier.Option{IgnoreEmpty: true, DeepCopy: true})

	//pretty.Pdiff(log.Default(), addon_config, &tmpConfig)

	if diff := pretty.Diff(addon_config, &tmpConfig); len(diff) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	} else {
		//pretty.Println(diff)
	}

	tmpSettings.To(&addon_config)
	//copier.CopyWithOption(&addon_config, &tmpConfig, copier.Option{IgnoreEmpty: true, DeepCopy: true})

	var retglobalConfig dto.Settings = dto.Settings{}
	data_dirty_tracker := r.Context().Value("data_dirty_tracker").(*dm.DataDirtyTracker)
	data_dirty_tracker.Settings = true

	// Recheck the config
	//copier.CopyWithOption(&retglobalConfig, &addon_option, copier.Option{IgnoreEmpty: true, DeepCopy: true})
	//copier.CopyWithOption(&retglobalConfig, &addon_config, copier.Option{IgnoreEmpty: true, DeepCopy: true})
	retglobalConfig.From(addon_config)

	retglobalConfig.ToResponse(http.StatusOK, w)

	/*

		jsonResponse, jsonError := json.Marshal(retglobalConfig)

		if jsonError != nil {
			log.Println("Unable to encode JSON")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(jsonError.Error()))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(jsonResponse)
			//	UpdateLimiter = rate.Sometimes{Interval: 30 * time.Minute}
		}
	*/

}

// GetGlobalConfig godoc
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
func GetGlobalConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var globalConfig dto.Settings

	addon_config := r.Context().Value("addon_config").(*config.Config)
	// addon_option := r.Context().Value("addon_option").(*config.Options)

	globalConfig.From(addon_config)

	//copier.CopyWithOption(&globalConfig, &addon_option, copier.Option{IgnoreEmpty: true, DeepCopy: true})
	//	copier.CopyWithOption(&globalConfig, &addon_config, copier.Option{IgnoreEmpty: true, DeepCopy: true})

	globalConfig.ToResponse(http.StatusOK, w)
}

// PersistConfig godoc
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
func PersistConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	addon_config := r.Context().Value("addon_config").(*config.Config)
	data_dirty_tracker := r.Context().Value("data_dirty_tracker").(*dm.DataDirtyTracker)

	//config.SaveConfig(addon_config) // FIXME: Change to DB
	data_dirty_tracker.Settings = false
	data_dirty_tracker.Users = false

	/*
		err := PersistSharesState()
		if err != nil {
			DoResponseError(http.StatusInternalServerError, w, "Error Persisting Share States", err)
			return
		}
			 FIXME: Persist share state
	*/
	data_dirty_tracker.Shares = false

	/*
		err = PersistVolumesState()
		if err != nil {
			DoResponseError(http.StatusInternalServerError, w, "Error Persisting Volume States", err)
			return
		}
			FIXME: Persist volume state
	*/
	data_dirty_tracker.Volumes = false

	var globalConfig dto.Settings = dto.Settings{}
	globalConfig.From(addon_config)
	globalConfig.ToResponse(http.StatusOK, w)
}

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
func RollbackConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	addon_config := r.Context().Value("addon_config").(*config.Config)
	var settings dto.Settings
	settings.From(addon_config)

	config, err := config.RollbackConfig(addon_config) // FIXME: Change to DB
	if err != nil {
		settings.ToResponseError(http.StatusInternalServerError, w, "Error rolling back config", err)
		return
	}

	copier.CopyWithOption(addon_config, config, copier.Option{IgnoreEmpty: false, DeepCopy: true})
	settings.From(config)

	data_dirty_tracker := r.Context().Value("data_dirty_tracker").(*dm.DataDirtyTracker)
	data_dirty_tracker.Settings = false
	//data_dirty_tracker.Users = false FIXME: Implement this
	//data_dirty_tracker.Shares = false FIXME: Implement this
	//data_dirty_tracker.Volumes = false FIXME: Implement this

	settings.ToResponse(http.StatusOK, w)
}
