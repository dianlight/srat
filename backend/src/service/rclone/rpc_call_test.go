package rclone

import (
	"context"
	"net/http"
	"testing"
)

// TestCallDelegatesWhenAvailable covers the available-backend path of the
// Call convenience helper: request marshalling plus delegation to CallRaw.
func TestCallDelegatesWhenAvailable(t *testing.T) {
	rc := newFakeRPC(true)
	rc.handlers["core/version"] = func(string) (string, int) {
		return `{"version":"1.67"}`, http.StatusOK
	}

	var out struct {
		Version string `json:"version"`
	}
	if err := Call(context.Background(), rc, "core/version", map[string]any{}, &out); err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if out.Version != "1.67" {
		t.Fatalf("decoded version = %q, want 1.67", out.Version)
	}
	if rc.calls["core/version"] != 1 {
		t.Fatalf("core/version called %d times, want 1", rc.calls["core/version"])
	}
}
