package dto

import (
	"net/http"

	"github.com/jinzhu/copier"
)

type ResponseError struct {
	Code  int    `json:"code"`
	Error string `json:"error"`
	Body  any    `json:"body"`
}

func (self *ResponseError) From(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self *ResponseError) FromIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self ResponseError) To(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self ResponseError) ToIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self ResponseError) ToResponse(code int, w http.ResponseWriter) error {
	return doResponse(code, w, self)
}
func (self ResponseError) ToResponseError(code int, w http.ResponseWriter, message string, body any) error {
	return doResponseError(code, w, message, body)
}
func (self *ResponseError) FromJSONBody(w http.ResponseWriter, r *http.Request) error {
	return fromJSONBody(w, r, self)
}
