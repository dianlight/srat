package dto

import (
	"net/http"

	"github.com/google/go-github/v68/github"
	"github.com/jinzhu/copier"
)

type ReleaseAsset struct {
	UpdateStatus int8                      `json:"update_status"`
	LastRelease  *github.RepositoryRelease `json:"last_release,omitempty"`
	ArchAsset    *github.ReleaseAsset      `json:"arch,omitempty"`
}

func (self *ReleaseAsset) From(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self *ReleaseAsset) FromIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self ReleaseAsset) To(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self ReleaseAsset) ToIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self ReleaseAsset) ToResponse(code int, w http.ResponseWriter) error {
	return doResponse(code, w, self)
}
func (self ReleaseAsset) ToResponseError(code int, w http.ResponseWriter, message string, body any) error {
	return doResponseError(code, w, message, body)
}
func (self *ReleaseAsset) FromJSONBody(w http.ResponseWriter, r *http.Request) error {
	return fromJSONBody(w, r, self)
}
