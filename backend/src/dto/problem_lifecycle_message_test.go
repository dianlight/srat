package dto_test

import (
	"encoding/json"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProblemLifecycleMessage_Validate(t *testing.T) {
	msg := dto.ProblemLifecycleMessage{
		Type:       dto.ClientEventTypes.CLIENTEVENTTYPEPROBLEMLIFECYCLE.String(),
		ProblemKey: "custom_component_restart_required",
		Status:     dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSCREATED,
	}
	require.NoError(t, msg.Validate())
}

func TestProblemLifecycleMessage_JSONRoundTrip(t *testing.T) {
	message := dto.ProblemLifecycleMessage{
		Type:       dto.ClientEventTypes.CLIENTEVENTTYPEPROBLEMLIFECYCLE.String(),
		ProblemKey: "custom_component_restart_required",
		Status:     dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSERROR,
		Error:      new("boom"),
	}

	encoded, err := json.Marshal(message)
	require.NoError(t, err)

	var decoded dto.ProblemLifecycleMessage
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	assert.Equal(t, message.Type, decoded.Type)
	assert.Equal(t, message.ProblemKey, decoded.ProblemKey)
	assert.Equal(t, message.Status, decoded.Status)
	require.NotNil(t, decoded.Error)
	assert.Equal(t, "boom", *decoded.Error)
}
