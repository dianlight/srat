package dto_test

import (
	"encoding/json"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepairCommandMessage_Validate(t *testing.T) {
	tests := []struct {
		name        string
		message     dto.RepairCommandMessage
		errorString string
	}{
		{
			name: "valid upsert command",
			message: dto.RepairCommandMessage{
				CommandID:      "cmd-1",
				RepairID:       "disk_space_low",
				Action:         dto.RepairCommandActionUpsert,
				TranslationKey: "disk_space_low",
				Severity:       dto.RepairIssueSeverityWarning,
				IsPersistent:   true,
			},
		},
		{
			name: "valid delete command without translation key",
			message: dto.RepairCommandMessage{
				CommandID: "cmd-2",
				RepairID:  "disk_space_low",
				Action:    dto.RepairCommandActionDelete,
			},
		},
		{
			name: "requires command_id",
			message: dto.RepairCommandMessage{
				RepairID:       "disk_space_low",
				Action:         dto.RepairCommandActionUpsert,
				TranslationKey: "disk_space_low",
				Severity:       dto.RepairIssueSeverityWarning,
			},
			errorString: "command_id is required",
		},
		{
			name: "requires repair_id",
			message: dto.RepairCommandMessage{
				CommandID:      "cmd-3",
				Action:         dto.RepairCommandActionUpsert,
				TranslationKey: "disk_space_low",
				Severity:       dto.RepairIssueSeverityWarning,
			},
			errorString: "repair_id is required",
		},
		{
			name: "requires translation key for non-delete actions",
			message: dto.RepairCommandMessage{
				CommandID: "cmd-4",
				RepairID:  "disk_space_low",
				Action:    dto.RepairCommandActionUpsert,
				Severity:  dto.RepairIssueSeverityWarning,
			},
			errorString: "translation_key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.message.Validate()
			if tt.errorString == "" {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.EqualError(t, err, tt.errorString)
		})
	}
}

func TestRepairLifecycleMessage_Validate(t *testing.T) {
	msg := dto.RepairLifecycleMessage{
		Type:      dto.ClientEventTypes.CLIENTEVENTTYPEREPAIRLIFECYCLE.String(),
		CommandID: "cmd-1",
		RepairID:  "disk_space_low",
		Status:    dto.RepairLifecycleStatusCreated,
	}

	require.NoError(t, msg.Validate())
}

func TestRepairLifecycleMessage_JSONRoundTrip(t *testing.T) {
	errText := "failed to create issue"
	message := dto.RepairLifecycleMessage{
		Type:      dto.ClientEventTypes.CLIENTEVENTTYPEREPAIRLIFECYCLE.String(),
		CommandID: "cmd-10",
		RepairID:  "disk_space_low",
		Status:    dto.RepairLifecycleStatusError,
		Error:     &errText,
		Details: map[string]any{
			"attempt": 1,
		},
	}

	data, err := json.Marshal(message)
	require.NoError(t, err)

	var decoded dto.RepairLifecycleMessage
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, message.Type, decoded.Type)
	assert.Equal(t, message.CommandID, decoded.CommandID)
	assert.Equal(t, message.RepairID, decoded.RepairID)
	assert.Equal(t, message.Status, decoded.Status)
	assert.Equal(t, message.Error, decoded.Error)
	assert.EqualValues(t, 1, decoded.Details["attempt"])
	require.NoError(t, decoded.Validate())
}
