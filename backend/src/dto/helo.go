package dto

import "fmt"

const (
	HomeAssistantClientMessageTypeHelo = "helo"
	HomeAssistantComponentSRAT         = "srat"
)

// ClientMessageEnvelope contains the routing metadata for inbound WebSocket
// messages sent by Home Assistant clients.
type ClientMessageEnvelope struct {
	Type string `json:"type"`
}

// HeloMessage is the first inbound WebSocket handshake payload sent by the
// Home Assistant custom component after the connection is established.
//
// The payload intentionally uses "helo" instead of "hello" to keep the new
// client-to-server handshake unambiguous from the existing outbound welcome
// event streamed by SRAT.
type HeloMessage struct {
	Type         string   `json:"type"`
	Component    string   `json:"component"`
	Version      string   `json:"version"`
	HAVersion    *string  `json:"ha_version,omitempty"`
	EntryID      *string  `json:"entry_id,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// Validate checks whether the helo payload is usable by the backend handshake
// router.
func (msg HeloMessage) Validate() error {
	if msg.Type != HomeAssistantClientMessageTypeHelo {
		return fmt.Errorf("invalid helo type %q", msg.Type)
	}
	if msg.Component == "" {
		return fmt.Errorf("component is required")
	}
	if msg.Version == "" {
		return fmt.Errorf("version is required")
	}
	return nil
}
