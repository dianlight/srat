package dto

import (
	"net/http"

	"github.com/jinzhu/copier"
)

type WebSocketMessageEnvelopeAction string

const (
	ActionSubscribe   WebSocketMessageEnvelopeAction = "subscribe"
	ActionUnsubscribe WebSocketMessageEnvelopeAction = "unsubscribe"
	ActionError       WebSocketMessageEnvelopeAction = "error"
)

type WebSocketMessageEnvelope struct {
	Event  EventType                      `json:"event"`
	Uid    string                         `json:"uid"`
	Data   any                            `json:"data"`
	Action WebSocketMessageEnvelopeAction `json:"action,omitempty"`
}

func (self *WebSocketMessageEnvelope) From(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self *WebSocketMessageEnvelope) FromIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self WebSocketMessageEnvelope) To(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self WebSocketMessageEnvelope) ToIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self WebSocketMessageEnvelope) ToResponse(code int, w http.ResponseWriter) error {
	return doResponse(code, w, self)
}
func (self WebSocketMessageEnvelope) ToResponseError(code int, w http.ResponseWriter, message string, body any) error {
	return doResponseError(code, w, message, body)
}
func (self *WebSocketMessageEnvelope) FromJSONBody(w http.ResponseWriter, r *http.Request) error {
	return fromJSONBody(w, r, self)
}
