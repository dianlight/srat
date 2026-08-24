// Package rclone provides a modular integration with rclone for cloud
// storage synchronization (issue #954). It is a lab-gated feature: all HTTP
// handlers that expose it must enforce experimental_lab_mode.
//
// This file defines the RPC seam used to talk to the embedded rclone
// library. The real implementation lives in rpc_librclone.go behind the
// "rclonelib" build tag (requires CGO); rpc_stub.go provides a no-op
// fallback so default static (CGO_ENABLED=0) builds keep working and report
// the feature as unavailable.
package rclone

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

// RcloneRPC is the seam between SRAT and the embedded rclone engine.
// RPC executes a single rclone RC API call (https://rclone.org/rc/):
// method is the API method name (e.g. "operations/list"), req is any value
// JSON-serializable as the request object and out receives the decoded JSON
// response. Non-200 statuses are translated to errors. Both methods are
// exported so test doubles can live outside this package too.
type RcloneRPC interface {
	RPC(ctx context.Context, method string, req any, out any) error
	// Available reports whether the embedded library backend was built in.
	Available() bool
}

// ErrorResponse mirrors the {"error": "..."} body returned by rclone on
// failures.
type ErrorResponse struct {
	Error string `json:"error"`
	Input any    `json:"input,omitempty"`
	Path  string `json:"path,omitempty"`
}

// Transport performs a synchronous JSON-over-JSON RPC round-trip and returns
// the raw response body plus HTTP-style status. It abstracts librclone's
// C entry point so behavior tests can script it.
type Transport func(method string, input string) (string, int)

// CallRaw executes one RC call over the given transport and decodes the
// response into out, translating non-200 statuses (including rclone's
// "error" field) into errors. Build-tag implementations share this helper so
// status/error handling stays identical between lib and stub builds.
func CallRaw(ctx context.Context, transport Transport, method string, req any, out any) error {
	logger := slog.Default()
	rawReq, encErr := json.Marshal(req)
	if encErr != nil {
		return fmt.Errorf("marshal rclone %s request: %w", method, encErr)
	}
	respBody, status := transport(method, string(rawReq))
	if status != 200 {
		var errResp ErrorResponse
		if json.Unmarshal([]byte(respBody), &errResp) == nil && errResp.Error != "" {
			logger.DebugContext(ctx, "rclone RPC error", "method", method, "status", status, "error", errResp.Error)
			return fmt.Errorf("rclone %s: %s", method, errResp.Error)
		}
		return fmt.Errorf("rclone %s failed with status %d", method, status)
	}
	if out == nil || len(respBody) == 0 {
		return nil
	}
	if decErr := json.Unmarshal([]byte(respBody), out); decErr != nil {
		return fmt.Errorf("decode rclone %s response: %w", method, decErr)
	}
	logger.DebugContext(ctx, "rclone RPC ok", "method", method)
	return nil
}

// Call performs an RPC request through the given engine and decodes the
// response into out. Unavailable engines fail fast.
func Call(ctx context.Context, rc RcloneRPC, method string, req any, out any) error {
	if !rc.Available() {
		return fmt.Errorf("rclone library backend not available: rebuild with -tags rclonelib")
	}
	return rc.RPC(ctx, method, req, out)
}
