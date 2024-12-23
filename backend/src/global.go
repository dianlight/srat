package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/dianlight/srat/data"
	"github.com/jinzhu/copier"
)

type GlobalConfig struct {
	Workgroup         string   `json:"workgroup"`
	Mountoptions      []string `json:"mountoptions"`
	AllowHost         []string `json:"allow_hosts"`
	VetoFiles         []string `json:"veto_files"`
	CompatibilityMode bool     `json:"compatibility_mode"`
	EnableRecycleBin  bool     `json:"recyle_bin_enabled"`
	Interfaces        []string `json:"interfaces"`
	BindAllInterfaces bool     `json:"bind_all_interfaces"`
	LogLevel          string   `json:"log_level"`
	MultiChannel      bool     `json:"multi_channel"`
}

// UpdateGlobalConfig godoc
//
//	@Summary		Update the configuration for the global samba settings
//	@Description	Update the configuration for the global samba settings
//	@Tags			samba
//	@Accept			json
//	@Produce		json
//	@Param			config	body	GlobalConfig	true	"Update model"
//	@Success		200 {object}    GlobalConfig
//	@Failure		400	{object}	ResponseError
//	@Failure		500	{object}	ResponseError
//	@Router			/global [put]
//	@Router			/global [patch]
func updateGlobalConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var globalConfig GlobalConfig

	err := json.NewDecoder(r.Body).Decode(&globalConfig)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//pretty.Logf("1res: %v", data.Config.Options)

	copier.CopyWithOption(&data.Config.Options, &globalConfig, copier.Option{IgnoreEmpty: true, DeepCopy: true})
	//mergo.MapWithOverwrite(&data.Config.Options, globalConfig)
	//pretty.Logf("2res: %v", data.Config.Options)

	// Recheck the config
	copier.CopyWithOption(&globalConfig, &data.Config.Options, copier.Option{IgnoreEmpty: true, DeepCopy: true})
	//mergo.MapWithOverwrite(&globalConfig, data.Config)

	jsonResponse, jsonError := json.Marshal(globalConfig)

	if jsonError != nil {
		log.Println("Unable to encode JSON")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(jsonError.Error()))
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
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
func getGlobalConfig(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var globalConfig GlobalConfig

	copier.CopyWithOption(&globalConfig, &data.Config.Options, copier.Option{IgnoreEmpty: true, DeepCopy: true})
	//mergo.MapWithOverwrite(&globalConfig, data.Config.Options)

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
