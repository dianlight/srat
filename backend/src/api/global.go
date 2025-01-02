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
//	@Failure		400	{object}	dm.ResponseError
//	@Failure		500	{object}	dm.ResponseError
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

// GetGlobakConfig godoc
//
//	@Summary		Get the configuration for the global samba settings
//	@Description	Get the configuration for the global samba settings
//	@Tags			samba
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	dto.Settings
//	@Failure		400	{object}	dm.ResponseError
//	@Failure		500	{object}	dm.ResponseError
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
