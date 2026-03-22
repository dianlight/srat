package dto_test

import (
	"encoding/json"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeloMessage_JSONRoundTrip(t *testing.T) {
	haVersion := "2026.3.0"
	entryID := "entry-123"

	message := dto.HeloMessage{
		Type:         dto.ClientEventTypes.CLIENTEVENTTYPEHELO.String(),
		Component:    dto.HomeAssistantComponentSRAT,
		Version:      "2026.03.1",
		HAVersion:    &haVersion,
		EntryID:      &entryID,
		Capabilities: []string{"reconnect", "events"},
	}

	data, err := json.Marshal(message)
	require.NoError(t, err)
	assert.JSONEq(t, `{"type":"helo","component":"srat","version":"2026.03.1","ha_version":"2026.3.0","entry_id":"entry-123","capabilities":["reconnect","events"]}`, string(data))

	var decoded dto.HeloMessage
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, message, decoded)
	assert.NoError(t, decoded.Validate())
}

func TestHeloMessage_Validate(t *testing.T) {
	tests := []struct {
		name        string
		message     dto.HeloMessage
		errorString string
	}{
		{
			name: "valid helo message",
			message: dto.HeloMessage{
				Type:      dto.ClientEventTypes.CLIENTEVENTTYPEHELO.String(),
				Component: dto.HomeAssistantComponentSRAT,
				Version:   "2026.03.1",
			},
		},
		{
			name: "rejects hello alias",
			message: dto.HeloMessage{
				Type:      "hello",
				Component: dto.HomeAssistantComponentSRAT,
				Version:   "2026.03.1",
			},
			errorString: `invalid helo type "hello"`,
		},
		{
			name: "requires component",
			message: dto.HeloMessage{
				Type:    dto.ClientEventTypes.CLIENTEVENTTYPEHELO.String(),
				Version: "2026.03.1",
			},
			errorString: "component is required",
		},
		{
			name: "requires version",
			message: dto.HeloMessage{
				Type:      dto.ClientEventTypes.CLIENTEVENTTYPEHELO.String(),
				Component: dto.HomeAssistantComponentSRAT,
			},
			errorString: "version is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.message.Validate()
			if tt.errorString == "" {
				assert.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.EqualError(t, err, tt.errorString)
		})
	}
}

func TestClientMessageEnvelope_UnmarshalHeloHeader(t *testing.T) {
	var envelope dto.ClientMessageEnvelope
	err := json.Unmarshal([]byte(`{"type":"helo","component":"srat","version":"2026.03.1"}`), &envelope)
	require.NoError(t, err)
	assert.Equal(t, dto.ClientEventTypes.CLIENTEVENTTYPEHELO.String(), envelope.Type)
}

func TestClientMessageEnvelope_UnmarshalRepairLifecycleHeader(t *testing.T) {
	var envelope dto.ClientMessageEnvelope
	err := json.Unmarshal([]byte(`{"type":"repair_lifecycle","repair_id":"disk_space_low","status":"created"}`), &envelope)
	require.NoError(t, err)
	assert.Equal(t, dto.ClientEventTypes.CLIENTEVENTTYPEREPAIRLIFECYCLE.String(), envelope.Type)
}
