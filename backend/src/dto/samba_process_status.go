package dto

import (
	"net/http"
	"time"

	"github.com/jinzhu/copier"
)

type SambaProcessStatus struct {
	PID           int32     `json:"pid"`
	Name          string    `json:"name"`
	CreateTime    time.Time `json:"create_time"`
	CPUPercent    float64   `json:"cpu_percent"`
	MemoryPercent float32   `json:"memory_percent"`
	OpenFiles     int32     `json:"open_files"`
	Connections   int32     `json:"connections"`
	Status        []string  `json:"status"`
	IsRunning     bool      `json:"is_running"`
}

func (self *SambaProcessStatus) From(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self *SambaProcessStatus) FromIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self SambaProcessStatus) To(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self SambaProcessStatus) ToIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self SambaProcessStatus) ToResponse(code int, w http.ResponseWriter) error {
	return doResponse(code, w, self)
}
func (self SambaProcessStatus) ToResponseError(code int, w http.ResponseWriter, message string, body any) error {
	return doResponseError(code, w, message, body)
}
func (self *SambaProcessStatus) FromJSONBody(w http.ResponseWriter, r *http.Request) error {
	return fromJSONBody(w, r, self)
}
