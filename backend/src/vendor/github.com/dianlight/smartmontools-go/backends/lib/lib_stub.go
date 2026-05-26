//go:build !linux && !darwin

package lib

import (
	"context"
	"errors"
)

// ErrNotSupported is returned by New on platforms where the wrapper library is unavailable.
var ErrNotSupported = errors.New("lib backend is not supported on this platform")

// Option configures a LibBackend.
type Option func(*unsupportedBackend)

// unsupportedBackend is a placeholder that satisfies Backend on unsupported platforms.
type unsupportedBackend struct{}

// LibBackend is an alias for the unsupported placeholder on this platform.
type LibBackend = unsupportedBackend

// WithLibraryPath is a no-op option on unsupported platforms.
func WithLibraryPath(_ string) Option { return func(*unsupportedBackend) {} }

// WithSlogHandler is a no-op option on unsupported platforms.
func WithSlogHandler(_ any) Option { return func(*unsupportedBackend) {} }

// WithTLogHandler is a no-op option on unsupported platforms.
func WithTLogHandler(_ any) Option { return func(*unsupportedBackend) {} }

// WithLogHandler is a no-op option on unsupported platforms.
func WithLogHandler(_ LogAdapter) Option { return func(*unsupportedBackend) {} }

// New always returns ErrNotSupported on unsupported platforms.
func New(_ ...Option) (*LibBackend, error) {
	return nil, ErrNotSupported
}

func (u *unsupportedBackend) Name() string { return "lib" }
func (u *unsupportedBackend) ScanDevices(_ context.Context) ([]Device, error) {
	return nil, ErrNotSupported
}
func (u *unsupportedBackend) GetSMARTInfo(_ context.Context, _ string) (*SMARTInfo, error) {
	return nil, ErrNotSupported
}
func (u *unsupportedBackend) CheckHealth(_ context.Context, _ string) (bool, error) {
	return false, ErrNotSupported
}
func (u *unsupportedBackend) GetDeviceInfo(_ context.Context, _ string) (map[string]any, error) {
	return nil, ErrNotSupported
}
func (u *unsupportedBackend) RunSelfTest(_ context.Context, _ string, _ string) error {
	return ErrNotSupported
}
func (u *unsupportedBackend) GetAvailableSelfTests(_ context.Context, _ string) (*SelfTestInfo, error) {
	return nil, ErrNotSupported
}
func (u *unsupportedBackend) EnableSMART(_ context.Context, _ string) error   { return ErrNotSupported }
func (u *unsupportedBackend) DisableSMART(_ context.Context, _ string) error  { return ErrNotSupported }
func (u *unsupportedBackend) AbortSelfTest(_ context.Context, _ string) error { return ErrNotSupported }
func (u *unsupportedBackend) Close() error                                    { return nil }
