package api_test

import (
	"testing"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dto"
)

// TestHealthHandlers_simple verifies HealthCheckHandler and HealthStatusHandler
// return the embedded HealthPing values when the handler is constructed with
// a pre-populated dto.HealthPing. This avoids DI/lifecycle complexity and
// focuses on the handler behaviour.
func TestHealthHandlers_simple(t *testing.T) {
	h := api.HealthHanler{}
	h.HealthPing = dto.HealthPing{Alive: true}

	ping, err := h.HealthCheckHandler(nil, &struct{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ping.Body.Alive {
		t.Fatalf("expected Alive=true")
	}

	status, err := h.HealthStatusHandler(nil, &struct{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !status.Body {
		t.Fatalf("expected status true")
	}
}
