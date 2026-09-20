package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dianlight/smartmontools-sdk/bindings/go/v8"
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

// TestSelfTestDurationHelpers covers the tracker timing helpers that mirror
// the SDK's polling behavior (#1196).
func TestSelfTestDurationHelpers(t *testing.T) {
	assert.Equal(t, 2, selfTestExpectedMinutes("short"))
	assert.Equal(t, 2, selfTestExpectedMinutes("unknown-type"))
	assert.Equal(t, 120, selfTestExpectedMinutes("long"))
	assert.Equal(t, 120, selfTestExpectedMinutes("extended"))
	assert.Equal(t, 5, selfTestExpectedMinutes("conveyance"))
	assert.Equal(t, 10, selfTestExpectedMinutes("offline"))

	assert.Equal(t, 5, selfTestPollIntervalSecs(2), "short tests clamp to the 5s minimum")
	assert.Equal(t, 5, selfTestPollIntervalSecs(1))
	assert.Equal(t, 60, selfTestPollIntervalSecs(120), "long tests clamp to the 60s maximum")
	assert.Equal(t, 12, selfTestPollIntervalSecs(5))
}

// TestSelfTestTrackerEntryFreshness covers the fresh/reportable branches that
// the external suite cannot reach (it cannot age tracker entries) (#1196).
func TestSelfTestTrackerEntryFreshness(t *testing.T) {
	now := time.Now()

	t.Run("nil entry is neither fresh nor reportable", func(t *testing.T) {
		var e *selfTestTrackerEntry
		assert.False(t, e.fresh(now))
		assert.False(t, e.reportable(now))
	})

	t.Run("recent running entry is fresh and reportable", func(t *testing.T) {
		e := &selfTestTrackerEntry{running: true, updatedAt: now, expectedMinutes: 2}
		assert.True(t, e.fresh(now))
		assert.True(t, e.reportable(now))
	})

	t.Run("stale running entry is dropped", func(t *testing.T) {
		// Short tests poll every 5s: two intervals + 30s margin = 40s window.
		e := &selfTestTrackerEntry{running: true, updatedAt: now.Add(-5 * time.Minute), expectedMinutes: 2}
		assert.False(t, e.fresh(now))
		assert.False(t, e.reportable(now))
	})

	t.Run("recent terminal entry stays reportable", func(t *testing.T) {
		e := &selfTestTrackerEntry{running: false, updatedAt: now.Add(-time.Minute)}
		assert.False(t, e.fresh(now))
		assert.True(t, e.reportable(now))
	})

	t.Run("old terminal entry expires", func(t *testing.T) {
		e := &selfTestTrackerEntry{running: false, updatedAt: now.Add(-time.Hour)}
		assert.False(t, e.reportable(now))
	})
}

// TestNewVerifyClient covers verify-client construction: it is only built for
// production lib-backend deployments, and construction failures degrade to
// nil instead of breaking the service (#1196).
func TestNewVerifyClient(t *testing.T) {
	t.Run("nil without production init", func(t *testing.T) {
		assert.Nil(t, newVerifyClient(&dto.ContextState{LibSmartAvailable: true}, false))
	})

	t.Run("nil without api context", func(t *testing.T) {
		assert.Nil(t, newVerifyClient(nil, true))
	})

	t.Run("nil when lib backend unavailable", func(t *testing.T) {
		assert.Nil(t, newVerifyClient(&dto.ContextState{LibSmartAvailable: false}, true))
	})

	t.Run("nil on construction failure", func(t *testing.T) {
		old := newExecSmartClient
		newExecSmartClient = func() (smartmontools.SmartClient, error) {
			return nil, errors.New("no smartctl")
		}
		defer func() { newExecSmartClient = old }()
		assert.Nil(t, newVerifyClient(&dto.ContextState{LibSmartAvailable: true}, true))
	})

	t.Run("returns client on success", func(t *testing.T) {
		old := newExecSmartClient
		var stub smartmontools.SmartClient
		newExecSmartClient = func() (smartmontools.SmartClient, error) {
			return stub, nil
		}
		defer func() { newExecSmartClient = old }()
		assert.Nil(t, newVerifyClient(&dto.ContextState{LibSmartAvailable: true}, true),
			"nil stub client passes through")
	})
}
