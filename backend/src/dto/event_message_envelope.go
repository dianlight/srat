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
	Uid   string    `json:"uid"`
	Data  any       `json:"data"`
	//Action WebSocketMessageEnvelopeAction `json:"action,omitempty"`
}
