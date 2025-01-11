package dto

type HealthPing struct {
	Alive     bool   `json:"alive"`
	ReadOnly  bool   `json:"read_only"`
	Samba     int32  `json:"samba_pid"`
	LastError string `json:"last_error"`
}

/*
func (self *HealthPing) From(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self *HealthPing) FromIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self HealthPing) To(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self HealthPing) ToIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self HealthPing) ToResponse(code int, w http.ResponseWriter) error {
	return doResponse(code, w, self)
}
func (self HealthPing) ToResponseError(code int, w http.ResponseWriter, message string, body any) error {
	return doResponseError(code, w, message, body)
}
func (self *HealthPing) FromJSONBody(w http.ResponseWriter, r *http.Request) error {
	return fromJSONBody(w, r, self)
}
*/
