package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	sr "github.com/dianlight/srat/service/rclone"
)

// TestFsSpec pins the rclone filesystem specifier format used on each side
// of diff/sync operations: local paths pass through untouched, remote paths
// get the remote name prefix and lose their leading slash.
func TestFsSpec(t *testing.T) {
	svc := new(RcloneService)
	remote := "srat_volume_x"

	tests := []struct {
		name   string
		remote *string
		local  string
		want   string
	}{
		{"nil remote returns local path", nil, "/mnt/data/supervisor", "/mnt/data/supervisor"},
		{"remote prefixes and strips leading slash", &remote, "/mnt/data", "srat_volume_x:mnt/data"},
		{"relative path kept as-is under remote", &remote, "sub/dir", "srat_volume_x:sub/dir"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := svc.fsSpec(tc.remote, tc.local); got != tc.want {
				t.Errorf("fsSpec(%v, %q) = %q, want %q", tc.remote, tc.local, got, tc.want)
			}
		})
	}
}

// TestRaiseSyncProblem_NilProblemServiceNoop ensures a service constructed
// without a problem recorder does not panic when a sync fails.
func TestRaiseSyncProblem_NilProblemServiceNoop(t *testing.T) {
	svc := &RcloneService{ctx: context.Background()}
	svc.raiseSyncProblem("volume", "/mnt/x", errors.New("boom"))
}

// TestHandleOAuthCallback_ExpiredState verifies the oauthStateTTL deadline:
// a state that is still in the pending map but past its expiry must be
// rejected before any database access happens (a zero-value service with no
// DB proves the early return).
func TestHandleOAuthCallback_ExpiredState(t *testing.T) {
	svc := &RcloneService{
		pending: map[string]rclonePendingAuth{
			"stale-state": {
				req:     sr.AuthRequest{TargetKind: "volume", TargetID: "/mnt/x"},
				expires: time.Now().Add(-time.Minute),
			},
		},
	}

	if _, err := svc.HandleOAuthCallback(context.Background(), "code", "stale-state"); err == nil {
		t.Fatal("expected expired state to be rejected")
	} else if got := err.Error(); !strings.Contains(got, "unknown or expired oauth state") {
		t.Fatalf("unexpected error: %s", got)
	}

	svc.pendingMu.Lock()
	_, stillThere := svc.pending["stale-state"]
	svc.pendingMu.Unlock()
	if stillThere {
		t.Fatal("expired entry must be consumed like any other use")
	}
}
