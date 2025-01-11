package dto

type EventType string

const (
	EventUpdate    EventType = "update"
	EventHeartbeat EventType = "heartbeat"
	EventShare     EventType = "share"
	EventVolumes   EventType = "volumes"
	EventDirty     EventType = "dirty"
)

var EventTypes = []string{
	string(EventUpdate),
	string(EventHeartbeat),
	string(EventShare),
	string(EventVolumes),
}

/*
func (self *EventType) From(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self *EventType) FromIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self EventType) To(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self EventType) ToIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self EventType) ToResponse(code int, w http.ResponseWriter) error {
	return doResponse(code, w, self)
}
func (self EventType) ToResponseError(code int, w http.ResponseWriter, message string, body any) error {
	return doResponseError(code, w, message, body)
}
func (self *EventType) FromJSONBody(w http.ResponseWriter, r *http.Request) error {
	return fromJSONBody(w, r, self)
}
*/
