package dto

import (
	"net/http"

	"github.com/jinzhu/copier"
)

type BlockInfo struct {
	TotalSizeBytes uint64 `json:"total_size_bytes"`
	// Partitions contains an array of pointers to `Partition` structs, one for
	// each partition on any disk drive on the host system.
	Partitions []*BlockPartition `json:"partitions"`
}

func (self *BlockInfo) From(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self *BlockInfo) FromIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self BlockInfo) To(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self BlockInfo) ToIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self BlockInfo) ToResponse(code int, w http.ResponseWriter) error {
	return doResponse(code, w, self)
}
func (self BlockInfo) ToResponseError(code int, w http.ResponseWriter, message string, body any) error {
	return doResponseError(code, w, message, body)
}
func (self *BlockInfo) FromJSONBody(w http.ResponseWriter, r *http.Request) error {
	return fromJSONBody(w, r, self)
}
