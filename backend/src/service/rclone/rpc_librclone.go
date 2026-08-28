//go:build rclonelib

package rclone

import (
	"context"

	librclone "github.com/rclone/rclone/librclone/librclone"
)

// librcloneRPC implements RcloneRPC against the embedded rclone library.
// Build with -tags rclonelib (and CGO_ENABLED=1) to enable it.
type librcloneRPC struct{}

var _ RcloneRPC = (*librcloneRPC)(nil)

func NewRPC() RcloneRPC { return &librcloneRPC{} }

func (r *librcloneRPC) Available() bool { return true }

// transport performs the actual RPC round-trip. Initialize is idempotent in
// rclone (guarded internally), so calling it per-call is safe and keeps the
// seam stateless for tests.
func (r *librcloneRPC) transport(method string, input string) (string, int) {
	librclone.Initialize()
	return librclone.RPC(method, input)
}

// Finalize releases rclone resources; wired from service shutdown.
func Finalize() {
	librclone.Finalize()
}

func (r *librcloneRPC) RPC(ctx context.Context, method string, req any, out any) error {
	return CallRaw(ctx, r.transport, method, req, out)
}
