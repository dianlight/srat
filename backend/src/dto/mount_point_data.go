package dto

import (
	"net/http"

	"github.com/jinzhu/copier"
)

type MountPointData struct {
	Path        string        `json:"path"`
	DefaultPath string        `json:"default_path"`
	Label       string        `json:"label"`
	Name        string        `json:"name"`
	FSType      string        `json:"fstype"`
	Flags       MounDataFlags `json:"flags"`
	Data        string        `json:"data,omitempty"`
}

func (self *MountPointData) From(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self *MountPointData) FromIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self MountPointData) To(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self MountPointData) ToIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self MountPointData) ToResponse(code int, w http.ResponseWriter) error {
	return doResponse(code, w, self)
}
func (self MountPointData) ToResponseError(code int, w http.ResponseWriter, message string, body any) error {
	return doResponseError(code, w, message, body)
}
func (self *MountPointData) FromJSONBody(w http.ResponseWriter, r *http.Request) error {
	return fromJSONBody(w, r, self)
}
