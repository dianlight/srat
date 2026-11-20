package service

import (
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/tlog"
	"github.com/ovechkin-dm/mockio/v2/mock"
	goerrors "gitlab.com/tozd/go/errors"
)

// recordedLog is a captured tlog record used to assert logging behavior.
type recordedLog struct {
	level slog.Level
	msg   string
	attrs map[string]any
}

// captureTlogByMessage registers callbacks for levels and returns a snapshot of
// records whose message equals msg. tlog callbacks run asynchronously, so
// callers must wait with Eventually before asserting on the snapshot.
func captureTlogByMessage(t *testing.T, msg string, levels ...slog.Level) func() []recordedLog {
	t.Helper()
	var mu sync.Mutex
	var matched []recordedLog
	type registration struct {
		level slog.Level
		id    string
	}
	var registrations []registration
	for _, level := range levels {
		lvl := level
		id := tlog.RegisterCallback(lvl, func(e tlog.LogEvent) {
			if e.Record.Message != msg {
				return
			}
			attrs := make(map[string]any, e.Record.NumAttrs())
			e.Record.Attrs(func(a slog.Attr) bool {
				attrs[a.Key] = a.Value.Any()
				return true
			})
			mu.Lock()
			matched = append(matched, recordedLog{level: lvl, msg: e.Record.Message, attrs: attrs})
			mu.Unlock()
		})
		registrations = append(registrations, registration{level: lvl, id: id})
	}
	t.Cleanup(func() {
		for _, reg := range registrations {
			tlog.UnregisterCallback(reg.level, reg.id)
		}
	})
	return func() []recordedLog {
		mu.Lock()
		defer mu.Unlock()
		return append([]recordedLog(nil), matched...)
	}
}

// TestIsSmartEnabled_QueryFailureRateLimited asserts the log noise fix: a
// failing SMART query (expected on USB bridges without SMART support) is logged
// once at Warn per rate-limit window instead of once per cache refresh.
func (suite *DiskStatsServiceSuite) TestIsSmartEnabled_QueryFailureRateLimited() {
	diskID := "disk-usb-bridge"
	queryErr := goerrors.New("Unknown USB bridge [0x090c:0x1000]: SMART not supported")
	mock.When(suite.smartMock.GetSmartInfo(mock.AnyContext(), mock.Exact(diskID))).
		ThenReturn((*dto.SmartInfo)(nil), queryErr)

	const wantMsg = "SMART query failed, caching as disabled"
	snap := captureTlogByMessage(suite.T(), wantMsg, slog.LevelWarn, slog.LevelDebug)
	tlog.ResetRateLimit()
	suite.T().Cleanup(tlog.ResetRateLimit)

	// Simulate two cache refreshes for the same unsupported bridge.
	suite.False(suite.ds.isSmartEnabled(diskID))
	suite.ds.InvalidateSmartCache(diskID)
	suite.False(suite.ds.isSmartEnabled(diskID))

	suite.Require().Eventually(func() bool {
		return len(snap()) == 1
	}, 2*time.Second, 10*time.Millisecond, "expected one SMART-query-failure log record")
	suite.Never(func() bool {
		return len(snap()) > 1
	}, 200*time.Millisecond, 10*time.Millisecond, "the repeated failure must be silenced within the window")

	got := snap()
	suite.Require().Len(got, 1)
	suite.Equal(slog.LevelWarn, got[0].level, "the first failure must be visible at Warn")
	suite.Equal(queryErr.Error(), got[0].attrs["err"],
		"error message must be kept as a plain string, without a formatted stacktrace group")
}
