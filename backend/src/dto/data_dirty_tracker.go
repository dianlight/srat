package dto

type DataDirtyTracker struct {
	Shares   bool `json:"shares"`
	Users    bool `json:"users"`
	Volumes  bool `json:"volumes"`
	Settings bool `json:"settings"`
}

/*
func (self *DataDirtyTracker) From(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self *DataDirtyTracker) FromIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self DataDirtyTracker) To(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self DataDirtyTracker) ToIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self DataDirtyTracker) ToResponse(code int, w http.ResponseWriter) error {
	return doResponse(code, w, self)
}
func (self DataDirtyTracker) ToResponseError(code int, w http.ResponseWriter, message string, body any) error {
	return doResponseError(code, w, message, body)
}
func (self *DataDirtyTracker) FromJSONBody(w http.ResponseWriter, r *http.Request) error {
	return fromJSONBody(w, r, self)
}
*/
