//go:build !rclonelib

package rclone

import (
	"context"
	"strings"
	"testing"
)

// TestStubRPCUnavailable pins the behaviour of the no-librclone stub build:
// it reports unavailable, never touches a transport and every RPC fails with
// an actionable message telling the operator to rebuild with -tags rclonelib.
func TestStubRPCUnavailable(t *testing.T) {
	rc := NewRPC()
	if rc.Available() {
		t.Fatal("stub backend must report Available() == false")
	}

	var out map[string]any
	err := rc.RPC(context.Background(), "core/version", map[string]any{}, &out)
	if err == nil || !strings.Contains(err.Error(), "rclonelib") {
		t.Fatalf("stub RPC error = %v, want message mentioning rclonelib", err)
	}

	// Call() guards on availability first and surfaces the same error.
	if err := Call(context.Background(), rc, "core/version", map[string]any{}, &out); err == nil {
		t.Fatal("Call on stub backend must return an error")
	}
}
