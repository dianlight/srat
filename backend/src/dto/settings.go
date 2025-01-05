package dto

import (
	"net/http"

	"github.com/jinzhu/copier"
)

type Settings struct {
	Workgroup         string        `json:"workgroup"`
	Mountoptions      []string      `json:"mountoptions"`
	AllowHost         []string      `json:"allow_hosts"`
	VetoFiles         []string      `json:"veto_files"`
	CompatibilityMode bool          `json:"compatibility_mode"`
	EnableRecycleBin  bool          `json:"recyle_bin_enabled"`
	Interfaces        []string      `json:"interfaces"`
	BindAllInterfaces bool          `json:"bind_all_interfaces"`
	LogLevel          string        `json:"log_level"`
	MultiChannel      bool          `json:"multi_channel"`
	UpdateChannel     UpdateChannel `json:"update_channel"`
}

func (self *Settings) From(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self *Settings) FromIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self Settings) To(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self Settings) ToIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self Settings) ToResponse(code int, w http.ResponseWriter) error {
	return doResponse(code, w, self)
}
func (self Settings) ToResponseError(code int, w http.ResponseWriter, message string, body any) error {
	return doResponseError(code, w, message, body)
}
func (self *Settings) FromJSONBody(w http.ResponseWriter, r *http.Request) error {
	return fromJSONBody(w, r, self)
}

func (self *Settings) ToMap() map[string]interface{} {
	result := make(map[string]interface{})
	copier.Copy(&result, self)
	return result
}
