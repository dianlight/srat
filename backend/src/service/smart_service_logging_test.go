package service_test

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/smartmontools-sdk/bindings/go/v8"
	"github.com/dianlight/tlog"
	"github.com/ovechkin-dm/mockio/v2/mock"
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

// TestGetHealthStatus_VerifyDisagreementLoggedOncePerDevice asserts the log
// noise fix: the #1196 smartctl-disagreement reconciliation stays visible at
// Warn on first occurrence per device and is demoted to Debug on the routine
// repeats observed in the field (~once per minute for a healthy drive).
func (suite *SmartServiceSuite) TestGetHealthStatus_VerifyDisagreementLoggedOncePerDevice() {
	tempFile, err := os.CreateTemp("", "testdevice")
	suite.Require().NoError(err)
	defer os.Remove(tempFile.Name())

	// Lib-backend shape: no SmartStatus, healthy attribute table.
	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(&smartmontools.SMARTInfo{
			SmartSupport: &smartmontools.SmartSupport{Available: true, Enabled: true},
			AtaSmartData: &smartmontools.AtaSmartData{
				Table: []smartmontools.SmartAttribute{
					{ID: 5, Name: "Reallocated_Sector_Ct", Value: 100, Worst: 100, Thresh: 10},
				},
			},
		}, nil)
	mock.When(suite.smartClient.CheckHealth(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(false, nil)

	ctrl := mock.NewMockController(suite.T())
	verifyMock := mock.Mock[smartmontools.SmartClient](ctrl)
	mock.When(verifyMock.CheckHealth(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(true, nil)

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) { return tempFile.Name(), nil })
	suite.service.MockVerifyClient(verifyMock)

	const wantMsg = "SMART lib CheckHealth disagreed with smartctl; trusting smartctl PASSED"
	snap := captureTlogByMessage(suite.T(), wantMsg, slog.LevelWarn, slog.LevelDebug)

	for range 2 {
		health, hErr := suite.service.GetHealthStatus(context.Background(), tempFile.Name())
		suite.Require().NoError(hErr)
		suite.Require().NotNil(health)
		suite.True(health.Passed)
	}

	var got []recordedLog
	suite.Require().Eventually(func() bool {
		got = snap()
		return len(got) == 2
	}, 2*time.Second, 10*time.Millisecond, "expected exactly two disagreement log records")

	warns, debugs := 0, 0
	for _, r := range got {
		switch r.level {
		case slog.LevelWarn:
			warns++
		case slog.LevelDebug:
			debugs++
		}
	}
	suite.Equal(1, warns, "first disagreement must remain visible at Warn")
	suite.Equal(1, debugs, "repeated disagreement must be demoted to Debug")
}
