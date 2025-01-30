package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
	"github.com/tj/go-spin"
	gomock "go.uber.org/mock/gomock"
)

func TestHealthCheckHandler(t *testing.T) {
	req, err := http.NewRequestWithContext(testContext, "GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	health := api.NewHealth(testContext, false)
	handler := http.HandlerFunc(health.HealthCheckHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expectedContentType := "application/json"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("handler returned wrong content type: got %v want %v",
			contentType, expectedContentType)
	}

	var response dto.HealthPing
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response body: %v", err)
	}

	if !response.Alive {
		t.Errorf("Expected Alive to be true, got false")
	}
}

func TestHealthEventEmitter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	sharedContex := api.StateFromContext(testContext)
	mockBrocker := NewMockBrokerInterface(ctrl)
	sharedContex.SSEBroker = mockBrocker

	health := api.NewHealth(testContext, false)
	numcal := 0
	startTime := time.Now()
	mockBrocker.EXPECT().BroadcastMessage(gomock.Any()).Do((func(data any) {
		msg := data.(*dto.EventMessageEnvelope)
		assert.NotNil(t, msg.Data)
		if msg.Event != dto.EventHeartbeat {
			t.Errorf("Expected Event to be Heartbeat, got %v", msg.Event)
		}
		numcal++
		return
	})).AnyTimes()
	s := spin.New()
	for health.OutputEventsCount < 3 {
		fmt.Printf("\r  \033[36mcomputing\033[m %s (%fs)", s.Next(), time.Since(startTime).Abs().Seconds())
		time.Sleep(time.Millisecond * 500)
	}
	elapsed := time.Since(startTime)
	assert.Greater(t, elapsed.Seconds()/10, numcal)
	assert.Equal(t, health.OutputEventsCount, numcal)

	//t.Log(health)
}
