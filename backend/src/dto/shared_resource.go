package dto

import (
	"net/http"

	"github.com/jinzhu/copier"
)

type SharedResource struct {
	Name        string   `json:"name,omitempty"`
	Path        string   `json:"path"`
	FS          string   `json:"fs"`
	Disabled    bool     `json:"disabled,omitempty"`
	Users       []string `json:"users,omitempty"`
	RoUsers     []string `json:"ro_users,omitempty"`
	TimeMachine bool     `json:"timemachine,omitempty"`
	Usage       string   `json:"usage,omitempty"`
}

func (self *SharedResource) From(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self *SharedResource) FromIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self SharedResource) To(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self SharedResource) ToIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self SharedResource) ToResponse(code int, w http.ResponseWriter) error {
	return doResponse(code, w, self)
}
func (self SharedResource) ToResponseError(code int, w http.ResponseWriter, message string, body any) error {
	return doResponseError(code, w, message, body)
}
func (self *SharedResource) FromJSONBody(w http.ResponseWriter, r *http.Request) error {
	return fromJSONBody(w, r, self)
}

type SharedResources map[string]SharedResource

func (self *SharedResources) From(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self *SharedResources) FromIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self SharedResources) To(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self SharedResources) ToIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self SharedResources) ToResponse(code int, w http.ResponseWriter) error {
	return doResponse(code, w, self)
}
func (self SharedResources) ToResponseError(code int, w http.ResponseWriter, message string, body any) error {
	return doResponseError(code, w, message, body)
}
func (self *SharedResources) FromJSONBody(w http.ResponseWriter, r *http.Request) error {
	return fromJSONBody(w, r, self)
}
