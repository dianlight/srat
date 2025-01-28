package dto

//type WebSocketMessageEnvelopeAction string
//
//const (
//	ActionSubscribe   WebSocketMessageEnvelopeAction = "subscribe"
//	ActionUnsubscribe WebSocketMessageEnvelopeAction = "unsubscribe"
//	ActionError       WebSocketMessageEnvelopeAction = "error"
//)

type EventMessageEnvelope struct {
	Event EventType `json:"event"`
	Id    string    `json:"id"`
	Data  any       `json:"data"`
	//Action WebSocketMessageEnvelopeAction `json:"action,omitempty"`
}
