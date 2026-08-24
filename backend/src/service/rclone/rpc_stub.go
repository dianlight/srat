//go:build !rclonelib

package rclone

import (
	"context"
	"fmt"
)

// stubRPC is compiled when the "rclonelib" build tag is absent (the default,
// CGO_ENABLED=0 static builds). Every call reports the feature as
// unavailable so the UI can surface a clear message instead of failing
// mysteriously.
type stubRPC struct{}

var _ RcloneRPC = (*stubRPC)(nil)

// NewRPC returns the stub implementation for static builds.
func NewRPC() RcloneRPC { return &stubRPC{} }

func (r *stubRPC) Available() bool { return false }

// Finalize is a no-op without the lib backend.
func Finalize() {}

func (r *stubRPC) RPC(ctx context.Context, method string, req any, out any) error {
	return fmt.Errorf("rclone library backend not built in; rebuild with -tags rclonelib")
}
