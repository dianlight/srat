package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/data"
	"github.com/dianlight/srat/dto"
	"github.com/jinzhu/copier"
)

// UpdateGlobalConfig godoc
//
//	@Summary		Update the configuration for the global samba settings
//	@Description	Update the configuration for the global samba settings
//	@Tags			samba
//	@Accept			json
//	@Produce		json
//	@Param			config	body		GlobalConfig	true	"Update model"
//	@Success		200		{object}	GlobalConfig
//	@Success		204
//	@Failure		400	{object}	ResponseError
//	@Failure		500	{object}	ResponseError
//	@Router			/global [put]
//	@Router			/global [patch]
func UpdateGlobalConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var globalConfig dto.Settings

	err := json.NewDecoder(r.Body).Decode(&globalConfig)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tmpConfig := config.Config{}
	addon_config := r.Context().Value("addon_config").(*config.Config)
	addon_option := r.Context().Value("addon_option").(*config.Options)

	copier.CopyWithOption(&tmpConfig, &addon_config, copier.Option{IgnoreEmpty: true, DeepCopy: true})
	copier.CopyWithOption(&tmpConfig.Options, &globalConfig, copier.Option{IgnoreEmpty: true, DeepCopy: true})
	copier.CopyWithOption(&tmpConfig, &globalConfig, copier.Option{IgnoreEmpty: true, DeepCopy: true})

	if reflect.DeepEqual(addon_config, &tmpConfig) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	copier.CopyWithOption(&addon_config, &tmpConfig, copier.Option{IgnoreEmpty: true, DeepCopy: true})

	var retglobalConfig dto.Settings = dto.Settings{}
	data.DirtySectionState.Settings = true // FIXME: Change mode I set dirty

	// Recheck the config
	copier.CopyWithOption(&retglobalConfig, &addon_option, copier.Option{IgnoreEmpty: true, DeepCopy: true})
	copier.CopyWithOption(&retglobalConfig, &addon_config, copier.Option{IgnoreEmpty: true, DeepCopy: true})

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

}

// GetGlobakConfig godoc
//
//	@Summary		Get the configuration for the global samba settings
//	@Description	Get the configuration for the global samba settings
//	@Tags			samba
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	GlobalConfig
//	@Failure		400	{object}	ResponseError
//	@Failure		500	{object}	ResponseError
//	@Router			/global [get]
func GetGlobalConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var globalConfig dto.Settings

	addon_config := r.Context().Value("addon_config").(*config.Config)
	addon_option := r.Context().Value("addon_option").(*config.Options)

	copier.CopyWithOption(&globalConfig, &addon_option, copier.Option{IgnoreEmpty: true, DeepCopy: true})
	copier.CopyWithOption(&globalConfig, &addon_config, copier.Option{IgnoreEmpty: true, DeepCopy: true})

	jsonResponse, jsonError := json.Marshal(globalConfig)

	if jsonError != nil {
		fmt.Println("Unable to encode JSON")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(jsonError.Error()))
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}
}
