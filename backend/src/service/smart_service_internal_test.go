package service

import (
	"context"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRecordLibSmartBackendOutcome covers the shared capability recorder used by
// the smartlib and !smartlib build variants (Task 3: lib_smart_unavailable_reason).
func TestRecordLibSmartBackendOutcome(t *testing.T) {
	t.Run("nil context is a no-op", func(t *testing.T) {
		require.NotPanics(t, func() {
			recordLibSmartBackendOutcome(nil, true, "")
		})
		require.NotPanics(t, func() {
			recordLibSmartBackendOutcome(nil, false, "boom")
		})
	})

	t.Run("available clears any stale reason", func(t *testing.T) {
		apiCtx := &dto.ContextState{LibSmartAvailable: false, LibSmartUnavailableReason: "stale failure"}
		recordLibSmartBackendOutcome(apiCtx, true, "")
		assert.True(t, apiCtx.LibSmartAvailable)
		assert.Empty(t, apiCtx.LibSmartUnavailableReason)
	})

	t.Run("unavailable records the reason", func(t *testing.T) {
		apiCtx := &dto.ContextState{LibSmartAvailable: true, LibSmartUnavailableReason: ""}
		recordLibSmartBackendOutcome(apiCtx, false, "dlopen failed: cannot open libsmartmon_go.so")
		assert.False(t, apiCtx.LibSmartAvailable)
		assert.Equal(t, "dlopen failed: cannot open libsmartmon_go.so", apiCtx.LibSmartUnavailableReason)
	})

	t.Run("unavailable with empty reason still clears availability", func(t *testing.T) {
		apiCtx := &dto.ContextState{LibSmartAvailable: true, LibSmartUnavailableReason: "old"}
		recordLibSmartBackendOutcome(apiCtx, false, "")
		assert.False(t, apiCtx.LibSmartAvailable)
		assert.Empty(t, apiCtx.LibSmartUnavailableReason)
	})
}

// TestNewSmartServiceWithoutClientRecordsBackendOutcome exercises the production
// nil-Client path: NewSmartService must call initSmartClient and record the
// resulting backend outcome on the runtime context. It is build-tag agnostic:
//   - !smartlib (default CI): records the "static build" reason
//   - smartlib (local dev): libbackend.New fails without libsmartmon_go.so and the
//     dlopen error is recorded as the reason
//
// In both cases LibSmartAvailable must be false and the reason non-empty.
func TestNewSmartServiceWithoutClientRecordsBackendOutcome(t *testing.T) {
	apiCtx := &dto.ContextState{}
	eventBus := events.NewEventBus(context.Background())

	svc := NewSmartService(SmartServiceParams{
		Client:   nil, // force the initSmartClient path
		ApiCtx:   apiCtx,
		EventBus: eventBus,
	})
	require.NotNil(t, svc)
	assert.False(t, apiCtx.LibSmartAvailable, "without a bundled .so the lib backend cannot be available")
	assert.NotEmpty(t, apiCtx.LibSmartUnavailableReason, "a reason for the fallback must be recorded")
}
